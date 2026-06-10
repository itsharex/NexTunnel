package auth

import (
	"crypto/tls"
	"fmt"

	"github.com/nextunnel/pkg/tlsutil"
)

// MTLSConfig holds mTLS certificate paths for client connections.
type MTLSConfig struct {
	CACert string // Path to CA certificate PEM
	Cert   string // Path to client certificate PEM
	Key    string // Path to client private key PEM
}

// Enabled returns true if all mTLS certificate paths are configured.
func (c MTLSConfig) Enabled() bool {
	return c.CACert != "" && c.Cert != "" && c.Key != ""
}

// LoadTLSConfig creates a tls.Config for mTLS client connections.
// This is used when connecting to Relay or Control Plane servers with mTLS enabled.
func (c MTLSConfig) LoadTLSConfig() (*tls.Config, error) {
	if !c.Enabled() {
		return nil, fmt.Errorf("mTLS config incomplete: all of CACert, Cert, Key must be set")
	}
	return tlsutil.LoadClientTLS(c.CACert, c.Cert, c.Key)
}

// DefaultMTLSConfig returns an empty MTLSConfig (mTLS disabled).
func DefaultMTLSConfig() MTLSConfig {
	return MTLSConfig{}
}
