// Package audit provides structured audit logging for security-relevant operations.
package audit

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"
)

// Action represents the type of audited operation.
type Action string

const (
	ActionCreate Action = "create"
	ActionUpdate Action = "update"
	ActionDelete Action = "delete"
	ActionLogin  Action = "login"
	ActionLogout Action = "logout"
	ActionAccess Action = "access"
	ActionDenied Action = "denied"
)

// Result represents the outcome of an audited operation.
type Result string

const (
	ResultSuccess Result = "success"
	ResultDenied  Result = "denied"
	ResultError   Result = "error"
)

// AuditEvent records a single security-relevant operation.
type AuditEvent struct {
	Timestamp  time.Time         `json:"timestamp"`
	Actor      string            `json:"actor"`                // user ID or node ID
	Action     Action            `json:"action"`               // create, update, delete, login, etc.
	Resource   string            `json:"resource"`             // nodes, acl, keys, users, etc.
	ResourceID string            `json:"resource_id,omitempty"`
	Details    map[string]string `json:"details,omitempty"`
	SourceIP   string            `json:"source_ip,omitempty"`
	Result     Result            `json:"result"`               // success, denied, error
}

// AuditFilter specifies criteria for querying audit events.
type AuditFilter struct {
	Actor     string
	Action    Action
	Resource  string
	StartTime time.Time
	EndTime   time.Time
	Limit     int
}

// AuditLogger defines the interface for recording and querying audit events.
type AuditLogger interface {
	// Log records an audit event. Implementations should be non-blocking or buffered.
	Log(event AuditEvent)
	// Query retrieves audit events matching the filter.
	Query(filter AuditFilter) ([]AuditEvent, error)
	// Close flushes pending writes and releases resources.
	Close() error
}

// NewEvent creates an AuditEvent with the current timestamp.
func NewEvent(actor string, action Action, resource, resourceID string, result Result) AuditEvent {
	return AuditEvent{
		Timestamp:  time.Now().UTC(),
		Actor:      actor,
		Action:     action,
		Resource:   resource,
		ResourceID: resourceID,
		Result:     result,
	}
}

// --- JSON File Audit Logger ---

// JSONFileAuditLogger writes audit events as JSON Lines to a file.
type JSONFileAuditLogger struct {
	mu   sync.Mutex
	file *os.File
	enc  *json.Encoder
}

// NewJSONFileAuditLogger creates a logger that appends JSON Lines to the specified path.
func NewJSONFileAuditLogger(path string) (*JSONFileAuditLogger, error) {
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("open audit log %q: %w", path, err)
	}
	return &JSONFileAuditLogger{
		file: f,
		enc:  json.NewEncoder(f),
	}, nil
}

// Log writes an audit event as a JSON line.
func (l *JSONFileAuditLogger) Log(event AuditEvent) {
	l.mu.Lock()
	defer l.mu.Unlock()
	_ = l.enc.Encode(event)
}

// Query is not supported for JSON file logger; returns nil.
func (l *JSONFileAuditLogger) Query(_ AuditFilter) ([]AuditEvent, error) {
	return nil, fmt.Errorf("query not supported for JSON file audit logger")
}

// Close flushes and closes the file.
func (l *JSONFileAuditLogger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}

// --- SQLite Audit Logger ---

// SQLiteAuditLogger stores audit events in a SQLite database.
type SQLiteAuditLogger struct {
	db *sql.DB
}

// NewSQLiteAuditLogger creates a logger backed by a SQLite database.
// If the db is nil, it creates an in-memory database.
func NewSQLiteAuditLogger(db *sql.DB) (*SQLiteAuditLogger, error) {
	if db == nil {
		var err error
		db, err = sql.Open("sqlite", ":memory:")
		if err != nil {
			return nil, fmt.Errorf("open in-memory sqlite: %w", err)
		}
	}

	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS audit_events (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		timestamp TEXT NOT NULL,
		actor TEXT NOT NULL,
		action TEXT NOT NULL,
		resource TEXT NOT NULL,
		resource_id TEXT DEFAULT '',
		details TEXT DEFAULT '{}',
		source_ip TEXT DEFAULT '',
		result TEXT NOT NULL
	)`)
	if err != nil {
		return nil, fmt.Errorf("create audit_events table: %w", err)
	}

	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_events(actor)`)
	if err != nil {
		return nil, fmt.Errorf("create index: %w", err)
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit_events(resource)`)
	if err != nil {
		return nil, fmt.Errorf("create index: %w", err)
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_events(timestamp)`)
	if err != nil {
		return nil, fmt.Errorf("create index: %w", err)
	}

	return &SQLiteAuditLogger{db: db}, nil
}

// Log inserts an audit event into the database.
func (l *SQLiteAuditLogger) Log(event AuditEvent) {
	detailsJSON, _ := json.Marshal(event.Details)
	_, _ = l.db.Exec(
		`INSERT INTO audit_events (timestamp, actor, action, resource, resource_id, details, source_ip, result) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		event.Timestamp.Format(time.RFC3339Nano),
		event.Actor,
		string(event.Action),
		event.Resource,
		event.ResourceID,
		string(detailsJSON),
		event.SourceIP,
		string(event.Result),
	)
}

// Query retrieves audit events matching the filter.
func (l *SQLiteAuditLogger) Query(filter AuditFilter) ([]AuditEvent, error) {
	query := `SELECT timestamp, actor, action, resource, resource_id, details, source_ip, result FROM audit_events WHERE 1=1`
	args := []interface{}{}

	if filter.Actor != "" {
		query += ` AND actor = ?`
		args = append(args, filter.Actor)
	}
	if filter.Action != "" {
		query += ` AND action = ?`
		args = append(args, string(filter.Action))
	}
	if filter.Resource != "" {
		query += ` AND resource = ?`
		args = append(args, filter.Resource)
	}
	if !filter.StartTime.IsZero() {
		query += ` AND timestamp >= ?`
		args = append(args, filter.StartTime.Format(time.RFC3339Nano))
	}
	if !filter.EndTime.IsZero() {
		query += ` AND timestamp <= ?`
		args = append(args, filter.EndTime.Format(time.RFC3339Nano))
	}

	query += ` ORDER BY timestamp DESC`

	if filter.Limit > 0 {
		query += ` LIMIT ?`
		args = append(args, filter.Limit)
	}

	rows, err := l.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("query audit_events: %w", err)
	}
	defer rows.Close()

	var events []AuditEvent
	for rows.Next() {
		var e AuditEvent
		var ts, action, result, details string
		if err := rows.Scan(&ts, &e.Actor, &action, &e.Resource, &e.ResourceID, &details, &e.SourceIP, &result); err != nil {
			return nil, fmt.Errorf("scan audit event: %w", err)
		}
		e.Timestamp, _ = time.Parse(time.RFC3339Nano, ts)
		e.Action = Action(action)
		e.Result = Result(result)
		_ = json.Unmarshal([]byte(details), &e.Details)
		events = append(events, e)
	}
	return events, rows.Err()
}

// Close closes the underlying database.
func (l *SQLiteAuditLogger) Close() error {
	return l.db.Close()
}

// --- No-op Audit Logger ---

// NopAuditLogger discards all audit events. Useful for testing.
type NopAuditLogger struct{}

func (NopAuditLogger) Log(AuditEvent)                          {}
func (NopAuditLogger) Query(AuditFilter) ([]AuditEvent, error) { return nil, nil }
func (NopAuditLogger) Close() error                            { return nil }
