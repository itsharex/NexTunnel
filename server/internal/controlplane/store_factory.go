package controlplane

import "fmt"

// NewStoreFromConfig creates a Store implementation based on the configuration.
// If cfg.StorePath is empty, returns an in-memory MemoryStore (suitable for testing).
// If cfg.StorePath is set, returns a persistent SQLiteStore.
func NewStoreFromConfig(cfg ControlPlaneConfig) (Store, error) {
	if cfg.StorePath == "" {
		cfg.Logger.Info("using in-memory store (no persistence)")
		return NewMemoryStore(), nil
	}
	store, err := NewSQLiteStore(cfg.StorePath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite store %q: %w", cfg.StorePath, err)
	}
	cfg.Logger.Info("using SQLite persistent store", "path", cfg.StorePath)
	return store, nil
}
