package main

import (
	"net"
	"path/filepath"
	"strings"
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

	logs := mustListActivityLogs(t, app, ActivityLogFilter{Category: activityLogCategorySecurity, Limit: 10})
	if len(logs) == 0 || logs[0].Action != activityActionSaveSettings {
		t.Fatalf("expected save settings activity log, got %+v", logs)
	}
	if _, exists := logs[0].Metadata["relay_token"]; exists {
		t.Fatalf("activity log metadata must not contain raw relay token: %+v", logs[0].Metadata)
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

func TestServerSettingsMultiNodeSwitch(t *testing.T) {
	app := newTestApp(t)
	settings := ServerSettings{
		ActiveNodeID:      "remote",
		RelayAddr:         "relay.remote.example:7000",
		ControlPlaneURL:   "150.158.18.55:9090",
		ControlPlaneToken: "cp-token",
		STUNServer:        "stun.remote.example:3478",
		STUNAltServer:     "stun-alt.remote.example:3478",
		Nodes: []ServerNodeSettings{
			{
				ID:            "local",
				Name:          "本地",
				RelayAddr:     "127.0.0.1:7000",
				STUNServer:    "stun.local.example:3478",
				STUNAltServer: "stun.local.example:3478",
			},
			{
				ID:                "remote",
				Name:              "远端",
				RelayAddr:         "relay.remote.example:7000",
				ControlPlaneURL:   "150.158.18.55:9090",
				ControlPlaneToken: "cp-token",
				STUNServer:        "stun.remote.example:3478",
				STUNAltServer:     "stun-alt.remote.example:3478",
			},
		},
	}

	if err := app.SaveServerSettings(settings); err != nil {
		t.Fatalf("SaveServerSettings: %v", err)
	}
	got := app.GetServerSettings()
	if got.ActiveNodeID != "remote" || got.RelayAddr != "relay.remote.example:7000" {
		t.Fatalf("unexpected active node settings: %+v", got)
	}
	if got.ControlPlaneURL != "http://150.158.18.55:9090" {
		t.Fatalf("ControlPlaneURL = %q", got.ControlPlaneURL)
	}
	if len(got.Nodes) != 2 {
		t.Fatalf("expected 2 nodes, got %+v", got.Nodes)
	}
}

func TestNormalizeHTTPBaseURLAddsScheme(t *testing.T) {
	got, err := normalizeHTTPBaseURL("150.158.18.55:9090/")
	if err != nil {
		t.Fatalf("normalizeHTTPBaseURL: %v", err)
	}
	if got != "http://150.158.18.55:9090" {
		t.Fatalf("url = %q", got)
	}
}

func TestNormalizeWintunRepairSource(t *testing.T) {
	tests := []struct {
		name    string
		source  string
		want    string
		wantErr bool
	}{
		{name: "empty defaults to download", source: "", want: wintunRepairSourceDownload},
		{name: "download accepted", source: " download ", want: wintunRepairSourceDownload},
		{name: "bundled accepted", source: "Bundled", want: wintunRepairSourceBundled},
		{name: "invalid rejected", source: "mirror", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := normalizeWintunRepairSource(tt.source)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected unsupported Wintun repair source to fail")
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeWintunRepairSource: %v", err)
			}
			if got != tt.want {
				t.Fatalf("source = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFavoritePortLifecycle(t *testing.T) {
	app := newTestApp(t)

	defaultPorts, err := app.ListFavoritePorts()
	if err != nil {
		t.Fatalf("ListFavoritePorts defaults: %v", err)
	}
	if len(defaultPorts) == 0 {
		t.Fatal("expected built-in favorite ports")
	}

	saved, err := app.SaveFavoritePort(FavoritePortInput{
		Name:        "Custom Dev",
		Category:    "development",
		Port:        45678,
		Protocol:    "tcp",
		Description: "自定义本地服务",
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("SaveFavoritePort: %v", err)
	}
	if saved.ID == "" || saved.Port != 45678 || saved.Builtin {
		t.Fatalf("unexpected saved favorite port: %+v", saved)
	}

	if err := app.DeleteFavoritePort(saved.ID); err != nil {
		t.Fatalf("DeleteFavoritePort: %v", err)
	}

	logs := mustListActivityLogs(t, app, ActivityLogFilter{Category: activityLogCategoryOperation, Limit: 10})
	if !containsActivityAction(logs, activityActionSaveFavoritePort) || !containsActivityAction(logs, activityActionDeleteFavoritePort) {
		t.Fatalf("expected favorite port activity logs, got %+v", logs)
	}
}

func TestCreateTunnelWritesActivityLog(t *testing.T) {
	app := newTestApp(t)

	tunnelInfo, err := app.CreateTunnel(CreateTunnelInput{
		Name:       "web",
		ProxyType:  "tcp",
		LocalAddr:  "127.0.0.1",
		LocalPort:  3000,
		RemotePort: 13000,
	})
	if err != nil {
		t.Fatalf("CreateTunnel: %v", err)
	}

	logs := mustListActivityLogs(t, app, ActivityLogFilter{Category: activityLogCategoryOperation, Limit: 10})
	if len(logs) == 0 || logs[0].Action != activityActionCreateTunnel || logs[0].TargetID != tunnelInfo.ID {
		t.Fatalf("expected create tunnel activity log, got %+v", logs)
	}
}

func TestUpdateTunnelWritesActivityLog(t *testing.T) {
	app := newTestApp(t)

	tunnelInfo, err := app.CreateTunnel(CreateTunnelInput{
		Name:       "web",
		ProxyType:  "tcp",
		LocalAddr:  "127.0.0.1",
		LocalPort:  3000,
		RemotePort: 13000,
	})
	if err != nil {
		t.Fatalf("CreateTunnel: %v", err)
	}

	updated, err := app.UpdateTunnel(UpdateTunnelInput{
		ID:         tunnelInfo.ID,
		Name:       "web-updated",
		ProxyType:  "http",
		LocalAddr:  "127.0.0.1",
		LocalPort:  5173,
		RemotePort: 15173,
	})
	if err != nil {
		t.Fatalf("UpdateTunnel: %v", err)
	}
	if updated.Name != "web-updated" || updated.ProxyType != "http" || updated.LocalPort != 5173 || updated.RemotePort != 15173 {
		t.Fatalf("unexpected updated tunnel: %+v", updated)
	}

	logs := mustListActivityLogs(t, app, ActivityLogFilter{Category: activityLogCategoryOperation, Limit: 10})
	if !containsActivityAction(logs, activityActionUpdateTunnel) {
		t.Fatalf("expected update tunnel activity log, got %+v", logs)
	}
}

func TestUpdateTunnelRejectsInvalidInput(t *testing.T) {
	app := newTestApp(t)

	if _, err := app.UpdateTunnel(UpdateTunnelInput{Name: "web", LocalAddr: "127.0.0.1", LocalPort: 3000, RemotePort: 13000}); err == nil {
		t.Fatal("expected missing tunnel id to fail")
	}

	tunnelInfo, err := app.CreateTunnel(CreateTunnelInput{
		Name:       "web",
		ProxyType:  "tcp",
		LocalAddr:  "127.0.0.1",
		LocalPort:  3000,
		RemotePort: 13000,
	})
	if err != nil {
		t.Fatalf("CreateTunnel: %v", err)
	}
	if _, err := app.UpdateTunnel(UpdateTunnelInput{
		ID:         tunnelInfo.ID,
		Name:       "web",
		ProxyType:  "tcp",
		LocalAddr:  "127.0.0.1",
		LocalPort:  0,
		RemotePort: 13000,
	}); err == nil {
		t.Fatal("expected invalid local port to fail")
	}
}

func TestUpdateTunnelRejectsRunningTunnel(t *testing.T) {
	app := newTestApp(t)

	tunnelInfo, err := app.CreateTunnel(CreateTunnelInput{
		Name:       "web",
		ProxyType:  "tcp",
		LocalAddr:  "127.0.0.1",
		LocalPort:  3000,
		RemotePort: 13000,
	})
	if err != nil {
		t.Fatalf("CreateTunnel: %v", err)
	}
	if err := app.store.UpdateStatus(tunnelInfo.ID, statusRunning); err != nil {
		t.Fatalf("UpdateStatus: %v", err)
	}

	if _, err := app.UpdateTunnel(UpdateTunnelInput{
		ID:         tunnelInfo.ID,
		Name:       "web-updated",
		ProxyType:  "tcp",
		LocalAddr:  "127.0.0.1",
		LocalPort:  3001,
		RemotePort: 13001,
	}); err == nil {
		t.Fatal("expected running tunnel update to fail")
	}
}

func TestRuntimeErrorWritesActivityLog(t *testing.T) {
	app := newTestApp(t)

	err := app.StartTunnel("missing")
	if err == nil {
		t.Fatal("expected StartTunnel to fail")
	}

	logs := mustListActivityLogs(t, app, ActivityLogFilter{Level: activityLogLevelError, Category: activityLogCategoryError, Limit: 10})
	if len(logs) == 0 || logs[0].Action != activityActionRuntimeError {
		t.Fatalf("expected runtime error activity log, got %+v", logs)
	}
}

func TestScanLocalPortsDetectsOpenLoopbackPort(t *testing.T) {
	app := newTestApp(t)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen local port: %v", err)
	}
	defer listener.Close()

	port := listener.Addr().(*net.TCPAddr).Port
	results, err := app.ScanLocalPorts(LocalPortScanInput{
		Host:    "127.0.0.1",
		Ports:   []int{port},
		Timeout: 500,
	})
	if err != nil {
		t.Fatalf("ScanLocalPorts: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 scan result, got %d", len(results))
	}
	if !results[0].Open || results[0].Port != port {
		t.Fatalf("expected open port %d, got %+v", port, results[0])
	}

	logs := mustListActivityLogs(t, app, ActivityLogFilter{Category: activityLogCategorySecurity, Limit: 10})
	if !containsActivityAction(logs, activityActionScanLocalPorts) {
		t.Fatalf("expected local port scan activity log, got %+v", logs)
	}
}

func TestScanLocalPortsRejectsNonLoopbackHost(t *testing.T) {
	app := newTestApp(t)
	_, err := app.ScanLocalPorts(LocalPortScanInput{
		Host:  "192.0.2.1",
		Ports: []int{80},
	})
	if err == nil {
		t.Fatal("expected non-loopback scan to be rejected")
	}
	logs := mustListActivityLogs(t, app, ActivityLogFilter{Level: activityLogLevelError, Category: activityLogCategoryError, Limit: 10})
	if len(logs) == 0 {
		t.Fatal("expected rejected scan to write error activity log")
	}
}

func TestScanLocalPortsRejectsUnlistedLoopbackAlias(t *testing.T) {
	app := newTestApp(t)
	_, err := app.ScanLocalPorts(LocalPortScanInput{
		Host:  "127.0.0.2",
		Ports: []int{80},
	})
	if err == nil {
		t.Fatal("expected unlisted loopback alias to be rejected")
	}
}

func TestClearActivityLogsLeavesAuditRecord(t *testing.T) {
	app := newTestApp(t)
	app.recordError(net.ErrClosed)

	if err := app.ClearActivityLogs(); err != nil {
		t.Fatalf("ClearActivityLogs: %v", err)
	}
	logs := mustListActivityLogs(t, app, ActivityLogFilter{Limit: 10})
	if len(logs) != 1 || logs[0].Action != activityActionClearActivityLogs {
		t.Fatalf("expected clear audit record, got %+v", logs)
	}
}

func TestConfigExportMasksSensitiveTokens(t *testing.T) {
	app := newTestApp(t)
	if err := app.SaveServerSettings(ServerSettings{
		RelayAddr:         "relay.example.com:7000",
		RelayToken:        "secret-relay-token",
		ControlPlaneURL:   "https://cp.example.com",
		ControlPlaneToken: "secret-control-token",
		STUNServer:        "stun-a.example.com:3478",
		STUNAltServer:     "stun-b.example.com:3478",
	}); err != nil {
		t.Fatalf("SaveServerSettings: %v", err)
	}

	exported, err := app.ExportConfig(ExportConfigOptions{})
	if err != nil {
		t.Fatalf("ExportConfig: %v", err)
	}
	if strings.Contains(exported, "secret-relay-token") || strings.Contains(exported, "secret-control-token") {
		t.Fatalf("export must mask sensitive tokens: %s", exported)
	}

	exportedWithSecrets, err := app.ExportConfig(ExportConfigOptions{IncludeSensitive: true})
	if err != nil {
		t.Fatalf("ExportConfig include sensitive: %v", err)
	}
	if !strings.Contains(exportedWithSecrets, "secret-relay-token") || !strings.Contains(exportedWithSecrets, "secret-control-token") {
		t.Fatalf("export should include sensitive tokens when requested: %s", exportedWithSecrets)
	}
}

func TestConfigImportRestoresPreferencesAndTunnel(t *testing.T) {
	source := newTestApp(t)
	if err := source.SaveAppearanceSettings(AppearanceSettings{
		ThemeMode:   "system",
		MotionLevel: "reduced",
		Language:    "en-US",
		AccentColor: "#8a2be2",
	}); err != nil {
		t.Fatalf("SaveAppearanceSettings: %v", err)
	}
	if err := source.SaveGeneralSettings(GeneralSettings{AutoConnect: true, MinimizeToTray: true}); err != nil {
		t.Fatalf("SaveGeneralSettings: %v", err)
	}
	if _, err := source.CreateTunnel(CreateTunnelInput{
		Name:       "api",
		ProxyType:  "tcp",
		LocalAddr:  "127.0.0.1",
		LocalPort:  8080,
		RemotePort: 18080,
	}); err != nil {
		t.Fatalf("CreateTunnel: %v", err)
	}
	exported, err := source.ExportConfig(ExportConfigOptions{IncludeSensitive: true})
	if err != nil {
		t.Fatalf("ExportConfig: %v", err)
	}

	target := newTestApp(t)
	if err := target.ImportConfig(exported); err != nil {
		t.Fatalf("ImportConfig: %v", err)
	}
	appearance := target.GetAppearanceSettings()
	if appearance.ThemeMode != "system" || appearance.MotionLevel != "reduced" || appearance.Language != "en-US" {
		t.Fatalf("unexpected imported appearance: %+v", appearance)
	}
	general := target.GetGeneralSettings()
	if !general.AutoConnect || !general.MinimizeToTray {
		t.Fatalf("unexpected imported general settings: %+v", general)
	}
	tunnels, err := target.GetTunnels()
	if err != nil {
		t.Fatalf("GetTunnels: %v", err)
	}
	if len(tunnels) != 1 || tunnels[0].Name != "api" {
		t.Fatalf("unexpected imported tunnels: %+v", tunnels)
	}
}

func TestConfigImportKeepsExistingTokensWhenExportWasMasked(t *testing.T) {
	source := newTestApp(t)
	if err := source.SaveServerSettings(ServerSettings{
		RelayAddr:         "relay.example.com:7000",
		RelayToken:        "source-token",
		ControlPlaneURL:   "https://cp.example.com",
		ControlPlaneToken: "source-control-token",
		STUNServer:        "stun-a.example.com:3478",
		STUNAltServer:     "stun-b.example.com:3478",
	}); err != nil {
		t.Fatalf("SaveServerSettings source: %v", err)
	}
	exported, err := source.ExportConfig(ExportConfigOptions{})
	if err != nil {
		t.Fatalf("ExportConfig: %v", err)
	}

	target := newTestApp(t)
	if err := target.SaveServerSettings(ServerSettings{
		RelayAddr:         "old.example.com:7000",
		RelayToken:        "existing-token",
		ControlPlaneURL:   "https://old.example.com",
		ControlPlaneToken: "existing-control-token",
		STUNServer:        "stun-old.example.com:3478",
		STUNAltServer:     "stun-old.example.com:3478",
	}); err != nil {
		t.Fatalf("SaveServerSettings target: %v", err)
	}
	if err := target.ImportConfig(exported); err != nil {
		t.Fatalf("ImportConfig: %v", err)
	}
	settings := target.GetServerSettings()
	if settings.RelayToken != "existing-token" || settings.ControlPlaneToken != "existing-control-token" {
		t.Fatalf("masked import should keep existing tokens: %+v", settings)
	}
}

func TestCollectDiagnosticsMasksTokenWords(t *testing.T) {
	app := newTestApp(t)
	info := app.CollectDiagnostics()
	if info.Text == "" || !strings.Contains(info.Text, "NexTunnel Diagnostics") {
		t.Fatalf("unexpected diagnostics text: %+v", info)
	}
	if strings.Contains(info.Text, "relay_token") || strings.Contains(info.Text, "control_plane_token") {
		t.Fatalf("diagnostics should not expose raw token field names: %s", info.Text)
	}
}

func mustListActivityLogs(t *testing.T, app *App, filter ActivityLogFilter) []ActivityLogInfo {
	t.Helper()
	logs, err := app.ListActivityLogs(filter)
	if err != nil {
		t.Fatalf("ListActivityLogs: %v", err)
	}
	return logs
}

func containsActivityAction(logs []ActivityLogInfo, action string) bool {
	for _, log := range logs {
		if log.Action == action {
			return true
		}
	}
	return false
}
