package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNewEvent(t *testing.T) {
	e := NewEvent("user-1", ActionCreate, "nodes", "node-42", ResultSuccess)
	if e.Actor != "user-1" {
		t.Errorf("Actor = %q, want %q", e.Actor, "user-1")
	}
	if e.Action != ActionCreate {
		t.Errorf("Action = %q, want %q", e.Action, ActionCreate)
	}
	if e.Resource != "nodes" {
		t.Errorf("Resource = %q, want %q", e.Resource, "nodes")
	}
	if e.ResourceID != "node-42" {
		t.Errorf("ResourceID = %q, want %q", e.ResourceID, "node-42")
	}
	if e.Result != ResultSuccess {
		t.Errorf("Result = %q, want %q", e.Result, ResultSuccess)
	}
	if e.Timestamp.IsZero() {
		t.Error("Timestamp should not be zero")
	}
}

func TestJSONFileAuditLogger(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")

	logger, err := NewJSONFileAuditLogger(path)
	if err != nil {
		t.Fatalf("NewJSONFileAuditLogger: %v", err)
	}

	// Write events
	e1 := NewEvent("admin", ActionLogin, "auth", "", ResultSuccess)
	e1.SourceIP = "192.168.1.1"
	e1.Details = map[string]string{"method": "password"}
	logger.Log(e1)

	e2 := NewEvent("admin", ActionCreate, "nodes", "node-1", ResultSuccess)
	logger.Log(e2)

	e3 := NewEvent("viewer", ActionDelete, "acl", "rule-5", ResultDenied)
	logger.Log(e3)

	if err := logger.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	// Read back and verify
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read audit log: %v", err)
	}

	lines := splitJSONLines(data)
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3", len(lines))
	}

	var first AuditEvent
	if err := json.Unmarshal(lines[0], &first); err != nil {
		t.Fatalf("unmarshal first event: %v", err)
	}
	if first.Actor != "admin" {
		t.Errorf("first Actor = %q, want %q", first.Actor, "admin")
	}
	if first.Action != ActionLogin {
		t.Errorf("first Action = %q, want %q", first.Action, ActionLogin)
	}
	if first.SourceIP != "192.168.1.1" {
		t.Errorf("first SourceIP = %q, want %q", first.SourceIP, "192.168.1.1")
	}
	if first.Details["method"] != "password" {
		t.Errorf("first Details method = %q, want %q", first.Details["method"], "password")
	}

	var denied AuditEvent
	if err := json.Unmarshal(lines[2], &denied); err != nil {
		t.Fatalf("unmarshal third event: %v", err)
	}
	if denied.Result != ResultDenied {
		t.Errorf("third Result = %q, want %q", denied.Result, ResultDenied)
	}
}

func TestJSONFileAuditLogger_QueryUnsupported(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")

	logger, err := NewJSONFileAuditLogger(path)
	if err != nil {
		t.Fatalf("NewJSONFileAuditLogger: %v", err)
	}
	defer logger.Close()

	_, err = logger.Query(AuditFilter{})
	if err == nil {
		t.Error("expected error for unsupported query")
	}
}

func TestJSONFileAuditLogger_AppendMode(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "audit.jsonl")

	// First session
	logger1, err := NewJSONFileAuditLogger(path)
	if err != nil {
		t.Fatalf("NewJSONFileAuditLogger: %v", err)
	}
	logger1.Log(NewEvent("user-1", ActionCreate, "nodes", "n1", ResultSuccess))
	logger1.Close()

	// Second session should append
	logger2, err := NewJSONFileAuditLogger(path)
	if err != nil {
		t.Fatalf("NewJSONFileAuditLogger second: %v", err)
	}
	logger2.Log(NewEvent("user-2", ActionDelete, "nodes", "n2", ResultSuccess))
	logger2.Close()

	data, _ := os.ReadFile(path)
	lines := splitJSONLines(data)
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2", len(lines))
	}
}

func TestJSONFileAuditLogger_BadPath(t *testing.T) {
	_, err := NewJSONFileAuditLogger("/nonexistent/dir/audit.jsonl")
	if err == nil {
		t.Error("expected error for bad path")
	}
}

func TestNopAuditLogger(t *testing.T) {
	logger := NopAuditLogger{}
	// Should not panic
	logger.Log(NewEvent("test", ActionCreate, "nodes", "", ResultSuccess))
	events, err := logger.Query(AuditFilter{})
	if err != nil {
		t.Errorf("Query: %v", err)
	}
	if events != nil {
		t.Errorf("events = %v, want nil", events)
	}
	if err := logger.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

func TestAuditFilter_Fields(t *testing.T) {
	f := AuditFilter{
		Actor:     "admin",
		Action:    ActionDelete,
		Resource:  "nodes",
		StartTime: time.Now().Add(-24 * time.Hour),
		EndTime:   time.Now(),
		Limit:     100,
	}
	if f.Actor != "admin" {
		t.Errorf("Actor = %q", f.Actor)
	}
	if f.Limit != 100 {
		t.Errorf("Limit = %d", f.Limit)
	}
}

// splitJSONLines splits JSON Lines data into individual JSON objects.
func splitJSONLines(data []byte) [][]byte {
	var lines [][]byte
	start := 0
	for i, b := range data {
		if b == '\n' {
			line := data[start:i]
			if len(line) > 0 {
				lines = append(lines, line)
			}
			start = i + 1
		}
	}
	if start < len(data) {
		lines = append(lines, data[start:])
	}
	return lines
}
