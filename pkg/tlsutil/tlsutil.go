// Package tlsutil provides TLS certificate management utilities for mTLS authentication.
package tlsutil

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"math/big"
	"net"
	"os"
	"time"
)

// TLSConfig holds file paths for TLS certificate configuration.
type TLSConfig struct {
	CACert             string // Path to CA certificate PEM file
	Cert               string // Path to certificate PEM file
	Key                string // Path to private key PEM file
	InsecureSkipVerify bool   // Skip certificate verification (testing only)
}

// Enabled returns true if TLS certificate paths are configured.
func (c TLSConfig) Enabled() bool {
	return c.CACert != "" && c.Cert != "" && c.Key != ""
}

// LoadServerTLS creates a tls.Config for a server with mTLS support.
// It loads the CA certificate pool and requires client certificate verification.
func LoadServerTLS(caCert, cert, key string) (*tls.Config, error) {
	caPEM, err := os.ReadFile(caCert)
	if err != nil {
		return nil, fmt.Errorf("read CA cert %q: %w", caCert, err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("parse CA cert %q: no valid certificates found", caCert)
	}

	serverCert, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return nil, fmt.Errorf("load server cert/key (%q, %q): %w", cert, key, err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{serverCert},
		ClientCAs:    caPool,
		ClientAuth:   tls.VerifyClientCertIfGiven,
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// LoadClientTLS creates a tls.Config for a client with mTLS support.
// It loads the CA certificate pool and the client certificate for authentication.
func LoadClientTLS(caCert, cert, key string) (*tls.Config, error) {
	caPEM, err := os.ReadFile(caCert)
	if err != nil {
		return nil, fmt.Errorf("read CA cert %q: %w", caCert, err)
	}
	caPool := x509.NewCertPool()
	if !caPool.AppendCertsFromPEM(caPEM) {
		return nil, fmt.Errorf("parse CA cert %q: no valid certificates found", caCert)
	}

	clientCert, err := tls.LoadX509KeyPair(cert, key)
	if err != nil {
		return nil, fmt.Errorf("load client cert/key (%q, %q): %w", cert, key, err)
	}

	return &tls.Config{
		Certificates: []tls.Certificate{clientCert},
		RootCAs:      caPool,
		MinVersion:   tls.VersionTLS13,
	}, nil
}

// GenerateSelfSignedCA generates a self-signed CA certificate and key pair.
// The CA is valid for 10 years.
func GenerateSelfSignedCA() (tls.Certificate, *x509.Certificate, error) {
	caKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("generate CA key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("generate serial: %w", err)
	}

	caTemplate := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"NexTunnel"},
			CommonName:   "NexTunnel CA",
		},
		NotBefore:             time.Now().Add(-1 * time.Hour),
		NotAfter:              time.Now().Add(10 * 365 * 24 * time.Hour),
		KeyUsage:              x509.KeyUsageCertSign | x509.KeyUsageCRLSign,
		BasicConstraintsValid: true,
		IsCA:                  true,
		MaxPathLen:            1,
	}

	caDER, err := x509.CreateCertificate(rand.Reader, caTemplate, caTemplate, &caKey.PublicKey, caKey)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("create CA certificate: %w", err)
	}

	caCertParsed, err := x509.ParseCertificate(caDER)
	if err != nil {
		return tls.Certificate{}, nil, fmt.Errorf("parse CA certificate: %w", err)
	}

	caCertKeyPair := tls.Certificate{
		Certificate: [][]byte{caDER},
		PrivateKey:  caKey,
		Leaf:        caCertParsed,
	}

	return caCertKeyPair, caCertParsed, nil
}

// GenerateSignedCert issues a certificate signed by the given CA.
// If isServer is true, the certificate includes server-auth extended key usage
// and SAN entries for localhost/127.0.0.1.
func GenerateSignedCert(ca *x509.Certificate, caKey interface{}, cn string, isServer bool) (tls.Certificate, error) {
	certKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate cert key: %w", err)
	}

	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate serial: %w", err)
	}

	template := &x509.Certificate{
		SerialNumber: serial,
		Subject: pkix.Name{
			Organization: []string{"NexTunnel"},
			CommonName:   cn,
		},
		NotBefore: time.Now().Add(-1 * time.Hour),
		NotAfter:  time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:  x509.KeyUsageDigitalSignature | x509.KeyUsageKeyEncipherment,
	}

	if isServer {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}
		template.IPAddresses = []net.IP{net.ParseIP("127.0.0.1"), net.ParseIP("::1")}
		template.DNSNames = []string{"localhost"}
	} else {
		template.ExtKeyUsage = []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, ca, &certKey.PublicKey, caKey)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create certificate: %w", err)
	}

	certParsed, err := x509.ParseCertificate(certDER)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("parse certificate: %w", err)
	}

	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  certKey,
		Leaf:        certParsed,
	}, nil
}

// WritePEMFiles writes a TLS certificate and key to PEM files at the given paths.
// This is useful for testing and for bootstrapping certificate infrastructure.
func WritePEMFiles(cert tls.Certificate, certPath, keyPath string) error {
	certFile, err := os.Create(certPath)
	if err != nil {
		return fmt.Errorf("create cert file %q: %w", certPath, err)
	}
	defer certFile.Close()
	for _, der := range cert.Certificate {
		if err := pem.Encode(certFile, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
			return fmt.Errorf("encode cert: %w", err)
		}
	}

	keyFile, err := os.Create(keyPath)
	if err != nil {
		return fmt.Errorf("create key file %q: %w", keyPath, err)
	}
	defer keyFile.Close()

	keyDER, err := x509.MarshalECPrivateKey(cert.PrivateKey.(*ecdsa.PrivateKey))
	if err != nil {
		return fmt.Errorf("marshal private key: %w", err)
	}
	if err := pem.Encode(keyFile, &pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}); err != nil {
		return fmt.Errorf("encode key: %w", err)
	}

	return nil
}
