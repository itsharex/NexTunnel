package dashboard

import (
	"database/sql"
	"encoding/json"
	"fmt"

	_ "modernc.org/sqlite"
)

// DashboardStore defines the persistence interface for the dashboard.
type DashboardStore interface {
	// Node operations
	SaveNode(node *NodeStatus) error
	GetNode(nodeID string) (*NodeStatus, error)
	ListNodes() ([]*NodeStatus, error)
	DeleteNode(nodeID string) error

	// ACL operations
	SaveACL(rule *ACLRuleView) error
	GetACL(ruleID string) (*ACLRuleView, error)
	ListACLs() ([]*ACLRuleView, error)
	DeleteACL(ruleID string) error

	// Alert operations
	SaveAlert(alert *Alert) error
	GetAlert(alertID string) (*Alert, error)
	ListAlerts() ([]*Alert, error)
	AckAlert(alertID string) error
	DeleteAlert(alertID string) error

	// User operations
	SaveUser(user *User) error
	GetUser(username string) (*User, error)
	ListUsers() ([]*User, error)
	DeleteUser(username string) error

	// Close releases database resources.
	Close() error
}

// sqliteSchema defines the database tables for the dashboard.
const sqliteSchema = `
CREATE TABLE IF NOT EXISTS dash_nodes (
    node_id      TEXT PRIMARY KEY,
    region       TEXT NOT NULL DEFAULT '',
    nat_type     TEXT NOT NULL DEFAULT '',
    online       INTEGER NOT NULL DEFAULT 0,
    connected_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    rx_bytes     INTEGER NOT NULL DEFAULT 0,
    tx_bytes     INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS dash_acls (
    id         TEXT PRIMARY KEY,
    source     TEXT NOT NULL DEFAULT '',
    target     TEXT NOT NULL DEFAULT '',
    action     TEXT NOT NULL DEFAULT 'allow',
    protocol   TEXT NOT NULL DEFAULT '',
    priority   INTEGER NOT NULL DEFAULT 0,
    enabled    INTEGER NOT NULL DEFAULT 1,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS dash_alerts (
    id         TEXT PRIMARY KEY,
    level      TEXT NOT NULL DEFAULT 'info',
    message    TEXT NOT NULL DEFAULT '',
    node_id    TEXT NOT NULL DEFAULT '',
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    acked      INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS dash_users (
    username      TEXT PRIMARY KEY,
    id            TEXT NOT NULL,
    role          TEXT NOT NULL DEFAULT 'viewer',
    email         TEXT NOT NULL DEFAULT '',
    password_hash TEXT NOT NULL DEFAULT ''
);

CREATE INDEX IF NOT EXISTS idx_dash_alerts_level ON dash_alerts(level);
CREATE INDEX IF NOT EXISTS idx_dash_alerts_acked ON dash_alerts(acked);
CREATE INDEX IF NOT EXISTS idx_dash_nodes_region ON dash_nodes(region);
`

// SQLiteDashboardStore is a persistent DashboardStore backed by SQLite.
type SQLiteDashboardStore struct {
	db   *sql.DB
	path string
}

// NewSQLiteDashboardStore opens or creates a SQLite database at the given path.
// If path is empty, uses an in-memory database (useful for testing).
func NewSQLiteDashboardStore(path string) (*SQLiteDashboardStore, error) {
	dsn := path
	if dsn == "" {
		dsn = ":memory:"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", dsn, err)
	}

	if path != "" {
		if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
			db.Close()
			return nil, fmt.Errorf("set WAL mode: %w", err)
		}
	}

	s := &SQLiteDashboardStore{db: db, path: path}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func (s *SQLiteDashboardStore) migrate() error {
	if _, err := s.db.Exec(sqliteSchema); err != nil {
		return fmt.Errorf("run schema migration: %w", err)
	}
	return nil
}

func (s *SQLiteDashboardStore) Close() error {
	return s.db.Close()
}

// --- Node operations ---

