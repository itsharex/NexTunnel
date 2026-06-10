package tlsutil

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/pem"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGenerateSelfSignedCA(t *testing.T) {
	caCert, caX509, err := GenerateSelfSignedCA()
	if err != nil {
		t.Fatalf("GenerateSelfSignedCA: %v", err)
	}
	if caX509 == nil {
		t.Fatal("ca x509 certificate is nil")
	}
	if !caX509.IsCA {
		t.Error("certificate IsCA should be true")
	}
	if caX509.Subject.CommonName != "NexTunnel CA" {
		t.Errorf("CN = %q, want %q", caX509.Subject.CommonName, "NexTunnel CA")
	}
	if len(caCert.Certificate) == 0 {
		t.Error("caCert.Certificate is empty")
	}
	if caCert.PrivateKey == nil {
		t.Error("caCert.PrivateKey is nil")
	}
	// CA should be valid for ~10 years
	if caX509.NotAfter.Before(time.Now().Add(9 * 365 * 24 * time.Hour)) {
		t.Error("CA expiry too short")
	}
}

func TestGenerateSignedCert_Server(t *testing.T) {
	caCert, caX509, err := GenerateSelfSignedCA()
	if err != nil {
		t.Fatalf("GenerateSelfSignedCA: %v", err)
	}

	serverCert, err := GenerateSignedCert(caX509, caCert.PrivateKey, "server-1", true)
	if err != nil {
		t.Fatalf("GenerateSignedCert server: %v", err)
	}
	if serverCert.Leaf == nil {
		t.Fatal("server cert Leaf is nil")
	}
	if serverCert.Leaf.Subject.CommonName != "server-1" {
		t.Errorf("CN = %q, want %q", serverCert.Leaf.Subject.CommonName, "server-1")
	}
	if serverCert.Leaf.IsCA {
		t.Error("server cert should not be CA")
	}
	hasServerAuth := false
	for _, eku := range serverCert.Leaf.ExtKeyUsage {
		if eku == x509.ExtKeyUsageServerAuth {
			hasServerAuth = true
		}
	}
	if !hasServerAuth {
		t.Error("server cert missing ServerAuth EKU")
	}
	if len(serverCert.Leaf.IPAddresses) == 0 {
		t.Error("server cert missing SAN IP addresses")
	}
}

func TestGenerateSignedCert_Client(t *testing.T) {
	caCert, caX509, err := GenerateSelfSignedCA()
	if err != nil {
		t.Fatalf("GenerateSelfSignedCA: %v", err)
	}

	clientCert, err := GenerateSignedCert(caX509, caCert.PrivateKey, "node-42", false)
	if err != nil {
		t.Fatalf("GenerateSignedCert client: %v", err)
	}
	if clientCert.Leaf.Subject.CommonName != "node-42" {
		t.Errorf("CN = %q, want %q", clientCert.Leaf.Subject.CommonName, "node-42")
	}
	hasClientAuth := false
	for _, eku := range clientCert.Leaf.ExtKeyUsage {
		if eku == x509.ExtKeyUsageClientAuth {
			hasClientAuth = true
		}
	}
	if !hasClientAuth {
		t.Error("client cert missing ClientAuth EKU")
	}
}

func TestWritePEMFiles(t *testing.T) {
	dir := t.TempDir()
	certPath := filepath.Join(dir, "cert.pem")
	keyPath := filepath.Join(dir, "key.pem")

	caCert, _, err := GenerateSelfSignedCA()
	if err != nil {
		t.Fatalf("GenerateSelfSignedCA: %v", err)
	}
	if err := WritePEMFiles(caCert, certPath, keyPath); err != nil {
		t.Fatalf("WritePEMFiles: %v", err)
	}

	// Verify cert file is readable
	certPEM, err := os.ReadFile(certPath)
	if err != nil {
		t.Fatalf("read cert: %v", err)
	}
	block, _ := pem.Decode(certPEM)
	if block == nil || block.Type != "CERTIFICATE" {
		t.Fatal("cert PEM decode failed")
	}

	// Verify key file is readable
	keyPEM, err := os.ReadFile(keyPath)
	if err != nil {
		t.Fatalf("read key: %v", err)
	}
	block, _ = pem.Decode(keyPEM)
	if block == nil || block.Type != "EC PRIVATE KEY" {
		t.Fatal("key PEM decode failed")
	}

	// Verify tls.LoadX509KeyPair works
	_, err = tls.LoadX509KeyPair(certPath, keyPath)
	if err != nil {
		t.Fatalf("LoadX509KeyPair: %v", err)
	}
}

