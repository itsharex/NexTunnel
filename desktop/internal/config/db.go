// Package config implements local configuration persistence using SQLite.
package config

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

const schema = `
CREATE TABLE IF NOT EXISTS tunnel_configs (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL UNIQUE,
    proxy_type  TEXT NOT NULL DEFAULT 'tcp',
    local_addr  TEXT NOT NULL,
    local_port  INTEGER NOT NULL,
    remote_port INTEGER NOT NULL,
    server_addr TEXT NOT NULL DEFAULT '',
    status      TEXT NOT NULL DEFAULT 'stopped',
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS app_settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS favorite_ports (
    id          TEXT PRIMARY KEY,
    name        TEXT NOT NULL,
    category    TEXT NOT NULL,
    port        INTEGER NOT NULL,
    protocol    TEXT NOT NULL DEFAULT 'tcp',
    description TEXT NOT NULL DEFAULT '',
    enabled     INTEGER NOT NULL DEFAULT 1,
    builtin     INTEGER NOT NULL DEFAULT 0,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_favorite_ports_protocol_port
    ON favorite_ports(protocol, port);

CREATE TABLE IF NOT EXISTS activity_logs (
    id            TEXT PRIMARY KEY,
    level         TEXT NOT NULL,
    category      TEXT NOT NULL,
    action        TEXT NOT NULL,
    target_type   TEXT NOT NULL DEFAULT '',
    target_id     TEXT NOT NULL DEFAULT '',
    title         TEXT NOT NULL,
    message       TEXT NOT NULL DEFAULT '',
    metadata_json TEXT NOT NULL DEFAULT '{}',
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_activity_logs_created_at
    ON activity_logs(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_activity_logs_level_category
    ON activity_logs(level, category, created_at DESC);
`

// DB wraps the SQLite database connection.
type DB struct {
	db   *sql.DB
	path string
}

// Open opens or creates a SQLite database at the given path.
// If path is empty, uses the default location (~/.nextunnel/config.db).
func Open(path string) (*DB, error) {
	if path == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("get home dir: %w", err)
		}
		dir := filepath.Join(home, ".nextunnel")
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create config dir: %w", err)
		}
		path = filepath.Join(dir, "config.db")
	}

	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	// Enable WAL mode for better concurrency
	if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}

	d := &DB{db: db, path: path}
	if err := d.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return d, nil
}

// migrate runs database migrations.
func (d *DB) migrate() error {
	if _, err := d.db.Exec(schema); err != nil {
		return fmt.Errorf("run schema migration: %w", err)
	}
	return nil
}

// Close closes the database connection.
func (d *DB) Close() error {
	return d.db.Close()
}

// Path returns the database file path.
func (d *DB) Path() string {
	return d.path
}
