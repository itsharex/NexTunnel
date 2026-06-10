package controlplane

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/nextunnel/pkg/tlsutil"
)

func TestAuthMiddleware_MTLS(t *testing.T) {
	// Generate CA and client cert in-memory
	caCert, caX509, err := tlsutil.GenerateSelfSignedCA()
	if err != nil {
		t.Fatalf("GenerateSelfSignedCA: %v", err)
	}

	dir := t.TempDir()
	caPath := filepath.Join(dir, "ca.pem")
	caKeyPath := filepath.Join(dir, "ca-key.pem")
	if err := tlsutil.WritePEMFiles(caCert, caPath, caKeyPath); err != nil {
		t.Fatalf("WritePEMFiles CA: %v", err)
	}

	serverCert, err := tlsutil.GenerateSignedCert(caX509, caCert.PrivateKey, "cp-server", true)
	if err != nil {
		t.Fatalf("GenerateSignedCert server: %v", err)
	}
	serverCertPath := filepath.Join(dir, "server.pem")
	serverKeyPath := filepath.Join(dir, "server-key.pem")
	if err := tlsutil.WritePEMFiles(serverCert, serverCertPath, serverKeyPath); err != nil {
		t.Fatalf("WritePEMFiles server: %v", err)
	}

	clientCert, err := tlsutil.GenerateSignedCert(caX509, caCert.PrivateKey, "node-42", false)
	if err != nil {
		t.Fatalf("GenerateSignedCert client: %v", err)
	}
	clientCertPath := filepath.Join(dir, "client.pem")
	clientKeyPath := filepath.Join(dir, "client-key.pem")
	if err := tlsutil.WritePEMFiles(clientCert, clientCertPath, clientKeyPath); err != nil {
		t.Fatalf("WritePEMFiles client: %v", err)
	}

	// Create server with mTLS config
	cfg := DefaultControlPlaneConfig()
	cfg.TLSEnabled = true
	cfg.TLS = tlsutil.TLSConfig{CACert: caPath, Cert: serverCertPath, Key: serverKeyPath}
	cfg.APIToken = "test-token"

	store := NewMemoryStore()
	srv := NewServer(cfg, store)

	handler := srv.Handler()

	// Create httptest server with TLS
	serverTLS, err := tlsutil.LoadServerTLS(caPath, serverCertPath, serverKeyPath)
	if err != nil {
		t.Fatalf("LoadServerTLS: %v", err)
	}

	ts := httptest.NewUnstartedServer(handler)
	ts.TLS = serverTLS
	ts.StartTLS()
	defer ts.Close()

	// Client with valid cert -> success
	clientTLS, err := tlsutil.LoadClientTLS(caPath, clientCertPath, clientKeyPath)
	if err != nil {
		t.Fatalf("LoadClientTLS: %v", err)
	}
	transport := clientTLS.Clone()
	transport.RootCAs = ts.Client().Transport.(*http.Transport).TLSClientConfig.RootCAs

	httpClient := &http.Client{
		Transport: &http.Transport{TLSClientConfig: transport},
	}

	// GET /healthz should work without auth
	resp, err := httpClient.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("healthz: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("healthz status = %d, want 200", resp.StatusCode)
	}

	// POST /api/v1/nodes with client cert should succeed
	resp, err = httpClient.Post(ts.URL+"/api/v1/nodes", "application/json", nil)
	if err != nil {
		t.Fatalf("POST nodes with cert: %v", err)
	}
	resp.Body.Close()
	// 400 (bad request) is fine - it means auth passed, body parsing failed
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		t.Errorf("auth should pass with valid client cert, got %d", resp.StatusCode)
	}

	// Client without cert but with Bearer token should also work
	noCertClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: ts.Client().Transport.(*http.Transport).TLSClientConfig,
		},
	}
	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/nodes", nil)
	req.Header.Set("Authorization", "Bearer test-token")
	resp, err = noCertClient.Do(req)
	if err != nil {
		t.Fatalf("GET nodes with token: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode == http.StatusUnauthorized {
		t.Errorf("Bearer token auth should work, got %d", resp.StatusCode)
	}
}

func TestAuthMiddleware_BearerFallback(t *testing.T) {
	cfg := DefaultControlPlaneConfig()
	cfg.APIToken = "my-secret"

	store := NewMemoryStore()
	srv := NewServer(cfg, store)
	handler := srv.Handler()

	ts := httptest.NewServer(handler)
	defer ts.Close()

	// No auth -> 401
	resp, err := http.Get(ts.URL + "/api/v1/nodes")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("no auth: status = %d, want 401", resp.StatusCode)
	}

	// Valid Bearer token -> 200
	req, _ := http.NewRequest("GET", ts.URL+"/api/v1/nodes", nil)
	req.Header.Set("Authorization", "Bearer my-secret")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET with token: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("valid token: status = %d, want 200", resp.StatusCode)
	}

	// Wrong token -> 401
	req, _ = http.NewRequest("GET", ts.URL+"/api/v1/nodes", nil)
	req.Header.Set("Authorization", "Bearer wrong-token")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("GET with wrong token: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Errorf("wrong token: status = %d, want 401", resp.StatusCode)
	}

	// Healthz always open
	resp, err = http.Get(ts.URL + "/healthz")
	if err != nil {
		t.Fatalf("healthz: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("healthz: status = %d, want 200", resp.StatusCode)
	}
}

func TestAuthMiddleware_NoAuth(t *testing.T) {
	cfg := DefaultControlPlaneConfig()
	// APIToken is empty

	store := NewMemoryStore()
	srv := NewServer(cfg, store)
	handler := srv.Handler()

	ts := httptest.NewServer(handler)
	defer ts.Close()

	// No auth required when APIToken is empty
	resp, err := http.Get(ts.URL + "/api/v1/nodes")
	if err != nil {
		t.Fatalf("GET: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("no auth mode: status = %d, want 200", resp.StatusCode)
	}
}
