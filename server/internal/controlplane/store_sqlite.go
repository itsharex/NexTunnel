package controlplane

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"net"
	"time"

	_ "modernc.org/sqlite"
)

// sqliteSchema defines the database tables for the control plane.
const sqliteSchema = `
CREATE TABLE IF NOT EXISTS nodes (
    node_id      TEXT PRIMARY KEY,
    public_key   TEXT NOT NULL DEFAULT '',
    nat_type     TEXT NOT NULL DEFAULT '',
    region       TEXT NOT NULL DEFAULT '',
    subnet       TEXT NOT NULL DEFAULT '',
    virtual_ip   TEXT NOT NULL DEFAULT '',
    metadata     TEXT NOT NULL DEFAULT '{}',
    connected_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS acl_rules (
    id         TEXT PRIMARY KEY,
    source     TEXT NOT NULL DEFAULT '',
    target     TEXT NOT NULL DEFAULT '',
    action     TEXT NOT NULL DEFAULT 'allow',
    protocol   TEXT NOT NULL DEFAULT '',
    ports      TEXT NOT NULL DEFAULT '[]',
    priority   INTEGER NOT NULL DEFAULT 0,
    expires_at DATETIME,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS key_material (
    node_id     TEXT PRIMARY KEY,
    public_key  TEXT NOT NULL,
    key_version INTEGER NOT NULL DEFAULT 1,
    rotated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at  DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_nodes_region ON nodes(region);
CREATE INDEX IF NOT EXISTS idx_acl_priority ON acl_rules(priority);

CREATE TABLE IF NOT EXISTS ip_allocations (
    node_id TEXT PRIMARY KEY,
    ip      TEXT NOT NULL
);
`

// SQLiteStore is a persistent Store implementation backed by SQLite.
type SQLiteStore struct {
	db   *sql.DB
	path string
}

// NewSQLiteStore opens or creates a SQLite database at the given path.
// If path is empty, uses an in-memory database (useful for testing).
func NewSQLiteStore(path string) (*SQLiteStore, error) {
	dsn := path
	if dsn == "" {
		dsn = ":memory:"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite %q: %w", dsn, err)
	}

	// WAL mode for better concurrency; skip for in-memory DBs
	if path != "" {
		if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
			db.Close()
			return nil, fmt.Errorf("set WAL mode: %w", err)
		}
	}

	// Enable foreign keys
	if _, err := db.Exec("PRAGMA foreign_keys=ON"); err != nil {
		db.Close()
		return nil, fmt.Errorf("enable foreign keys: %w", err)
	}

	s := &SQLiteStore{db: db, path: path}
	if err := s.migrate(); err != nil {
		db.Close()
		return nil, err
	}

	return s, nil
}

func (s *SQLiteStore) migrate() error {
	if _, err := s.db.Exec(sqliteSchema); err != nil {
		return fmt.Errorf("run schema migration: %w", err)
	}
	if err := s.ensureColumn("nodes", "virtual_ip", "TEXT NOT NULL DEFAULT ''"); err != nil {
		return err
	}
	return nil
}

