package anycast

import (
	"log/slog"
	"time"
)

// AnycastConfig configures the anycast router.
type AnycastConfig struct {
	// FailoverTimeout is how quickly to switch when a node becomes unhealthy.
	FailoverTimeout time.Duration

	// CacheTTL is the TTL for DNS resolution cache.
	CacheTTL time.Duration

	// MaxRetries is the number of retries before giving up on a node.
	MaxRetries int

	Logger *slog.Logger
}

// AnycastOption configures an AnycastConfig.
type AnycastOption func(*AnycastConfig)

// WithFailoverTimeout sets the failover timeout.
func WithFailoverTimeout(d time.Duration) AnycastOption {
	return func(c *AnycastConfig) { c.FailoverTimeout = d }
}

// WithCacheTTL sets the DNS cache TTL.
func WithCacheTTL(d time.Duration) AnycastOption {
	return func(c *AnycastConfig) { c.CacheTTL = d }
}

// WithMaxRetries sets max retries.
func WithMaxRetries(n int) AnycastOption {
	return func(c *AnycastConfig) { c.MaxRetries = n }
}

// WithAnycastLogger sets the logger.
func WithAnycastLogger(l *slog.Logger) AnycastOption {
	return func(c *AnycastConfig) { c.Logger = l }
}

// DefaultAnycastConfig returns sensible defaults.
func DefaultAnycastConfig() AnycastConfig {
	return AnycastConfig{
		FailoverTimeout: 3 * time.Second,
		CacheTTL:        60 * time.Second,
		MaxRetries:      3,
		Logger:          slog.Default(),
	}
}
