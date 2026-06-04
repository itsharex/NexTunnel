package scheduler

import (
	"log/slog"
	"time"
)

// SchedulerConfig configures the intelligent link scheduler.
type SchedulerConfig struct {
	// SwitchThreshold is the RTT degradation that triggers re-evaluation.
	SwitchThreshold time.Duration

	// SwitchCooldown is the minimum time between automatic switches.
	SwitchCooldown time.Duration

	// MaxPacketLoss is the max acceptable packet loss during a switch.
	MaxPacketLoss int

	// LossThreshold is the loss rate that triggers a switch.
	LossThreshold float64

	// BWThreshold is the bandwidth drop ratio that triggers a switch.
	BWThreshold float64

	// EvalInterval is the periodic re-evaluation interval.
	EvalInterval time.Duration

	// EnableAutoSwitch enables automatic path switching.
	EnableAutoSwitch bool

	// Logger is the structured logger.
	Logger *slog.Logger
}

// SchedulerOption configures a SchedulerConfig.
type SchedulerOption func(*SchedulerConfig)

// WithSwitchCooldown sets the minimum time between switches.
func WithSwitchCooldown(d time.Duration) SchedulerOption {
	return func(c *SchedulerConfig) { c.SwitchCooldown = d }
}

// WithEvalInterval sets the evaluation interval.
func WithEvalInterval(d time.Duration) SchedulerOption {
	return func(c *SchedulerConfig) { c.EvalInterval = d }
}

// WithAutoSwitch enables or disables automatic switching.
func WithAutoSwitch(enabled bool) SchedulerOption {
	return func(c *SchedulerConfig) { c.EnableAutoSwitch = enabled }
}

// WithSchedulerLogger sets the logger.
func WithSchedulerLogger(l *slog.Logger) SchedulerOption {
	return func(c *SchedulerConfig) { c.Logger = l }
}

// DefaultSchedulerConfig returns sensible defaults.
func DefaultSchedulerConfig() SchedulerConfig {
	return SchedulerConfig{
		SwitchThreshold:  200 * time.Millisecond,
		SwitchCooldown:   2 * time.Second,
		MaxPacketLoss:    3,
		LossThreshold:    0.1,
		BWThreshold:      0.3,
		EvalInterval:     1 * time.Second,
		EnableAutoSwitch: true,
		Logger:           slog.Default(),
	}
}