func (s *SQLiteDashboardStore) SaveNode(node *NodeStatus) error {
	_, err := s.db.Exec(`
		INSERT INTO dash_nodes (node_id, region, nat_type, online, connected_at, last_seen, rx_bytes, tx_bytes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(node_id) DO UPDATE SET
			region = excluded.region, nat_type = excluded.nat_type, online = excluded.online,
			connected_at = excluded.connected_at, last_seen = excluded.last_seen,
			rx_bytes = excluded.rx_bytes, tx_bytes = excluded.tx_bytes`,
		node.NodeID, node.Region, node.NATType, boolToInt(node.Online),
		node.ConnectedAt, node.LastSeen, node.RxBytes, node.TxBytes)
	return err
}

func (s *SQLiteDashboardStore) GetNode(nodeID string) (*NodeStatus, error) {
	row := s.db.QueryRow(`SELECT node_id, region, nat_type, online, connected_at, last_seen, rx_bytes, tx_bytes FROM dash_nodes WHERE node_id = ?`, nodeID)
	return scanNodeStatus(row)
}

func (s *SQLiteDashboardStore) ListNodes() ([]*NodeStatus, error) {
	rows, err := s.db.Query(`SELECT node_id, region, nat_type, online, connected_at, last_seen, rx_bytes, tx_bytes FROM dash_nodes ORDER BY connected_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*NodeStatus
	for rows.Next() {
		n, err := scanNodeStatusRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, n)
	}
	return result, rows.Err()
}

func (s *SQLiteDashboardStore) DeleteNode(nodeID string) error {
	_, err := s.db.Exec(`DELETE FROM dash_nodes WHERE node_id = ?`, nodeID)
	return err
}

// --- ACL operations ---

func (s *SQLiteDashboardStore) SaveACL(rule *ACLRuleView) error {
	_, err := s.db.Exec(`
		INSERT INTO dash_acls (id, source, target, action, protocol, priority, enabled, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			source = excluded.source, target = excluded.target, action = excluded.action,
			protocol = excluded.protocol, priority = excluded.priority, enabled = excluded.enabled`,
		rule.ID, rule.Source, rule.Target, rule.Action, rule.Protocol,
		rule.Priority, boolToInt(rule.Enabled), rule.CreatedAt)
	return err
}

func (s *SQLiteDashboardStore) GetACL(ruleID string) (*ACLRuleView, error) {
	row := s.db.QueryRow(`SELECT id, source, target, action, protocol, priority, enabled, created_at FROM dash_acls WHERE id = ?`, ruleID)
	return scanACLRuleView(row)
}

func (s *SQLiteDashboardStore) ListACLs() ([]*ACLRuleView, error) {
	rows, err := s.db.Query(`SELECT id, source, target, action, protocol, priority, enabled, created_at FROM dash_acls ORDER BY priority`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*ACLRuleView
	for rows.Next() {
		r, err := scanACLRuleViewRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, r)
	}
	return result, rows.Err()
}

func (s *SQLiteDashboardStore) DeleteACL(ruleID string) error {
	_, err := s.db.Exec(`DELETE FROM dash_acls WHERE id = ?`, ruleID)
	return err
}

// --- Alert operations ---

func (s *SQLiteDashboardStore) SaveAlert(alert *Alert) error {
	_, err := s.db.Exec(`
		INSERT INTO dash_alerts (id, level, message, node_id, created_at, acked)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			level = excluded.level, message = excluded.message, node_id = excluded.node_id, acked = excluded.acked`,
		alert.ID, alert.Level, alert.Message, alert.NodeID, alert.CreatedAt, boolToInt(alert.Acked))
	return err
}

func (s *SQLiteDashboardStore) GetAlert(alertID string) (*Alert, error) {
	row := s.db.QueryRow(`SELECT id, level, message, node_id, created_at, acked FROM dash_alerts WHERE id = ?`, alertID)
	return scanAlert(row)
}

func (s *SQLiteDashboardStore) ListAlerts() ([]*Alert, error) {
	rows, err := s.db.Query(`SELECT id, level, message, node_id, created_at, acked FROM dash_alerts ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*Alert
	for rows.Next() {
		a, err := scanAlertRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, a)
	}
	return result, rows.Err()
}

func (s *SQLiteDashboardStore) AckAlert(alertID string) error {
	res, err := s.db.Exec(`UPDATE dash_alerts SET acked = 1 WHERE id = ?`, alertID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return fmt.Errorf("alert not found: %s", alertID)
	}
	return nil
}

func (s *SQLiteDashboardStore) DeleteAlert(alertID string) error {
	_, err := s.db.Exec(`DELETE FROM dash_alerts WHERE id = ?`, alertID)
	return err
}

// --- User operations ---

func (s *SQLiteDashboardStore) SaveUser(user *User) error {
	_, err := s.db.Exec(`
		INSERT INTO dash_users (username, id, role, email, password_hash)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(username) DO UPDATE SET
			id = excluded.id, role = excluded.role, email = excluded.email, password_hash = excluded.password_hash`,
		user.Username, user.ID, user.Role, user.Email, user.PasswordHash)
	return err
}

func (s *SQLiteDashboardStore) GetUser(username string) (*User, error) {
	row := s.db.QueryRow(`SELECT username, id, role, email, password_hash FROM dash_users WHERE username = ?`, username)
	return scanUser(row)
}

func (s *SQLiteDashboardStore) ListUsers() ([]*User, error) {
	rows, err := s.db.Query(`SELECT username, id, role, email, password_hash FROM dash_users ORDER BY username`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*User
	for rows.Next() {
		u, err := scanUserRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, u)
	}
	return result, rows.Err()
}

func (s *SQLiteDashboardStore) DeleteUser(username string) error {
	_, err := s.db.Exec(`DELETE FROM dash_users WHERE username = ?`, username)
	return err
}

// --- Scan helpers ---

type rowScanner interface {
	Scan(dest ...any) error
}

func scanNodeStatus(row rowScanner) (*NodeStatus, error) {
	var n NodeStatus
	var online int
	err := row.Scan(&n.NodeID, &n.Region, &n.NATType, &online, &n.ConnectedAt, &n.LastSeen, &n.RxBytes, &n.TxBytes)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("node not found")
		}
		return nil, err
	}
	n.Online = online != 0
	return &n, nil
}

