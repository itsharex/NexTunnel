package auth_test

import (
	"testing"
	"time"

	"github.com/nextunnel/desktop/internal/auth"
)

var testSecret = []byte("test-secret-key-for-nexTunnel")

func TestGenerateAndValidate(t *testing.T) {
	token, err := auth.GenerateToken("client-1", testSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	claims, err := auth.ValidateToken(token, testSecret)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if claims.ClientID != "client-1" {
		t.Errorf("client_id = %q, want %q", claims.ClientID, "client-1")
	}
	if claims.ExpiresAt <= claims.IssuedAt {
		t.Error("expires_at should be after issued_at")
	}
}

func TestValidateToken_Expired(t *testing.T) {
	token, err := auth.GenerateToken("client-1", testSecret, -1*time.Hour)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	_, err = auth.ValidateToken(token, testSecret)
	if err != auth.ErrTokenExpired {
		t.Errorf("expected ErrTokenExpired, got %v", err)
	}
}

func TestValidateToken_WrongSecret(t *testing.T) {
	token, err := auth.GenerateToken("client-1", testSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	_, err = auth.ValidateToken(token, []byte("wrong-secret"))
	if err != auth.ErrTokenInvalid {
		t.Errorf("expected ErrTokenInvalid, got %v", err)
	}
}

func TestValidateToken_Malformed(t *testing.T) {
	tests := []string{
		"",
		"nodot",
		"a.b.c",
		"invalid-base64.sig",
	}
	for _, token := range tests {
		_, err := auth.ValidateToken(token, testSecret)
		if err == nil {
			t.Errorf("expected error for token %q", token)
		}
	}
}

func TestRefreshToken(t *testing.T) {
	token, _ := auth.GenerateToken("client-1", testSecret, 1*time.Hour)

	refreshed, err := auth.RefreshToken(token, testSecret, 2*time.Hour)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}

	claims, err := auth.ValidateToken(refreshed, testSecret)
	if err != nil {
		t.Fatalf("validate refreshed: %v", err)
	}
	if claims.ClientID != "client-1" {
		t.Errorf("client_id = %q, want %q", claims.ClientID, "client-1")
	}
}

func TestRefreshToken_Expired(t *testing.T) {
	token, _ := auth.GenerateToken("client-1", testSecret, -1*time.Hour)

	// Refresh should work even for expired tokens (re-issue)
	refreshed, err := auth.RefreshToken(token, testSecret, 1*time.Hour)
	if err != nil {
		t.Fatalf("refresh expired: %v", err)
	}

	claims, err := auth.ValidateToken(refreshed, testSecret)
	if err != nil {
		t.Fatalf("validate refreshed: %v", err)
	}
	if claims.ClientID != "client-1" {
		t.Errorf("client_id = %q", claims.ClientID)
	}
}

func TestIsExpiringSoon(t *testing.T) {
	// Token expires in 1 hour, window is 30 min -> not expiring soon
	token, _ := auth.GenerateToken("client-1", testSecret, 1*time.Hour)
	if auth.IsExpiringSoon(token, testSecret, 30*time.Minute) {
		t.Error("should not be expiring soon")
	}

	// Token expires in 10 min, window is 30 min -> expiring soon
	token2, _ := auth.GenerateToken("client-1", testSecret, 10*time.Minute)
	if !auth.IsExpiringSoon(token2, testSecret, 30*time.Minute) {
		t.Error("should be expiring soon")
	}

	// Invalid token -> expiring soon
	if !auth.IsExpiringSoon("invalid", testSecret, 30*time.Minute) {
		t.Error("invalid token should be expiring soon")
	}
}

func TestTokenUniqueness(t *testing.T) {
	t1, _ := auth.GenerateToken("client-1", testSecret, 1*time.Hour)
	t2, _ := auth.GenerateToken("client-1", testSecret, 1*time.Hour)
	if t1 == t2 {
		t.Error("two tokens should not be identical (nonce should differ)")
	}
}
