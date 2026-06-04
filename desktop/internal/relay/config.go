package relay

import (
	"log/slog"
	"time"
)

// RelayClientConfig configures a single relay client connection.
type RelayClientConfig struct {
	ServerAddr string
	ClientID   string
	AuthToken  string
	UseQUIC    bool
	Region     string
	Logger     *slog.Logger
}

// RelayManagerConfig configures the relay manager that handles multiple relay servers.
type RelayManagerConfig struct {
	Relays        []RelayClientConfig
	ProbeInterval time.Duration
	FailoverTime  time.Duration
	GeoSelect     bool
	Logger        *slog.Logger
}

// RelayOption configures a RelayManagerConfig.
type RelayOption func(*RelayManagerConfig)

// WithProbeInterval sets the relay probing interval.
func WithProbeInterval(d time.Duration) RelayOption {
	return func(c *RelayManagerConfig) { c.ProbeInterval = d }
}

// WithFailoverTime sets the max failover time.
func WithFailoverTime(d time.Duration) RelayOption {
	return func(c *RelayManagerConfig) { c.FailoverTime = d }
}

// WithGeoSelect enables geographic-based relay selection.
func WithGeoSelect(enabled bool) RelayOption {
	return func(c *RelayManagerConfig) { c.GeoSelect = enabled }
}

// WithRelayLogger sets the logger.
func WithRelayLogger(l *slog.Logger) RelayOption {
	return func(c *RelayManagerConfig) { c.Logger = l }
}

// DefaultRelayManagerConfig returns sensible defaults.
func DefaultRelayManagerConfig() RelayManagerConfig {
	return RelayManagerConfig{
		ProbeInterval: 5 * time.Second,
		FailoverTime:  2 * time.Second,
		GeoSelect:     true,
		Logger:        slog.Default(),
	}
}
