package probe

import (
	"log/slog"
	"time"
)

// ProbeConfig configures the link quality prober.
type ProbeConfig struct {
	// Interval is the time between probe rounds.
	Interval time.Duration

	// RTTWindow is the number of EWMA samples for RTT smoothing.
	RTTWindow int

	// LossWindow is the sliding window duration for loss calculation.
	LossWindow time.Duration

	// BWProbeSize is the bytes per bandwidth probe packet.
	BWProbeSize int

	// BWProbeCount is the number of packet pairs per bandwidth estimation.
	BWProbeCount int

	// OverheadBudget is the maximum fraction of bandwidth used for probing.
	OverheadBudget float64

	// Logger is the structured logger.
	Logger *slog.Logger
}

// ProbeOption configures a ProbeConfig.
type ProbeOption func(*ProbeConfig)

// WithInterval sets the probe interval.
func WithInterval(d time.Duration) ProbeOption {
	return func(c *ProbeConfig) { c.Interval = d }
}

// WithLossWindow sets the loss calculation sliding window.
func WithLossWindow(d time.Duration) ProbeOption {
	return func(c *ProbeConfig) { c.LossWindow = d }
}

// WithBWProbeSize sets the bandwidth probe packet size.
func WithBWProbeSize(n int) ProbeOption {
	return func(c *ProbeConfig) { c.BWProbeSize = n }
}

// WithProbeLogger sets the logger.
func WithProbeLogger(l *slog.Logger) ProbeOption {
	return func(c *ProbeConfig) { c.Logger = l }
}

// DefaultProbeConfig returns a ProbeConfig with sensible defaults.
func DefaultProbeConfig() ProbeConfig {
	return ProbeConfig{
		Interval:       1 * time.Second,
		RTTWindow:      32,
		LossWindow:     30 * time.Second,
		BWProbeSize:    65536,
		BWProbeCount:   8,
		OverheadBudget: 0.01,
		Logger:         slog.Default(),
	}
}
