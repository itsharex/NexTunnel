package crypto

import (
	"encoding/base64"
	"testing"
)

func TestGenerateWGKeyPair(t *testing.T) {
	privKey, pubKey, err := GenerateWGKeyPair()
	if err != nil {
		t.Fatalf("GenerateWGKeyPair failed: %v", err)
	}

	// Verify keys are valid base64
	privBytes, err := base64.StdEncoding.DecodeString(privKey)
	if err != nil {
		t.Fatalf("private key is not valid base64: %v", err)
	}
	if len(privBytes) != 32 {
		t.Errorf("private key length: got %d, want 32", len(privBytes))
	}

	pubBytes, err := base64.StdEncoding.DecodeString(pubKey)
	if err != nil {
		t.Fatalf("public key is not valid base64: %v", err)
	}
	if len(pubBytes) != 32 {
		t.Errorf("public key length: got %d, want 32", len(pubBytes))
	}

	// Verify public key can be derived from private key
	derivedPub, err := PublicKeyFromPrivate(privKey)
	if err != nil {
		t.Fatalf("PublicKeyFromPrivate failed: %v", err)
	}
	if derivedPub != pubKey {
		t.Errorf("derived public key %q != generated public key %q", derivedPub, pubKey)
	}
}

func TestGenerateWGKeyPair_Uniqueness(t *testing.T) {
	priv1, pub1, err := GenerateWGKeyPair()
	if err != nil {
		t.Fatalf("first key generation failed: %v", err)
	}
	priv2, pub2, err := GenerateWGKeyPair()
	if err != nil {
		t.Fatalf("second key generation failed: %v", err)
	}

	if priv1 == priv2 {
		t.Error("two generated private keys are identical")
	}
	if pub1 == pub2 {
		t.Error("two generated public keys are identical")
	}
}

func TestGeneratePSK(t *testing.T) {
	psk, err := GeneratePSK()
	if err != nil {
		t.Fatalf("GeneratePSK failed: %v", err)
	}

	pskBytes, err := base64.StdEncoding.DecodeString(psk)
	if err != nil {
		t.Fatalf("PSK is not valid base64: %v", err)
	}
	if len(pskBytes) != 32 {
		t.Errorf("PSK length: got %d, want 32", len(pskBytes))
	}

	// Verify uniqueness
	psk2, err := GeneratePSK()
	if err != nil {
		t.Fatalf("second GeneratePSK failed: %v", err)
	}
	if psk == psk2 {
		t.Error("two generated PSKs are identical")
	}
}

func TestPublicKeyFromPrivate_Invalid(t *testing.T) {
	// Invalid base64
	_, err := PublicKeyFromPrivate("not-base64!")
	if err == nil {
		t.Error("expected error for invalid base64, got nil")
	}

	// Wrong key length
	shortKey := base64.StdEncoding.EncodeToString([]byte("short"))
	_, err = PublicKeyFromPrivate(shortKey)
	if err == nil {
		t.Error("expected error for wrong key length, got nil")
	}
}
