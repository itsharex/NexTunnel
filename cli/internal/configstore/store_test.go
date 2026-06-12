package configstore

import (
	"path/filepath"
	"testing"
)

func TestLoadSaveRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.json")
	want := &Store{
		CurrentContext: "prod",
		Contexts: map[string]Context{
			"prod": {
				Name:           "prod",
				ControlPlane:   "http://127.0.0.1:9090",
				ControlToken:   "control-token",
				Dashboard:      "http://127.0.0.1:8080",
				DashboardToken: "dashboard-token",
			},
		},
	}
	if err := Save(path, want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.CurrentContext != want.CurrentContext {
		t.Fatalf("CurrentContext = %q, want %q", got.CurrentContext, want.CurrentContext)
	}
	if got.Contexts["prod"].ControlToken != "control-token" {
		t.Fatalf("ControlToken not persisted: %+v", got.Contexts["prod"])
	}
}

func TestLoadMissingFileReturnsEmptyStore(t *testing.T) {
	store, err := Load(filepath.Join(t.TempDir(), "missing.json"))
	if err != nil {
		t.Fatalf("Load missing file: %v", err)
	}
	if store.Contexts == nil || len(store.Contexts) != 0 {
		t.Fatalf("expected empty contexts, got %+v", store.Contexts)
	}
}