// ensureColumn 为已存在的 SQLite 数据库补齐新列，保证升级过程可重复执行。
func (s *SQLiteStore) ensureColumn(tableName, columnName, definition string) error {
	rows, err := s.db.Query(fmt.Sprintf("PRAGMA table_info(%s)", tableName))
	if err != nil {
		return fmt.Errorf("inspect table %s: %w", tableName, err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, columnType string
		var notNull int
		var defaultValue any
		var primaryKey int
		if err := rows.Scan(&cid, &name, &columnType, &notNull, &defaultValue, &primaryKey); err != nil {
			return fmt.Errorf("scan table info %s: %w", tableName, err)
		}
		if name == columnName {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate table info %s: %w", tableName, err)
	}

	if _, err := s.db.Exec(fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s", tableName, columnName, definition)); err != nil {
		return fmt.Errorf("add column %s.%s: %w", tableName, columnName, err)
	}
	return nil
}

// Close closes the database connection.
func (s *SQLiteStore) Close() error {
	return s.db.Close()
}

// --- Node operations ---

func (s *SQLiteStore) SaveNode(node *NodeInfo) error {
	meta, err := json.Marshal(node.Metadata)
	if err != nil {
		return fmt.Errorf("marshal metadata: %w", err)
	}
	_, err = s.db.Exec(`
		INSERT INTO nodes (node_id, public_key, nat_type, region, subnet, virtual_ip, metadata, connected_at, last_seen)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(node_id) DO UPDATE SET
			public_key   = excluded.public_key,
			nat_type     = excluded.nat_type,
			region       = excluded.region,
			subnet       = excluded.subnet,
			virtual_ip   = excluded.virtual_ip,
			metadata     = excluded.metadata,
			connected_at = excluded.connected_at,
			last_seen    = excluded.last_seen`,
		node.NodeID, node.PublicKey, node.NATType, node.Region, node.Subnet,
		node.VirtualIP, string(meta), node.ConnectedAt, node.LastSeen)
	if err != nil {
		return fmt.Errorf("save node %q: %w", node.NodeID, err)
	}
	return nil
}

func (s *SQLiteStore) GetNode(nodeID string) (*NodeInfo, error) {
	row := s.db.QueryRow(`SELECT node_id, public_key, nat_type, region, subnet, virtual_ip, metadata, connected_at, last_seen FROM nodes WHERE node_id = ?`, nodeID)
	return scanNode(row)
}

func (s *SQLiteStore) ListNodes() ([]*NodeInfo, error) {
	rows, err := s.db.Query(`SELECT node_id, public_key, nat_type, region, subnet, virtual_ip, metadata, connected_at, last_seen FROM nodes ORDER BY connected_at`)
	if err != nil {
		return nil, fmt.Errorf("list nodes: %w", err)
	}
	defer rows.Close()

	var result []*NodeInfo
	for rows.Next() {
		node, err := scanNodeRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, node)
	}
	return result, rows.Err()
}

func (s *SQLiteStore) DeleteNode(nodeID string) error {
	_, err := s.db.Exec(`DELETE FROM nodes WHERE node_id = ?`, nodeID)
	return err
}

// --- ACL operations ---

func (s *SQLiteStore) SaveACLRule(rule *ACLRule) error {
	ports, err := json.Marshal(rule.Ports)
	if err != nil {
		return fmt.Errorf("marshal ports: %w", err)
	}
	var expiresAt *time.Time
	if rule.ExpiresAt != nil {
		expiresAt = rule.ExpiresAt
	}
	_, err = s.db.Exec(`
		INSERT INTO acl_rules (id, source, target, action, protocol, ports, priority, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			source     = excluded.source,
			target     = excluded.target,
			action     = excluded.action,
			protocol   = excluded.protocol,
			ports      = excluded.ports,
			priority   = excluded.priority,
			expires_at = excluded.expires_at`,
		rule.ID, rule.Source, rule.Target, rule.Action, rule.Protocol,
		string(ports), rule.Priority, expiresAt, rule.CreatedAt)
	if err != nil {
		return fmt.Errorf("save acl rule %q: %w", rule.ID, err)
	}
	return nil
}

func (s *SQLiteStore) GetACLRule(ruleID string) (*ACLRule, error) {
	row := s.db.QueryRow(`SELECT id, source, target, action, protocol, ports, priority, expires_at, created_at FROM acl_rules WHERE id = ?`, ruleID)
	return scanACLRule(row)
}

func (s *SQLiteStore) ListACLRules() ([]*ACLRule, error) {
	rows, err := s.db.Query(`SELECT id, source, target, action, protocol, ports, priority, expires_at, created_at FROM acl_rules ORDER BY priority`)
	if err != nil {
		return nil, fmt.Errorf("list acl rules: %w", err)
	}
	defer rows.Close()

	var result []*ACLRule
	for rows.Next() {
		rule, err := scanACLRuleRows(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, rule)
	}
	return result, rows.Err()
}

func (s *SQLiteStore) DeleteACLRule(ruleID string) error {
	_, err := s.db.Exec(`DELETE FROM acl_rules WHERE id = ?`, ruleID)
	return err
}

// --- Key operations ---

func (s *SQLiteStore) SaveKeyMaterial(km *KeyMaterial) error {
	_, err := s.db.Exec(`
		INSERT INTO key_material (node_id, public_key, key_version, rotated_at, expires_at)
		VALUES (?, ?, ?, ?, ?)
		ON CONFLICT(node_id) DO UPDATE SET
			public_key  = excluded.public_key,
			key_version = excluded.key_version,
			rotated_at  = excluded.rotated_at,
			expires_at  = excluded.expires_at`,
		km.NodeID, km.PublicKey, km.KeyVersion, km.RotatedAt, km.ExpiresAt)
	if err != nil {
		return fmt.Errorf("save key material %q: %w", km.NodeID, err)
	}
	return nil
}

func (s *SQLiteStore) GetKeyMaterial(nodeID string) (*KeyMaterial, error) {
	row := s.db.QueryRow(`SELECT node_id, public_key, key_version, rotated_at, expires_at FROM key_material WHERE node_id = ?`, nodeID)
	return scanKeyMaterial(row)
}

// --- Scan helpers ---

type scanner interface {
	Scan(dest ...any) error
}

func scanNode(row scanner) (*NodeInfo, error) {
	var node NodeInfo
	var metaJSON string
	err := row.Scan(&node.NodeID, &node.PublicKey, &node.NATType, &node.Region,
		&node.Subnet, &node.VirtualIP, &metaJSON, &node.ConnectedAt, &node.LastSeen)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("node not found")
		}
		return nil, fmt.Errorf("scan node: %w", err)
	}
	if metaJSON != "" && metaJSON != "{}" {
		if err := json.Unmarshal([]byte(metaJSON), &node.Metadata); err != nil {
			return nil, fmt.Errorf("unmarshal metadata: %w", err)
		}
	}
	if node.Metadata == nil {
		node.Metadata = make(map[string]string)
	}
	return &node, nil
}

