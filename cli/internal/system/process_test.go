package system

import (
	"os"
	"path/filepath"
	"testing"
)

func TestActiveLinuxServicesUsesDefaultPrefix(t *testing.T) {
	paths := Paths{EnvPath: filepath.Join(t.TempDir(), "missing.env")}

	services := activeLinuxServices(paths)

	want := []string{
		"nextunnel-control-plane.service",
		"nextunnel-relay.service",
		"nextunnel-nat-detector.service",
		"nextunnel-dashboard.service",
	}
	assertStringSliceEqual(t, services, want)
}

func TestActiveLinuxServicesUsesConfiguredPrefixAndDashboardFlag(t *testing.T) {
	envPath := filepath.Join(t.TempDir(), "server.env")
	content := "NEXTUNNEL_SERVICE_PREFIX=nextunnel-wsltest\nDASHBOARD_ENABLED=false\n"
	if err := os.WriteFile(envPath, []byte(content), 0600); err != nil {
		t.Fatalf("write env: %v", err)
	}

	services := activeLinuxServices(Paths{EnvPath: envPath})

	want := []string{
		"nextunnel-wsltest-control-plane.service",
		"nextunnel-wsltest-relay.service",
		"nextunnel-wsltest-nat-detector.service",
	}
	assertStringSliceEqual(t, services, want)
}

func assertStringSliceEqual(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len(got) = %d, want %d; got=%v", len(got), len(want), got)
	}
	for index := range want {
		if got[index] != want[index] {
			t.Fatalf("got[%d] = %q, want %q; got=%v", index, got[index], want[index], got)
		}
	}
}
