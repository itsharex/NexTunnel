package edge

import (
	"log/slog"
	"time"
)

// EdgeConfig configures the edge node management system.
type EdgeConfig struct {
	// HeartbeatInterval is the interval between health check probes.
	HeartbeatInterval time.Duration

	// HealthCheckTimeout is the timeout for a single health check probe.
	HealthCheckTimeout time.Duration

	// UnhealthyThreshold is the number of consecutive failures before marking a node unhealthy.
	UnhealthyThreshold int

	// HealthyThreshold is the number of consecutive successes to mark a node healthy again.
	HealthyThreshold int

	// DeregisterTimeout is how long a node can be unhealthy before automatic deregistration.
	DeregisterTimeout time.Duration

	// ProbeConcurrency is the max number of concurrent health probes.
	ProbeConcurrency int

	Logger *slog.Logger
}

// EdgeOption configures an EdgeConfig.
type EdgeOption func(*EdgeConfig)

// WithHeartbeatInterval sets the health check interval.
func WithHeartbeatInterval(d time.Duration) EdgeOption {
	return func(c *EdgeConfig) { c.HeartbeatInterval = d }
}

// WithHealthCheckTimeout sets the single probe timeout.
func WithHealthCheckTimeout(d time.Duration) EdgeOption {
	return func(c *EdgeConfig) { c.HealthCheckTimeout = d }
}

// WithUnhealthyThreshold sets consecutive failures before marking unhealthy.
func WithUnhealthyThreshold(n int) EdgeOption {
	return func(c *EdgeConfig) { c.UnhealthyThreshold = n }
}

// WithDeregisterTimeout sets the auto-deregister timeout.
func WithDeregisterTimeout(d time.Duration) EdgeOption {
	return func(c *EdgeConfig) { c.DeregisterTimeout = d }
}

// WithEdgeLogger sets the logger.
func WithEdgeLogger(l *slog.Logger) EdgeOption {
	return func(c *EdgeConfig) { c.Logger = l }
}

// DefaultEdgeConfig returns sensible defaults.
func DefaultEdgeConfig() EdgeConfig {
	return EdgeConfig{
		HeartbeatInterval:  5 * time.Second,
		HealthCheckTimeout: 3 * time.Second,
		UnhealthyThreshold: 3,
		HealthyThreshold:   2,
		DeregisterTimeout:  60 * time.Second,
		ProbeConcurrency:   10,
		Logger:             slog.Default(),
	}
}
