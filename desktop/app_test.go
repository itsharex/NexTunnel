package main

import (
	"path/filepath"
	"testing"

	"github.com/nextunnel/desktop/internal/config"
)

func newTestApp(t *testing.T) *App {
	t.Helper()
	db, err := config.Open(filepath.Join(t.TempDir(), "desktop.db"))
	if err != nil {
		t.Fatalf("open config db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	app := NewApp()
	app.db = db
	app.store = config.NewStore(db)
	return app
}

func TestServerSettingsPersistence(t *testing.T) {
	app := newTestApp(t)
	settings := ServerSettings{
		RelayAddr:         "relay.example.com:7000",
		RelayToken:        "relay-token",
		ControlPlaneURL:   "https://cp.example.com/",
		ControlPlaneToken: "cp-token",
		STUNServer:        "stun-a.example.com:3478",
		STUNAltServer:     "stun-b.example.com:3478",
	}

	if err := app.SaveServerSettings(settings); err != nil {
		t.Fatalf("SaveServerSettings: %v", err)
	}
	got := app.GetServerSettings()
	if got.RelayAddr != settings.RelayAddr || got.RelayToken != settings.RelayToken {
		t.Fatalf("unexpected relay settings: %+v", got)
	}
	if got.ControlPlaneURL != "https://cp.example.com" {
		t.Fatalf("ControlPlaneURL = %q", got.ControlPlaneURL)
	}
	if got.STUNAltServer != settings.STUNAltServer {
		t.Fatalf("unexpected STUN alt server: %+v", got)
	}
}

func TestServerSettingsDefaults(t *testing.T) {
	app := newTestApp(t)
	got := app.GetServerSettings()
	if got.RelayAddr != defaultRelayAddr {
		t.Fatalf("RelayAddr = %q, want %q", got.RelayAddr, defaultRelayAddr)
	}
	if got.STUNServer != defaultSTUNServer {
		t.Fatalf("STUNServer = %q, want %q", got.STUNServer, defaultSTUNServer)
	}
}
