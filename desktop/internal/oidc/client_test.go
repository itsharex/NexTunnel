package oidc

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	provider := Provider{
		Name:        "Test",
		Issuer:      "https://test.example.com",
		ClientID:    "test-client-id",
		AuthURL:     "https://test.example.com/auth",
		TokenURL:    "https://test.example.com/token",
		UserInfoURL: "https://test.example.com/userinfo",
	}

	client := NewClient(provider, slog.Default())
	if client == nil {
		t.Fatal("expected non-nil client")
	}
	if client.provider.ClientID != "test-client-id" {
		t.Errorf("client ID: got %s, want test-client-id", client.provider.ClientID)
	}
}

func TestNewClientFromWellKnown(t *testing.T) {
	tests := []struct {
		name    string
		wantErr bool
	}{
		{"google", false},
		{"github", false},
		{"unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClientFromWellKnown(tt.name, "cid", "secret", nil)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if client == nil {
				t.Fatal("expected non-nil client")
			}
		})
	}
}

func TestAuthorizationURL(t *testing.T) {
	provider := Provider{
		Name:      "Test",
		ClientID:  "my-client",
		AuthURL:   "https://auth.example.com/authorize",
		TokenURL:  "https://auth.example.com/token",
		Scopes:    []string{"openid", "email"},
		RedirectURL: "http://localhost:8080/callback",
	}

	client := NewClient(provider, nil)
	authURL := client.AuthorizationURL("test-state-123")

	if authURL == "" {
		t.Fatal("expected non-empty auth URL")
	}

	// Verify URL contains required params
	for _, param := range []string{"client_id=my-client", "response_type=code", "state=test-state-123", "scope=openid+email"} {
		if !containsParam(authURL, param) {
			t.Errorf("auth URL missing param: %s\nURL: %s", param, authURL)
		}
	}
}

func TestTokenSet_IsExpired(t *testing.T) {
	expired := &TokenSet{ExpiresAt: time.Now().Add(-1 * time.Hour)}
	if !expired.IsExpired() {
		t.Error("expected expired token")
	}

	valid := &TokenSet{ExpiresAt: time.Now().Add(1 * time.Hour)}
	if valid.IsExpired() {
		t.Error("expected non-expired token")
	}
}

func TestTokenSet_IsExpiringSoon(t *testing.T) {
	soon := &TokenSet{ExpiresAt: time.Now().Add(30 * time.Second)}
	if !soon.IsExpiringSoon(1 * time.Minute) {
		t.Error("expected expiring soon")
	}

	later := &TokenSet{ExpiresAt: time.Now().Add(2 * time.Hour)}
	if later.IsExpiringSoon(1 * time.Minute) {
		t.Error("expected not expiring soon")
	}
}

func TestExchangeCode_MockServer(t *testing.T) {
	// Create a mock token endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			t.Errorf("expected POST, got %s", r.Method)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token":  "mock-access-token",
			"token_type":    "Bearer",
			"refresh_token": "mock-refresh-token",
			"expires_in":    3600,
			"scope":         "openid email",
		})
	}))
	defer server.Close()

	provider := Provider{
		Name:     "Mock",
		ClientID: "test-client",
		TokenURL: server.URL,
		Scopes:   []string{"openid"},
		RedirectURL: "http://localhost:19876/callback",
	}

	client := NewClient(provider, nil)
	tokenSet, err := client.ExchangeCode(context.Background(), "mock-code")
	if err != nil {
		t.Fatalf("ExchangeCode: %v", err)
	}

	if tokenSet.AccessToken != "mock-access-token" {
		t.Errorf("access token: got %s, want mock-access-token", tokenSet.AccessToken)
	}
	if tokenSet.RefreshToken != "mock-refresh-token" {
		t.Errorf("refresh token: got %s, want mock-refresh-token", tokenSet.RefreshToken)
	}
	if tokenSet.IsExpired() {
		t.Error("token should not be expired")
	}

	// Verify current token is stored
	current := client.CurrentToken()
	if current == nil {
		t.Fatal("current token should be stored")
	}
	if current.AccessToken != "mock-access-token" {
		t.Error("stored token mismatch")
	}
}

func TestRefreshToken_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"access_token": "new-access-token",
			"token_type":   "Bearer",
			"expires_in":   3600,
		})
	}))
	defer server.Close()

	provider := Provider{
		Name:     "Mock",
		ClientID: "test-client",
		TokenURL: server.URL,
	}

	client := NewClient(provider, nil)
	tokenSet, err := client.RefreshToken(context.Background(), "old-refresh-token")
	if err != nil {
		t.Fatalf("RefreshToken: %v", err)
	}

	if tokenSet.AccessToken != "new-access-token" {
		t.Errorf("access token: got %s, want new-access-token", tokenSet.AccessToken)
	}
	// Refresh token should be preserved when not returned
	if tokenSet.RefreshToken != "old-refresh-token" {
		t.Errorf("refresh token: got %s, want old-refresh-token", tokenSet.RefreshToken)
	}
}

func TestGetUserInfo_MockServer(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader := r.Header.Get("Authorization")
		if authHeader != "Bearer test-token" {
			t.Errorf("auth header: got %s, want Bearer test-token", authHeader)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"sub":            "user-123",
			"name":           "Test User",
			"email":          "test@example.com",
			"email_verified": true,
		})
	}))
	defer server.Close()

	provider := Provider{
		Name:        "Mock",
		UserInfoURL: server.URL,
	}

	client := NewClient(provider, nil)
	info, err := client.GetUserInfo(context.Background(), "test-token")
	if err != nil {
		t.Fatalf("GetUserInfo: %v", err)
	}

	if info.Subject != "user-123" {
		t.Errorf("subject: got %s, want user-123", info.Subject)
	}
	if info.Email != "test@example.com" {
		t.Errorf("email: got %s, want test@example.com", info.Email)
	}
	if !info.EmailVerified {
		t.Error("email should be verified")
	}

	// Verify stored
	current := client.CurrentUser()
	if current == nil {
		t.Fatal("current user should be stored")
	}
}

func TestExchangeCode_Error(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]any{
			"error":             "invalid_grant",
			"error_description": "Code expired",
		})
	}))
	defer server.Close()

	provider := Provider{
		Name:     "Mock",
		ClientID: "test-client",
		TokenURL: server.URL,
	}

	client := NewClient(provider, nil)
	_, err := client.ExchangeCode(context.Background(), "bad-code")
	if err == nil {
		t.Fatal("expected error for invalid grant")
	}
	if !containsParam(err.Error(), "invalid_grant") {
		t.Errorf("error should mention invalid_grant: %v", err)
	}
}

func TestGenerateState(t *testing.T) {
	state1, err := generateState()
	if err != nil {
		t.Fatalf("generateState: %v", err)
	}
	state2, err := generateState()
	if err != nil {
		t.Fatalf("generateState: %v", err)
	}

	if state1 == "" {
		t.Error("state should not be empty")
	}
	if state1 == state2 {
		t.Error("two states should be unique")
	}
	if len(state1) < 16 {
		t.Error("state should be at least 16 chars")
	}
}

func containsParam(s, param string) bool {
	return len(s) >= len(param) && (s == param || len(s) > 0 && contains(s, param))
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
