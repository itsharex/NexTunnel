package migration

import (
	"log/slog"
	"time"
)

// MigrationState tracks the migration progress.
type MigrationState string

const (
	MigrationIdle      MigrationState = "idle"
	MigrationDetecting MigrationState = "detecting"
	MigrationMigrating MigrationState = "migrating"
	MigrationSuccess   MigrationState = "success"
	MigrationFailed    MigrationState = "failed"
)

// NetworkEvent describes a network interface change.
type NetworkEvent struct {
	Type      string // "interface_added", "interface_removed", "address_changed"
	Interface string
	OldAddr   string
	NewAddr   string
	Timestamp time.Time
}

// MigrationConfig configures the connection migrator.
type MigrationConfig struct {
	DetectionInterval time.Duration
	MigrationTimeout  time.Duration
	MaxPacketLoss     int
	Logger            *slog.Logger
}

// MigrationOption configures a MigrationConfig.
type MigrationOption func(*MigrationConfig)

// WithDetectionInterval sets the interface polling interval.
func WithDetectionInterval(d time.Duration) MigrationOption {
	return func(c *MigrationConfig) { c.DetectionInterval = d }
}

// WithMigrationTimeout sets the migration timeout.
func WithMigrationTimeout(d time.Duration) MigrationOption {
	return func(c *MigrationConfig) { c.MigrationTimeout = d }
}

// WithMigrationLogger sets the logger.
func WithMigrationLogger(l *slog.Logger) MigrationOption {
	return func(c *MigrationConfig) { c.Logger = l }
}

// DefaultMigrationConfig returns sensible defaults.
func DefaultMigrationConfig() MigrationConfig {
	return MigrationConfig{
		DetectionInterval: 500 * time.Millisecond,
		MigrationTimeout:  1 * time.Second,
		MaxPacketLoss:     5,
		Logger:            slog.Default(),
	}
}
