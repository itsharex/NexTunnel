package relay

import (
	"flag"
	"time"
)

// Config holds the relay server configuration.
type Config struct {
	BindAddr            string
	ControlPort         int
	QUICPort            int
	AuthToken           string
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
	fs.DurationVar(&cfg.HeartbeatTimeout, "heartbeat-timeout", cfg.HeartbeatTimeout, "heartbeat timeout")
	fs.IntVar(&cfg.MaxProxiesPerClient, "max-proxies", cfg.MaxProxiesPerClient, "max proxies per client")
	fs.DurationVar(&cfg.WorkConnTimeout, "work-conn-timeout", cfg.WorkConnTimeout, "timeout waiting for work connection")
	fs.IntVar(&cfg.QUICPort, "quic-port", cfg.QUICPort, "QUIC transport port")
	return cfg
}
