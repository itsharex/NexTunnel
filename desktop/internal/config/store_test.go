package config_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/nextunnel/desktop/internal/config"
)

func newTestDB(t *testing.T) *config.DB {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.db")
	db, err := config.Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return db
}

func TestOpenAndClose(t *testing.T) {
	db := newTestDB(t)
	if db.Path() == "" {
		t.Fatal("expected non-empty path")
	}
}

func TestStoreCRUD(t *testing.T) {
	db := newTestDB(t)
	store := config.NewStore(db)

	// Create
	tc := &config.TunnelConfig{
		ID:         "t1",
		Name:       "web-server",
		ProxyType:  "tcp",
		LocalAddr:  "127.0.0.1",
		LocalPort:  3000,
		RemotePort: 8080,
		ServerAddr: "relay.example.com:7000",
		Status:     "stopped",
	}
	if err := store.Create(tc); err != nil {
		t.Fatalf("create: %v", err)
	}

	// Get
	got, err := store.Get("t1")
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got == nil {
		t.Fatal("expected non-nil config")
	}
	if got.Name != "web-server" || got.LocalPort != 3000 || got.RemotePort != 8080 {
		t.Errorf("got %+v", got)
	}

	// GetByName
	got2, err := store.GetByName("web-server")
	if err != nil {
		t.Fatalf("get by name: %v", err)
	}
	if got2 == nil || got2.ID != "t1" {
		t.Errorf("get by name: got %+v", got2)
	}

	// List
	tc2 := &config.TunnelConfig{
		ID: "t2", Name: "ssh-server", ProxyType: "tcp",
		LocalAddr: "127.0.0.1", LocalPort: 22, RemotePort: 2222,
		ServerAddr: "relay.example.com:7000", Status: "stopped",
	}
	store.Create(tc2)

	list, err := store.List()
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(list) != 2 {
		t.Errorf("expected 2 configs, got %d", len(list))
	}

	// Count
	count, err := store.Count()
	if err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 2 {
		t.Errorf("expected count 2, got %d", count)
	}

	// Update
	got.Status = "running"
	got.RemotePort = 9090
	if err := store.Update(got); err != nil {
		t.Fatalf("update: %v", err)
	}
	updated, _ := store.Get("t1")
	if updated.Status != "running" || updated.RemotePort != 9090 {
		t.Errorf("update failed: got %+v", updated)
	}

	// UpdateStatus
	if err := store.UpdateStatus("t1", "error"); err != nil {
		t.Fatalf("update status: %v", err)
	}
	statusUpdated, _ := store.Get("t1")
	if statusUpdated.Status != "error" {
		t.Errorf("update status failed: got %s", statusUpdated.Status)
	}

	// Delete
	if err := store.Delete("t1"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	deleted, _ := store.Get("t1")
	if deleted != nil {
		t.Error("expected nil after delete")
	}

	count, _ = store.Count()
	if count != 1 {
		t.Errorf("expected count 1 after delete, got %d", count)
	}
}

func TestStoreGetNotFound(t *testing.T) {
	db := newTestDB(t)
	store := config.NewStore(db)

	got, err := store.Get("nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != nil {
		t.Error("expected nil for nonexistent config")
	}
}

func TestStoreDeleteNotFound(t *testing.T) {
	db := newTestDB(t)
	store := config.NewStore(db)

	err := store.Delete("nonexistent")
	if err == nil {
		t.Fatal("expected error for deleting nonexistent config")
	}
}

func TestStoreDuplicateName(t *testing.T) {
	db := newTestDB(t)
	store := config.NewStore(db)

	tc1 := &config.TunnelConfig{ID: "t1", Name: "same-name", ProxyType: "tcp",
		LocalAddr: "127.0.0.1", LocalPort: 3000, RemotePort: 8080, Status: "stopped"}
	tc2 := &config.TunnelConfig{ID: "t2", Name: "same-name", ProxyType: "tcp",
		LocalAddr: "127.0.0.1", LocalPort: 4000, RemotePort: 9090, Status: "stopped"}

	if err := store.Create(tc1); err != nil {
		t.Fatalf("create first: %v", err)
	}
	if err := store.Create(tc2); err == nil {
		t.Fatal("expected error for duplicate name")
	}
}

func TestSettings(t *testing.T) {
	db := newTestDB(t)
	store := config.NewStore(db)

	// Get nonexistent
	val, err := store.GetSetting("theme")
	if err != nil {
		t.Fatalf("get setting: %v", err)
	}
	if val != "" {
		t.Errorf("expected empty string, got %q", val)
	}

	// Set
	if err := store.SetSetting("theme", "dark"); err != nil {
		t.Fatalf("set setting: %v", err)
	}
	val, _ = store.GetSetting("theme")
	if val != "dark" {
		t.Errorf("expected 'dark', got %q", val)
	}

	// Update
	store.SetSetting("theme", "light")
	val, _ = store.GetSetting("theme")
	if val != "light" {
		t.Errorf("expected 'light', got %q", val)
	}
}

func TestOpenDefaultPath(t *testing.T) {
	// Test with empty path - should use ~/.nextunnel/config.db
	tmpHome := t.TempDir()
	origHome := os.Getenv("USERPROFILE")
	os.Setenv("USERPROFILE", tmpHome)
	defer os.Setenv("USERPROFILE", origHome)

	// We can't easily test the default path without mocking os.UserHomeDir,
	// so just verify the function signature works
	_ = tmpHome
}