func scanNodeStatusRows(rows *sql.Rows) (*NodeStatus, error) { return scanNodeStatus(rows) }

func scanACLRuleView(row rowScanner) (*ACLRuleView, error) {
	var r ACLRuleView
	var enabled int
	err := row.Scan(&r.ID, &r.Source, &r.Target, &r.Action, &r.Protocol, &r.Priority, &enabled, &r.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("acl rule not found")
		}
		return nil, err
	}
	r.Enabled = enabled != 0
	return &r, nil
}

func scanACLRuleViewRows(rows *sql.Rows) (*ACLRuleView, error) { return scanACLRuleView(rows) }

func scanAlert(row rowScanner) (*Alert, error) {
	var a Alert
	var acked int
	err := row.Scan(&a.ID, &a.Level, &a.Message, &a.NodeID, &a.CreatedAt, &acked)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("alert not found")
		}
		return nil, err
	}
	a.Acked = acked != 0
	return &a, nil
}

func scanAlertRows(rows *sql.Rows) (*Alert, error) { return scanAlert(rows) }

func scanUser(row rowScanner) (*User, error) {
	var u User
	err := row.Scan(&u.Username, &u.ID, &u.Role, &u.Email, &u.PasswordHash)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("user not found")
		}
		return nil, err
	}
	return &u, nil
}

func scanUserRows(rows *sql.Rows) (*User, error) { return scanUser(rows) }

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}

// Ensure json import is used (for potential future serialization needs)
var _ = json.Marshal
