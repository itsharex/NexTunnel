package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"

	"golang.org/x/crypto/curve25519"
)

// GenerateWGKeyPair generates a WireGuard-compatible Curve25519 key pair.
// Returns base64-encoded private and public keys.
func GenerateWGKeyPair() (privateKey, publicKey string, err error) {
	var privKey [32]byte
	if _, err := rand.Read(privKey[:]); err != nil {
		return "", "", fmt.Errorf("generate private key: %w", err)
	}

	// Clamp the private key per WireGuard spec (same as curve25519.Clamp)
	privKey[0] &= 248
	privKey[31] &= 127
	privKey[31] |= 64

	pubKeyBytes, err := curve25519.X25519(privKey[:], curve25519.Basepoint)
	if err != nil {
		return "", "", fmt.Errorf("compute public key: %w", err)
	}

	privateKey = base64.StdEncoding.EncodeToString(privKey[:])
	publicKey = base64.StdEncoding.EncodeToString(pubKeyBytes)
	return privateKey, publicKey, nil
}

// GeneratePSK generates a 32-byte pre-shared key for post-quantum resistance.
// Returns a base64-encoded PSK.
func GeneratePSK() (string, error) {
	var psk [32]byte
	if _, err := rand.Read(psk[:]); err != nil {
		return "", fmt.Errorf("generate PSK: %w", err)
	}
	return base64.StdEncoding.EncodeToString(psk[:]), nil
}

// PublicKeyFromPrivate derives the public key from a base64-encoded private key.
func PublicKeyFromPrivate(privateKeyB64 string) (string, error) {
	privKeyBytes, err := base64.StdEncoding.DecodeString(privateKeyB64)
	if err != nil {
		return "", fmt.Errorf("decode private key: %w", err)
	}
	if len(privKeyBytes) != 32 {
		return "", fmt.Errorf("invalid private key length: got %d, want 32", len(privKeyBytes))
	}

	pubKeyBytes, err := curve25519.X25519(privKeyBytes, curve25519.Basepoint)
	if err != nil {
		return "", fmt.Errorf("compute public key: %w", err)
	}
	return base64.StdEncoding.EncodeToString(pubKeyBytes), nil
}
