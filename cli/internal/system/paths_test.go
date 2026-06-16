package system

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultLinuxPathsReadsLocalInstallEnv(t *testing.T) {
	root := t.TempDir()
	localEnvDir := filepath.Join(root, "deploy", "server")
	if err := os.MkdirAll(localEnvDir, 0700); err != nil {
		t.Fatalf("mkdir local env dir: %v", err)
	}
	content := "NEXTUNNEL_INSTALL_DIR=/opt/nextunnel-test\nNEXTUNNEL_CONFIG_DIR=/etc/nextunnel-test\nNEXTUNNEL_DATA_DIR=/var/lib/nextunnel-test\n"
	if err := os.WriteFile(filepath.Join(localEnvDir, ".env"), []byte(content), 0600); err != nil {
		t.Fatalf("write local env: %v", err)
	}

	paths := defaultLinuxPaths(func(string) string { return "" }, func() []string { return []string{root} })

	if paths.InstallDir != "/opt/nextunnel-test" {
		t.Fatalf("InstallDir = %q", paths.InstallDir)
	}
	if paths.ConfigDir != "/etc/nextunnel-test" {
		t.Fatalf("ConfigDir = %q", paths.ConfigDir)
	}
	if paths.DataDir != "/var/lib/nextunnel-test" {
		t.Fatalf("DataDir = %q", paths.DataDir)
	}
	wantEnvPath := filepath.ToSlash(filepath.Join("/etc/nextunnel-test", "server.env"))
	if filepath.ToSlash(paths.EnvPath) != wantEnvPath {
		t.Fatalf("EnvPath = %q", paths.EnvPath)
	}
}

func TestDefaultLinuxPathsEnvironmentOverridesLocalInstallEnv(t *testing.T) {
	root := t.TempDir()
	localEnvDir := filepath.Join(root, "deploy", "server")
	if err := os.MkdirAll(localEnvDir, 0700); err != nil {
		t.Fatalf("mkdir local env dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(localEnvDir, ".env"), []byte("NEXTUNNEL_INSTALL_DIR=/opt/from-local\n"), 0600); err != nil {
		t.Fatalf("write local env: %v", err)
	}
	getenv := func(key string) string {
		if key == "NEXTUNNEL_INSTALL_DIR" {
			return "/opt/from-env"
		}
		return ""
	}

	paths := defaultLinuxPaths(getenv, func() []string { return []string{root} })

	if paths.InstallDir != "/opt/from-env" {
		t.Fatalf("InstallDir = %q", paths.InstallDir)
	}
}