func TestLoadServerTLS_LoadClientTLS_MutualAuth(t *testing.T) {
	dir := t.TempDir()

	// Generate CA
	caCert, caX509, err := GenerateSelfSignedCA()
	if err != nil {
		t.Fatalf("GenerateSelfSignedCA: %v", err)
	}
	caCertPath := filepath.Join(dir, "ca.pem")
	if err := WritePEMFiles(caCert, caCertPath, filepath.Join(dir, "ca-key.pem")); err != nil {
		t.Fatalf("WritePEMFiles CA: %v", err)
	}

	// Generate server cert
	serverCert, err := GenerateSignedCert(caX509, caCert.PrivateKey, "server", true)
	if err != nil {
		t.Fatalf("GenerateSignedCert server: %v", err)
	}
	serverCertPath := filepath.Join(dir, "server.pem")
	serverKeyPath := filepath.Join(dir, "server-key.pem")
	if err := WritePEMFiles(serverCert, serverCertPath, serverKeyPath); err != nil {
		t.Fatalf("WritePEMFiles server: %v", err)
	}

	// Generate client cert
	clientCert, err := GenerateSignedCert(caX509, caCert.PrivateKey, "node-1", false)
	if err != nil {
		t.Fatalf("GenerateSignedCert client: %v", err)
	}
	clientCertPath := filepath.Join(dir, "client.pem")
	clientKeyPath := filepath.Join(dir, "client-key.pem")
	if err := WritePEMFiles(clientCert, clientCertPath, clientKeyPath); err != nil {
		t.Fatalf("WritePEMFiles client: %v", err)
	}

	// Load server TLS config
	serverTLS, err := LoadServerTLS(caCertPath, serverCertPath, serverKeyPath)
	if err != nil {
		t.Fatalf("LoadServerTLS: %v", err)
	}
	if serverTLS.ClientAuth != tls.VerifyClientCertIfGiven {
		t.Error("server should verify client cert if given")
	}

	// Load client TLS config
	clientTLS, err := LoadClientTLS(caCertPath, clientCertPath, clientKeyPath)
	if err != nil {
		t.Fatalf("LoadClientTLS: %v", err)
	}

	// Test mTLS handshake via httptest
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.TLS == nil || len(r.TLS.PeerCertificates) == 0 {
			t.Error("no peer certificate in request")
			http.Error(w, "no cert", http.StatusForbidden)
			return
		}
		cn := r.TLS.PeerCertificates[0].Subject.CommonName
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello " + cn))
	})

	ts := httptest.NewUnstartedServer(handler)
	ts.TLS = serverTLS
	ts.StartTLS()
	defer ts.Close()

	// Client with valid cert
	transport := clientTLS.Clone()
	transport.RootCAs = ts.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: transport,
		},
	}
	resp, err := client.Get(ts.URL)
	if err != nil {
		t.Fatalf("mTLS request: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("status = %d, want 200", resp.StatusCode)
	}
}

func TestLoadServerTLS_BadPaths(t *testing.T) {
	_, err := LoadServerTLS("/nonexistent/ca.pem", "/nonexistent/cert.pem", "/nonexistent/key.pem")
	if err == nil {
		t.Error("expected error for nonexistent files")
	}
}

func TestLoadClientTLS_BadPaths(t *testing.T) {
	_, err := LoadClientTLS("/nonexistent/ca.pem", "/nonexistent/cert.pem", "/nonexistent/key.pem")
	if err == nil {
		t.Error("expected error for nonexistent files")
	}
}