func scanNodeRows(rows *sql.Rows) (*NodeInfo, error) {
	return scanNode(rows)
}

func scanACLRule(row scanner) (*ACLRule, error) {
	var rule ACLRule
	var portsJSON string
	var expiresAt *time.Time
	err := row.Scan(&rule.ID, &rule.Source, &rule.Target, &rule.Action,
		&rule.Protocol, &portsJSON, &rule.Priority, &expiresAt, &rule.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("rule not found")
		}
		return nil, fmt.Errorf("scan acl rule: %w", err)
	}
	rule.ExpiresAt = expiresAt
	if portsJSON != "" && portsJSON != "[]" {
		if err := json.Unmarshal([]byte(portsJSON), &rule.Ports); err != nil {
			return nil, fmt.Errorf("unmarshal ports: %w", err)
		}
	}
	return &rule, nil
}

func scanACLRuleRows(rows *sql.Rows) (*ACLRule, error) {
	return scanACLRule(rows)
}

func scanKeyMaterial(row scanner) (*KeyMaterial, error) {
	var km KeyMaterial
	err := row.Scan(&km.NodeID, &km.PublicKey, &km.KeyVersion, &km.RotatedAt, &km.ExpiresAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("key material not found")
		}
		return nil, fmt.Errorf("scan key material: %w", err)
	}
	return &km, nil
}

// --- IP allocation methods ---

func (s *SQLiteStore) SaveIPAllocation(nodeID string, ip net.IP) error {
	_, err := s.db.Exec(
		`INSERT OR REPLACE INTO ip_allocations (node_id, ip) VALUES (?, ?)`,
		nodeID, ip.String(),
	)
	if err != nil {
		return fmt.Errorf("save IP allocation: %w", err)
	}
	return nil
}

func (s *SQLiteStore) GetIPAllocation(nodeID string) (net.IP, error) {
	var ipStr string
	err := s.db.QueryRow(`SELECT ip FROM ip_allocations WHERE node_id = ?`, nodeID).Scan(&ipStr)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("IP allocation not found: %s", nodeID)
		}
		return nil, fmt.Errorf("get IP allocation: %w", err)
	}
	ip := net.ParseIP(ipStr)
	if ip == nil {
		return nil, fmt.Errorf("invalid IP in database: %s", ipStr)
	}
	return ip, nil
}

func (s *SQLiteStore) DeleteIPAllocation(nodeID string) error {
	_, err := s.db.Exec(`DELETE FROM ip_allocations WHERE node_id = ?`, nodeID)
	return err
}

func (s *SQLiteStore) ListIPAllocations() (map[string]net.IP, error) {
	rows, err := s.db.Query(`SELECT node_id, ip FROM ip_allocations`)
	if err != nil {
		return nil, fmt.Errorf("list IP allocations: %w", err)
	}
	defer rows.Close()

	result := make(map[string]net.IP)
	for rows.Next() {
		var nodeID, ipStr string
		if err := rows.Scan(&nodeID, &ipStr); err != nil {
			return nil, fmt.Errorf("scan IP allocation: %w", err)
		}
		ip := net.ParseIP(ipStr)
		if ip != nil {
			result[nodeID] = ip
		}
	}
	return result, rows.Err()
}
