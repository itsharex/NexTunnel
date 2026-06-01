package oidc

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Provider represents an OIDC identity provider configuration.
type Provider struct {
	Name         string `json:"name"`
	Issuer       string `json:"issuer"`
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret,omitempty"`
	AuthURL      string `json:"auth_url"`
	TokenURL     string `json:"token_url"`
	UserInfoURL  string `json:"userinfo_url"`
	RedirectURL  string `json:"redirect_url"`
	Scopes       []string `json:"scopes"`
}

// Well-known OIDC providers with their discovery URLs.
var WellKnownProviders = map[string]Provider{
	"google": {
		Name:        "Google",
		Issuer:      "https://accounts.google.com",
		AuthURL:     "https://accounts.google.com/o/oauth2/v2/auth",
		TokenURL:    "https://oauth2.googleapis.com/token",
		UserInfoURL: "https://openidconnect.googleapis.com/v1/userinfo",
		Scopes:      []string{"openid", "email", "profile"},
	},
	"github": {
		Name:        "GitHub",
		Issuer:      "https://github.com",
		AuthURL:     "https://github.com/login/oauth/authorize",
		TokenURL:    "https://github.com/login/oauth/access_token",
		UserInfoURL: "https://api.github.com/user",
		Scopes:      []string{"read:user", "user:email"},
	},
}

// TokenSet holds the OAuth2/OIDC token response.
type TokenSet struct {
	AccessToken  string    `json:"access_token"`
	TokenType    string    `json:"token_type"`
	RefreshToken string    `json:"refresh_token,omitempty"`
	IDToken      string    `json:"id_token,omitempty"`
	ExpiresAt    time.Time `json:"expires_at"`
	Scope        string    `json:"scope,omitempty"`
}

// IsExpired returns true if the access token has expired.
func (t *TokenSet) IsExpired() bool {
	return time.Now().After(t.ExpiresAt)
}

// IsExpiringSoon returns true if the token will expire within the given duration.
func (t *TokenSet) IsExpiringSoon(within time.Duration) bool {
	return time.Now().Add(within).After(t.ExpiresAt)
}

// UserInfo represents the user profile from the IdP.
type UserInfo struct {
	Subject       string `json:"sub"`
	Name          string `json:"name,omitempty"`
	Email         string `json:"email,omitempty"`
	EmailVerified bool   `json:"email_verified,omitempty"`
	Picture       string `json:"picture,omitempty"`
	Provider      string `json:"provider,omitempty"`
}

// DeviceCode holds the device authorization grant response.
type DeviceCode struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

// Client is the OIDC authentication client.
type Client struct {
	provider Provider
	logger   *slog.Logger
	httpClient *http.Client

	mu          sync.RWMutex
	currentToken *TokenSet
	currentUser  *UserInfo
}

