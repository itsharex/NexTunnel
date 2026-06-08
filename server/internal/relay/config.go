package relay

import (
	"flag"
	"fmt"
	"time"
)

// Config holds the relay server configuration.
type Config struct {
	BindAddr            string
	ControlPort         int
	QUICPort            int
	AuthToken           string
	RequireAuth         bool // When true, refuse to start without AuthToken
	HeartbeatTimeout    time.Duration
	MaxProxiesPerClient int
	WorkConnTimeout     time.Duration
}

// DefaultConfig returns a Config with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		BindAddr:            "0.0.0.0",
		ControlPort:         7000,
		QUICPort:            7443,
		HeartbeatTimeout:    90 * time.Second,
		MaxProxiesPerClient: 100,
		WorkConnTimeout:     30 * time.Second,
	}
}

// ParseFlags parses CLI flags into a Config.
func ParseFlags(fs *flag.FlagSet) *Config {
	cfg := DefaultConfig()
	fs.StringVar(&cfg.BindAddr, "bind", cfg.BindAddr, "bind address")
	fs.IntVar(&cfg.ControlPort, "control-port", cfg.ControlPort, "control port for client connections")
	fs.StringVar(&cfg.AuthToken, "auth-token", cfg.AuthToken, "shared auth token for relay clients (required for non-local deployments)")
	fs.BoolVar(&cfg.RequireAuth, "require-auth", cfg.RequireAuth, "require auth token (auto-enabled for non-localhost bind)")
	fs.DurationVar(&cfg.HeartbeatTimeout, "heartbeat-timeout", cfg.HeartbeatTimeout, "heartbeat timeout")
	fs.IntVar(&cfg.MaxProxiesPerClient, "max-proxies", cfg.MaxProxiesPerClient, "max proxies per client")
	fs.DurationVar(&cfg.WorkConnTimeout, "work-conn-timeout", cfg.WorkConnTimeout, "timeout waiting for work connection")
	fs.IntVar(&cfg.QUICPort, "quic-port", cfg.QUICPort, "QUIC transport port")
	return cfg
}

// Validate checks the configuration for security and correctness.
// Returns an error if the configuration is invalid.
func (c *Config) Validate() error {
	// Auto-enable RequireAuth for non-localhost deployments
	if !c.RequireAuth && c.BindAddr != "127.0.0.1" && c.BindAddr != "localhost" && c.BindAddr != "::1" {
		c.RequireAuth = true
	}
	if c.RequireAuth && c.AuthToken == "" {
		return fmt.Errorf("security: auth-token is required for non-local bind address %q; use -auth-token or bind to 127.0.0.1 for development", c.BindAddr)
	}
	if c.ControlPort <= 0 || c.ControlPort > 65535 {
		return fmt.Errorf("invalid control port: %d", c.ControlPort)
	}
	return nil
}
