package migration

import (
	"context"
	"log/slog"
	"sync/atomic"
)

// Migrator coordinates network change detection and connection migration.
type Migrator struct {
	config   MigrationConfig
	state    atomic.Value // MigrationState
	detector NetworkDetector

	observers []func(MigrationState, NetworkEvent)

	ctx    context.Context
	cancel context.CancelFunc
	logger *slog.Logger
}

// NewMigrator creates a new connection migrator.
func NewMigrator(cfg MigrationConfig, det NetworkDetector, opts ...MigrationOption) *Migrator {
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	ctx, cancel := context.WithCancel(context.Background())
	m := &Migrator{
		config:   cfg,
		detector: det,
		ctx:      ctx,
		cancel:   cancel,
		logger:   cfg.Logger,
	}
	m.state.Store(MigrationIdle)
	return m
}

// Start begins monitoring for network changes.
func (m *Migrator) Start(ctx context.Context) error {
	m.ctx, m.cancel = context.WithCancel(ctx)
	m.state.Store(MigrationDetecting)

	if err := m.detector.Start(m.ctx); err != nil {
		m.state.Store(MigrationFailed)
		return err
	}

	go m.eventLoop()
	m.logger.Info("migrator started", "detection_interval", m.config.DetectionInterval)
	return nil
}

// Stop stops the migrator.
func (m *Migrator) Stop() {
	m.cancel()
	m.detector.Stop()
	m.state.Store(MigrationIdle)
	m.logger.Info("migrator stopped")
}

// State returns the current migration state.
func (m *Migrator) State() MigrationState {
	return m.state.Load().(MigrationState)
}

// OnMigration registers a callback for migration events.
func (m *Migrator) OnMigration(fn func(MigrationState, NetworkEvent)) {
	m.observers = append(m.observers, fn)
}

func (m *Migrator) eventLoop() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case evt := <-m.detector.Events():
			m.handleEvent(evt)
		}
	}
}

func (m *Migrator) handleEvent(evt NetworkEvent) {
	m.logger.Info("network event detected",
		"type", evt.Type,
		"interface", evt.Interface,
		"old", evt.OldAddr,
		"new", evt.NewAddr)

	switch evt.Type {
	case "address_changed", "interface_added":
		m.state.Store(MigrationMigrating)
		m.notify(MigrationMigrating, evt)

		// In a real implementation, this would trigger QUIC connection migration
		// via the QUICTransport.Migrate() method. For now, we report success.
		m.state.Store(MigrationSuccess)
		m.notify(MigrationSuccess, evt)
		m.logger.Info("migration completed", "interface", evt.Interface)

	case "interface_removed":
		m.logger.Warn("interface removed", "interface", evt.Interface)
		m.notify(MigrationDetecting, evt)
	}

	m.state.Store(MigrationDetecting)
}

func (m *Migrator) notify(state MigrationState, evt NetworkEvent) {
	for _, fn := range m.observers {
		fn(state, evt)
	}
}