// NewClient creates a new OIDC client for the given provider.
func NewClient(provider Provider, logger *slog.Logger) *Client {
	if logger == nil {
		logger = slog.Default()
	}
	if len(provider.Scopes) == 0 {
		provider.Scopes = []string{"openid", "email", "profile"}
	}
	if provider.RedirectURL == "" {
		provider.RedirectURL = "http://localhost:19876/callback"
	}
	return &Client{
		provider:   provider,
		logger:     logger,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// NewClientFromWellKnown creates a client from a well-known provider name.
func NewClientFromWellKnown(name, clientID, clientSecret string, logger *slog.Logger) (*Client, error) {
	provider, ok := WellKnownProviders[name]
	if !ok {
		return nil, fmt.Errorf("unknown provider: %s (available: google, github)", name)
	}
	provider.ClientID = clientID
	provider.ClientSecret = clientSecret
	return NewClient(provider, logger), nil
}

// AuthorizationURL generates the authorization URL for the auth code flow.
func (c *Client) AuthorizationURL(state string) string {
	params := url.Values{
		"client_id":     {c.provider.ClientID},
		"redirect_uri":  {c.provider.RedirectURL},
		"response_type": {"code"},
		"scope":         {strings.Join(c.provider.Scopes, " ")},
		"state":         {state},
	}
	if c.provider.Issuer == "https://accounts.google.com" {
		params.Set("access_type", "offline")
		params.Set("prompt", "consent")
	}
	return c.provider.AuthURL + "?" + params.Encode()
}

// ExchangeCode exchanges an authorization code for tokens.
func (c *Client) ExchangeCode(ctx context.Context, code string) (*TokenSet, error) {
	data := url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {code},
		"redirect_uri":  {c.provider.RedirectURL},
		"client_id":     {c.provider.ClientID},
	}
	if c.provider.ClientSecret != "" {
		data.Set("client_secret", c.provider.ClientSecret)
	}

	tokenSet, err := c.requestToken(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("exchange code: %w", err)
	}

	c.mu.Lock()
	c.currentToken = tokenSet
	c.mu.Unlock()

	return tokenSet, nil
}

// RefreshToken refreshes the access token using a refresh token.
func (c *Client) RefreshToken(ctx context.Context, refreshToken string) (*TokenSet, error) {
	data := url.Values{
		"grant_type":    {"refresh_token"},
		"refresh_token": {refreshToken},
		"client_id":     {c.provider.ClientID},
	}
	if c.provider.ClientSecret != "" {
		data.Set("client_secret", c.provider.ClientSecret)
	}

	tokenSet, err := c.requestToken(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("refresh token: %w", err)
	}

	// Preserve the refresh token if not returned
	if tokenSet.RefreshToken == "" {
		tokenSet.RefreshToken = refreshToken
	}

	c.mu.Lock()
	c.currentToken = tokenSet
	c.mu.Unlock()

	return tokenSet, nil
}

// GetUserInfo fetches the user profile from the IdP.
func (c *Client) GetUserInfo(ctx context.Context, accessToken string) (*UserInfo, error) {
	if c.provider.UserInfoURL == "" {
		return nil, fmt.Errorf("userinfo URL not configured")
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.provider.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch userinfo: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("userinfo request failed: %d %s", resp.StatusCode, string(body))
	}

	var info UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, fmt.Errorf("decode userinfo: %w", err)
	}
	info.Provider = c.provider.Name

	c.mu.Lock()
	c.currentUser = &info
	c.mu.Unlock()

	return &info, nil
}

// RequestDeviceCode initiates the device authorization grant flow.
func (c *Client) RequestDeviceCode(ctx context.Context) (*DeviceCode, error) {
	deviceAuthURL := c.provider.TokenURL
	// For Google, the device code endpoint is different
	if c.provider.Issuer == "https://accounts.google.com" {
		deviceAuthURL = "https://oauth2.googleapis.com/device/code"
	}

	data := url.Values{
		"client_id": {c.provider.ClientID},
		"scope":     {strings.Join(c.provider.Scopes, " ")},
	}

	req, err := http.NewRequestWithContext(ctx, "POST", deviceAuthURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request device code: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("device code request failed: %d %s", resp.StatusCode, string(body))
	}

	var dc DeviceCode
	if err := json.NewDecoder(resp.Body).Decode(&dc); err != nil {
		return nil, fmt.Errorf("decode device code: %w", err)
	}
	if dc.Interval == 0 {
		dc.Interval = 5
	}

	return &dc, nil
}

// PollDeviceToken polls for the device token after user authorization.
func (c *Client) PollDeviceToken(ctx context.Context, deviceCode *DeviceCode) (*TokenSet, error) {
	data := url.Values{
		"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		"device_code": {deviceCode.DeviceCode},
		"client_id":   {c.provider.ClientID},
	}

	interval := time.Duration(deviceCode.Interval) * time.Second
	deadline := time.After(time.Duration(deviceCode.ExpiresIn) * time.Second)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-deadline:
			return nil, fmt.Errorf("device code expired")
		case <-time.After(interval):
		}

		tokenSet, err := c.requestToken(ctx, data)
		if err != nil {
			if strings.Contains(err.Error(), "authorization_pending") {
				continue // User hasn't authorized yet
			}
			if strings.Contains(err.Error(), "slow_down") {
				interval += 5 * time.Second
				continue
			}
			return nil, err
		}

		c.mu.Lock()
		c.currentToken = tokenSet
		c.mu.Unlock()

		return tokenSet, nil
	}
}