func TestTLSConfig_Enabled(t *testing.T) {
	c := TLSConfig{}
	if c.Enabled() {
		t.Error("empty config should not be enabled")
	}
	c = TLSConfig{CACert: "ca.pem", Cert: "cert.pem", Key: "key.pem"}
	if !c.Enabled() {
		t.Error("full config should be enabled")
	}
	c = TLSConfig{CACert: "ca.pem"}
	if c.Enabled() {
		t.Error("partial config should not be enabled")
	}
}

func TestMTLS_ClientCertRejection(t *testing.T) {
	dir := t.TempDir()

	caCert, caX509, err := GenerateSelfSignedCA()
	if err != nil {
		t.Fatalf("GenerateSelfSignedCA: %v", err)
	}
	caCertPath := filepath.Join(dir, "ca.pem")
	if err := WritePEMFiles(caCert, caCertPath, filepath.Join(dir, "ca-key.pem")); err != nil {
		t.Fatalf("WritePEMFiles CA: %v", err)
	}

	serverCert, err := GenerateSignedCert(caX509, caCert.PrivateKey, "server", true)
	if err != nil {
		t.Fatalf("GenerateSignedCert server: %v", err)
	}
	serverCertPath := filepath.Join(dir, "server.pem")
	serverKeyPath := filepath.Join(dir, "server-key.pem")
	if err := WritePEMFiles(serverCert, serverCertPath, serverKeyPath); err != nil {
		t.Fatalf("WritePEMFiles server: %v", err)
	}

	serverTLS, err := LoadServerTLS(caCertPath, serverCertPath, serverKeyPath)
	if err != nil {
		t.Fatalf("LoadServerTLS: %v", err)
	}

	// Track whether client cert was presented
	var certPresented bool
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		certPresented = r.TLS != nil && len(r.TLS.PeerCertificates) > 0
		w.WriteHeader(http.StatusOK)
	})
	ts := httptest.NewUnstartedServer(handler)
	ts.TLS = serverTLS
	ts.StartTLS()
	defer ts.Close()

	// Client without cert can connect at TLS level (VerifyClientCertIfGiven)
	// but no peer certificates should be present
	client := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{
				RootCAs:    serverTLS.ClientCAs,
				MinVersion: tls.VersionTLS13,
			},
		},
	}
	resp, err := client.Get(ts.URL)
	if err != nil {
		t.Fatalf("request without cert: %v", err)
	}
	resp.Body.Close()
	if certPresented {
		t.Error("client cert should not be presented when not configured")
	}
}

func TestGenerateSignedCert_DifferentCNs(t *testing.T) {
	caCert, caX509, err := GenerateSelfSignedCA()
	if err != nil {
		t.Fatalf("GenerateSelfSignedCA: %v", err)
	}

	for _, cn := range []string{"node-1", "relay-server", "edge-us-west", ""} {
		cert, err := GenerateSignedCert(caX509, caCert.PrivateKey, cn, false)
		if err != nil {
			t.Fatalf("GenerateSignedCert CN=%q: %v", cn, err)
		}
		if cert.Leaf.Subject.CommonName != cn {
			t.Errorf("CN = %q, want %q", cert.Leaf.Subject.CommonName, cn)
		}
	}
}

// Verify the server cert has localhost SAN for testing
func TestServerCert_SAN(t *testing.T) {
	caCert, caX509, err := GenerateSelfSignedCA()
	if err != nil {
		t.Fatalf("GenerateSelfSignedCA: %v", err)
	}

	cert, err := GenerateSignedCert(caX509, caCert.PrivateKey, "test-server", true)
	if err != nil {
		t.Fatalf("GenerateSignedCert: %v", err)
	}

	foundLocalhost := false
	for _, dns := range cert.Leaf.DNSNames {
		if dns == "localhost" {
			foundLocalhost = true
		}
	}
	if !foundLocalhost {
		t.Error("server cert missing localhost SAN")
	}

	foundLoopback := false
	for _, ip := range cert.Leaf.IPAddresses {
		if ip.Equal(net.ParseIP("127.0.0.1")) {
			foundLoopback = true
		}
	}
	if !foundLoopback {
		t.Error("server cert missing 127.0.0.1 SAN")
	}
}
