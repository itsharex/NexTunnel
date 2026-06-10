package auth

import (
	"path/filepath"
	"testing"

	"github.com/nextunnel/pkg/tlsutil"
)

func TestMTLSConfig_Enabled(t *testing.T) {
	cases := []struct {
		name    string
		cfg     MTLSConfig
		enabled bool
	}{
		{"empty", MTLSConfig{}, false},
		{"partial-ca", MTLSConfig{CACert: "ca.pem"}, false},
		{"partial-cert", MTLSConfig{CACert: "ca.pem", Cert: "cert.pem"}, false},
		{"full", MTLSConfig{CACert: "ca.pem", Cert: "cert.pem", Key: "key.pem"}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.cfg.Enabled() != tc.enabled {
				t.Errorf("Enabled() = %v, want %v", tc.cfg.Enabled(), tc.enabled)
			}
		})
	}
}

func TestMTLSConfig_LoadTLSConfig_Incomplete(t *testing.T) {
	cfg := MTLSConfig{CACert: "ca.pem"}
	_, err := cfg.LoadTLSConfig()
	if err == nil {
		t.Error("expected error for incomplete config")
	}
}

func TestMTLSConfig_LoadTLSConfig_Valid(t *testing.T) {
	dir := t.TempDir()

	caCert, caX509, err := tlsutil.GenerateSelfSignedCA()
	if err != nil {
		t.Fatalf("GenerateSelfSignedCA: %v", err)
	}
	caPath := filepath.Join(dir, "ca.pem")
	if err := tlsutil.WritePEMFiles(caCert, caPath, filepath.Join(dir, "ca-key.pem")); err != nil {
		t.Fatal(err)
	}

	clientCert, err := tlsutil.GenerateSignedCert(caX509, caCert.PrivateKey, "desktop-client", false)
	if err != nil {
		t.Fatal(err)
	}
	certPath := filepath.Join(dir, "client.pem")
	keyPath := filepath.Join(dir, "client-key.pem")
	if err := tlsutil.WritePEMFiles(clientCert, certPath, keyPath); err != nil {
		t.Fatal(err)
	}

	cfg := MTLSConfig{CACert: caPath, Cert: certPath, Key: keyPath}
	tlsCfg, err := cfg.LoadTLSConfig()
	if err != nil {
		t.Fatalf("LoadTLSConfig: %v", err)
	}
	if tlsCfg == nil {
		t.Fatal("tls config is nil")
	}
	if len(tlsCfg.Certificates) == 0 {
		t.Error("no client certificates loaded")
	}
	if tlsCfg.RootCAs == nil {
		t.Error("root CAs not loaded")
	}
}

func TestDefaultMTLSConfig(t *testing.T) {
	cfg := DefaultMTLSConfig()
	if cfg.Enabled() {
		t.Error("default config should not be enabled")
	}
}
