package tunnel

import (
	"context"
	"math"
	"math/rand/v2"
	"time"
)

// BackoffConfig configures the exponential backoff strategy.
type BackoffConfig struct {
	BaseDelay      time.Duration
	MaxDelay       time.Duration
	Multiplier     float64
	JitterFraction float64
}

// DefaultBackoffConfig returns a default backoff configuration.
func DefaultBackoffConfig() BackoffConfig {
	return BackoffConfig{
		BaseDelay:      1 * time.Second,
		MaxDelay:       60 * time.Second,
		Multiplier:     2.0,
		JitterFraction: 0.3,
	}
}

// Backoff tracks the current retry attempt and computes delays.
type Backoff struct {
	config  BackoffConfig
	attempt int
}

// NewBackoff creates a new Backoff with the given config.
func NewBackoff(cfg BackoffConfig) *Backoff {
	return &Backoff{config: cfg}
}

// NextDelay computes the next backoff delay with jitter.
func (b *Backoff) NextDelay() time.Duration {
	delay := float64(b.config.BaseDelay) * math.Pow(b.config.Multiplier, float64(b.attempt))
	if delay > float64(b.config.MaxDelay) {
		delay = float64(b.config.MaxDelay)
	}

	// Add jitter
	jitter := delay * b.config.JitterFraction
	delay = delay + (rand.Float64()*2-1)*jitter

	if delay < 0 {
		delay = float64(b.config.BaseDelay)
	}

	b.attempt++
	return time.Duration(delay)
}

// Reset resets the attempt counter (call on successful reconnect).
func (b *Backoff) Reset() {
	b.attempt = 0
}

// Run executes the reconnect loop. The provided fn should attempt to connect;
// if it returns nil, the backoff resets. If it returns an error, we sleep and retry.
// Exits when ctx is cancelled.
func (b *Backoff) Run(ctx context.Context, fn func() error) error {
	for {
		err := fn()
		if err == nil {
			b.Reset()
			return nil
		}

		delay := b.NextDelay()
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// retry
		}
	}
}