// CurrentToken returns the current token set, or nil.
func (c *Client) CurrentToken() *TokenSet {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentToken
}

// CurrentUser returns the current user info, or nil.
func (c *Client) CurrentUser() *UserInfo {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.currentUser
}

// StartCallbackServer starts a local HTTP server to receive the OAuth callback.
// Returns the authorization code and state, or an error.
func (c *Client) StartCallbackServer(ctx context.Context) (code, state string, err error) {
	// Parse the redirect URL to get the listen address
	u, err := url.Parse(c.provider.RedirectURL)
	if err != nil {
		return "", "", fmt.Errorf("parse redirect URL: %w", err)
	}

	listener, err := net.Listen("tcp", u.Host)
	if err != nil {
		return "", "", fmt.Errorf("listen on %s: %w", u.Host, err)
	}
	defer listener.Close()

	codeCh := make(chan string, 1)
	stateCh := make(chan string, 1)
	errCh := make(chan error, 1)

	generatedState, err := generateState()
	if err != nil {
		return "", "", fmt.Errorf("generate state: %w", err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc(u.Path, func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if errMsg := q.Get("error"); errMsg != "" {
			errCh <- fmt.Errorf("auth error: %s - %s", errMsg, q.Get("error_description"))
			fmt.Fprintf(w, "<html><body><h2>Authentication failed</h2><p>%s</p></body></html>", errMsg)
			return
		}

		authCode := q.Get("code")
		authState := q.Get("state")

		if authCode == "" {
			errCh <- fmt.Errorf("no code in callback")
			fmt.Fprintf(w, "<html><body><h2>Error</h2><p>No authorization code received</p></body></html>")
			return
		}

		codeCh <- authCode
		stateCh <- authState
		fmt.Fprintf(w, "<html><body><h2>Authentication successful!</h2><p>You can close this window.</p></body></html>")
	})

	server := &http.Server{Handler: mux}
	go func() {
		if err := server.Serve(listener); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	defer server.Shutdown(context.Background())

	select {
	case <-ctx.Done():
		return "", "", ctx.Err()
	case err := <-errCh:
		return "", "", err
	case authCode := <-codeCh:
		authState := <-stateCh
		if authState != generatedState {
			return "", "", fmt.Errorf("state mismatch: expected %s, got %s", generatedState, authState)
		}
		return authCode, authState, nil
	}
}

// GenerateState returns the generated state parameter for the current auth flow.
func (c *Client) GenerateState() (string, error) {
	return generateState()
}

// --- Internal methods ---

func (c *Client) requestToken(ctx context.Context, data url.Values) (*TokenSet, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", c.provider.TokenURL, strings.NewReader(data.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("token request failed: %d %s", resp.StatusCode, string(body))
	}

	// Parse token response
	var tokenResp struct {
		AccessToken  string `json:"access_token"`
		TokenType    string `json:"token_type"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
		ExpiresIn    int    `json:"expires_in"`
		Scope        string `json:"scope"`
		Error        string `json:"error"`
		ErrorDesc    string `json:"error_description"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("decode token response: %w", err)
	}

	if tokenResp.Error != "" {
		return nil, fmt.Errorf("%s: %s", tokenResp.Error, tokenResp.ErrorDesc)
	}

	expiresAt := time.Now().Add(time.Duration(tokenResp.ExpiresIn) * time.Second)
	if tokenResp.ExpiresIn == 0 {
		expiresAt = time.Now().Add(1 * time.Hour) // default 1 hour
	}

	return &TokenSet{
		AccessToken:  tokenResp.AccessToken,
		TokenType:    tokenResp.TokenType,
		RefreshToken: tokenResp.RefreshToken,
		IDToken:      tokenResp.IDToken,
		ExpiresAt:    expiresAt,
		Scope:        tokenResp.Scope,
	}, nil
}

func generateState() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(b), nil
}
