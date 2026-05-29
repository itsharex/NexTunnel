// Package auth implements token-based authentication for NexTunnel connections.
package auth

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var (
	ErrTokenExpired = errors.New("auth: token expired")
	ErrTokenInvalid = errors.New("auth: token invalid")
	ErrTokenMalformed = errors.New("auth: token malformed")
)

// TokenClaims holds the claims embedded in a token.
type TokenClaims struct {
	ClientID  string `json:"client_id"`
	IssuedAt  int64  `json:"iat"`
	ExpiresAt int64  `json:"exp"`
	Nonce     string `json:"nonce"`
}

// GenerateToken creates a new signed token for the given client ID.
func GenerateToken(clientID string, secret []byte, ttl time.Duration) (string, error) {
	nonce := make([]byte, 16)
	if _, err := rand.Read(nonce); err != nil {
		return "", fmt.Errorf("generate nonce: %w", err)
	}

	now := time.Now()
	claims := TokenClaims{
		ClientID:  clientID,
		IssuedAt:  now.Unix(),
		ExpiresAt: now.Add(ttl).Unix(),
		Nonce:     base64.RawURLEncoding.EncodeToString(nonce),
	}

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal claims: %w", err)
	}

	payloadB64 := base64.RawURLEncoding.EncodeToString(payload)

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payloadB64))
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return payloadB64 + "." + sig, nil
}

// ValidateToken validates a token and returns the embedded claims.
func ValidateToken(token string, secret []byte) (*TokenClaims, error) {
	// Split token into payload.signature
	var payloadB64, sigB64 string
	for i := len(token) - 1; i >= 0; i-- {
		if token[i] == '.' {
			payloadB64 = token[:i]
			sigB64 = token[i+1:]
			break
		}
	}
	if payloadB64 == "" || sigB64 == "" {
		return nil, ErrTokenMalformed
	}

	// Verify signature
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payloadB64))
	expectedSig := mac.Sum(nil)

	sig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, ErrTokenMalformed
	}

	if !hmac.Equal(expectedSig, sig) {
		return nil, ErrTokenInvalid
	}

	// Decode payload
	payload, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, ErrTokenMalformed
	}

	var claims TokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, ErrTokenMalformed
	}

	// Check expiration
	if time.Now().Unix() > claims.ExpiresAt {
		return nil, ErrTokenExpired
	}

	return &claims, nil
}

// RefreshToken creates a new token with extended expiration, preserving the original client ID.
// Unlike ValidateToken, this allows expired tokens to be refreshed (signature is still verified).
func RefreshToken(oldToken string, secret []byte, ttl time.Duration) (string, error) {
	claims, err := decodeTokenVerifySig(oldToken, secret)
	if err != nil {
		return "", err
	}
	return GenerateToken(claims.ClientID, secret, ttl)
}

// decodeTokenVerifySig validates the signature but ignores expiration.
func decodeTokenVerifySig(token string, secret []byte) (*TokenClaims, error) {
	var payloadB64, sigB64 string
	for i := len(token) - 1; i >= 0; i-- {
		if token[i] == '.' {
			payloadB64 = token[:i]
			sigB64 = token[i+1:]
			break
		}
	}
	if payloadB64 == "" || sigB64 == "" {
		return nil, ErrTokenMalformed
	}

	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(payloadB64))
	expectedSig := mac.Sum(nil)

	sig, err := base64.RawURLEncoding.DecodeString(sigB64)
	if err != nil {
		return nil, ErrTokenMalformed
	}
	if !hmac.Equal(expectedSig, sig) {
		return nil, ErrTokenInvalid
	}

	payload, err := base64.RawURLEncoding.DecodeString(payloadB64)
	if err != nil {
		return nil, ErrTokenMalformed
	}

	var claims TokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return nil, ErrTokenMalformed
	}
	return &claims, nil
}

// IsExpiringSoon returns true if the token will expire within the given window.
func IsExpiringSoon(token string, secret []byte, window time.Duration) bool {
	claims, err := ValidateToken(token, secret)
	if err != nil {
		return true
	}
	return time.Until(time.Unix(claims.ExpiresAt, 0)) < window
}
