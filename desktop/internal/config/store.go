package config

import (
	"database/sql"
	"fmt"
	"time"
)

// TunnelConfig represents a persisted tunnel configuration.
type TunnelConfig struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	ProxyType  string    `json:"proxy_type"`
	LocalAddr  string    `json:"local_addr"`
	LocalPort  int       `json:"local_port"`
	RemotePort int       `json:"remote_port"`
	ServerAddr string    `json:"server_addr"`
	Status     string    `json:"status"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Store provides CRUD operations for tunnel configurations.
type Store struct {
	db *DB
}

// NewStore creates a new Store backed by the given DB.
func NewStore(db *DB) *Store {
	return &Store{db: db}
}

// Create inserts a new tunnel configuration.
func (s *Store) Create(tc *TunnelConfig) error {
	_, err := s.db.db.Exec(`
		INSERT INTO tunnel_configs (id, name, proxy_type, local_addr, local_port, remote_port, server_addr, status)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		tc.ID, tc.Name, tc.ProxyType, tc.LocalAddr, tc.LocalPort, tc.RemotePort, tc.ServerAddr, tc.Status)
	if err != nil {
		return fmt.Errorf("insert tunnel config: %w", err)
	}
	return nil
}

// Get retrieves a tunnel configuration by ID.
func (s *Store) Get(id string) (*TunnelConfig, error) {
	tc := &TunnelConfig{}
	err := s.db.db.QueryRow(`
		SELECT id, name, proxy_type, local_addr, local_port, remote_port, server_addr, status, created_at, updated_at
		FROM tunnel_configs WHERE id = ?`, id).Scan(
		&tc.ID, &tc.Name, &tc.ProxyType, &tc.LocalAddr, &tc.LocalPort,
		&tc.RemotePort, &tc.ServerAddr, &tc.Status, &tc.CreatedAt, &tc.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get tunnel config: %w", err)
	}
	return tc, nil
}

// GetByName retrieves a tunnel configuration by name.
func (s *Store) GetByName(name string) (*TunnelConfig, error) {
	tc := &TunnelConfig{}
	err := s.db.db.QueryRow(`
		SELECT id, name, proxy_type, local_addr, local_port, remote_port, server_addr, status, created_at, updated_at
		FROM tunnel_configs WHERE name = ?`, name).Scan(
		&tc.ID, &tc.Name, &tc.ProxyType, &tc.LocalAddr, &tc.LocalPort,
		&tc.RemotePort, &tc.ServerAddr, &tc.Status, &tc.CreatedAt, &tc.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get tunnel config by name: %w", err)
	}
	return tc, nil
}

// List returns all tunnel configurations.
func (s *Store) List() ([]*TunnelConfig, error) {
	rows, err := s.db.db.Query(`
		SELECT id, name, proxy_type, local_addr, local_port, remote_port, server_addr, status, created_at, updated_at
		FROM tunnel_configs ORDER BY created_at DESC`)
	if err != nil {
		return nil, fmt.Errorf("list tunnel configs: %w", err)
	}
	defer rows.Close()

	var configs []*TunnelConfig
	for rows.Next() {
		tc := &TunnelConfig{}
		if err := rows.Scan(&tc.ID, &tc.Name, &tc.ProxyType, &tc.LocalAddr, &tc.LocalPort,
			&tc.RemotePort, &tc.ServerAddr, &tc.Status, &tc.CreatedAt, &tc.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan tunnel config: %w", err)
		}
		configs = append(configs, tc)
	}
	return configs, rows.Err()
}

// Update modifies an existing tunnel configuration.
func (s *Store) Update(tc *TunnelConfig) error {
	result, err := s.db.db.Exec(`
		UPDATE tunnel_configs
		SET name = ?, proxy_type = ?, local_addr = ?, local_port = ?, remote_port = ?,
		    server_addr = ?, status = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?`,
		tc.Name, tc.ProxyType, tc.LocalAddr, tc.LocalPort, tc.RemotePort,
		tc.ServerAddr, tc.Status, tc.ID)
	if err != nil {
		return fmt.Errorf("update tunnel config: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("tunnel config not found: %s", tc.ID)
	}
	return nil
}

// UpdateStatus updates only the status field of a tunnel configuration.
func (s *Store) UpdateStatus(id, status string) error {
	_, err := s.db.db.Exec(`
		UPDATE tunnel_configs SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`,
		status, id)
	return err
}

// Delete removes a tunnel configuration by ID.
func (s *Store) Delete(id string) error {
	result, err := s.db.db.Exec("DELETE FROM tunnel_configs WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete tunnel config: %w", err)
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("tunnel config not found: %s", id)
	}
	return nil
}

// Count returns the total number of tunnel configurations.
func (s *Store) Count() (int, error) {
	var count int
	err := s.db.db.QueryRow("SELECT COUNT(*) FROM tunnel_configs").Scan(&count)
	return count, err
}

// GetSetting retrieves an application setting by key.
func (s *Store) GetSetting(key string) (string, error) {
	var value string
	err := s.db.db.QueryRow("SELECT value FROM app_settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

// SetSetting stores an application setting.
func (s *Store) SetSetting(key, value string) error {
	_, err := s.db.db.Exec(`
		INSERT INTO app_settings (key, value) VALUES (?, ?)
		ON CONFLICT(key) DO UPDATE SET value = excluded.value`, key, value)
	return err
}
