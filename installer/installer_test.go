package main

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type memoryPayloadSource struct {
	manifest PayloadManifest
	payload  []byte
}

func (s memoryPayloadSource) Manifest() (PayloadManifest, error) {
	manifest := s.manifest
	applyManifestDefaults(&manifest)
	return manifest, nil
}

func (s memoryPayloadSource) PayloadBytes(_ PayloadManifest) ([]byte, error) {
	return s.payload, nil
}

type fakePlatform struct {
	defaultDir         string
	elevated           bool
	webView2Ready      bool
	failUninstallWrite bool
	wroteUninstall     bool
	createdShortcuts   ShortcutOptions
	launchedPath       string
}

func (p *fakePlatform) DefaultInstallDir() string         { return p.defaultDir }
func (p *fakePlatform) IsElevated() bool                  { return p.elevated }
func (p *fakePlatform) WebView2Ready() bool               { return p.webView2Ready }
func (p *fakePlatform) RelaunchElevated(_ []string) error { return nil }
func (p *fakePlatform) StopProcess(_ string) error        { return nil }
func (p *fakePlatform) WriteUninstallInfo(_ UninstallInfo) error {
	if p.failUninstallWrite {
		return fmt.Errorf("forced integration failure")
	}
	p.wroteUninstall = true
	return nil
}
func (p *fakePlatform) RemoveUninstallInfo() error { return nil }
func (p *fakePlatform) CreateShortcuts(options ShortcutOptions) error {
	p.createdShortcuts = options
	return nil
}
func (p *fakePlatform) RemoveShortcuts(_ string) error { return nil }
func (p *fakePlatform) Launch(path string) error {
	p.launchedPath = path
	return nil
}
func (p *fakePlatform) RemoveInstallDir(path string, _ string) error { return os.RemoveAll(path) }
func (p *fakePlatform) ShowFatalMessage(_, _ string)                 {}

func TestSafeExtractZipRejectsTraversal(t *testing.T) {
	payload := zipPayload(t, map[string]string{
		"../escape.txt": "bad",
	})
	err := safeExtractZip(payload, t.TempDir(), nil)
	if err == nil || !strings.Contains(err.Error(), "非法路径") {
		t.Fatalf("expected traversal error, got %v", err)
	}
}

func TestSafeExtractZipRejectsWindowsAlternateStream(t *testing.T) {
	payload := zipPayload(t, map[string]string{
		"app.exe:evil": "bad",
	})
	err := safeExtractZip(payload, t.TempDir(), nil)
	if err == nil || !strings.Contains(err.Error(), "非法路径") {
		t.Fatalf("expected alternate stream path error, got %v", err)
	}
}

func TestAssertPayloadHash(t *testing.T) {
	payload := []byte("installer")
	hash := sha256.Sum256(payload)
	if err := assertPayloadHash(payload, fmt.Sprintf("%x", hash)); err != nil {
		t.Fatalf("expected hash to pass: %v", err)
	}
	if err := assertPayloadHash(payload, strings.Repeat("0", 64)); err == nil {
		t.Fatal("expected hash mismatch")
	}
}

func TestInstallCreatesFilesAndIntegration(t *testing.T) {
	payload := zipPayload(t, map[string]string{
		appExecutableName: "exe",
		"wintun.dll":      "dll",
	})
	hash := sha256.Sum256(payload)
	installDir := filepath.Join(t.TempDir(), appName)
	platform := &fakePlatform{defaultDir: installDir, elevated: true, webView2Ready: true}
	installer := newTestInstaller(payload, hash, platform)

	result := installer.Install(context.Background(), InstallOptions{
		CreateDesktopShortcut:   true,
		CreateStartMenuShortcut: true,
		LaunchAfterInstall:      true,
	}, nil)
	if !result.Success {
		t.Fatalf("install failed: %+v", result)
	}
	if _, err := os.Stat(filepath.Join(installDir, appExecutableName)); err != nil {
		t.Fatalf("expected app executable: %v", err)
	}
	if !platform.wroteUninstall {
		t.Fatal("expected uninstall info to be written")
	}
	if !platform.createdShortcuts.Desktop || !platform.createdShortcuts.StartMenu {
		t.Fatalf("expected shortcuts, got %+v", platform.createdShortcuts)
	}
	if platform.launchedPath == "" {
		t.Fatal("expected app launch")
	}
}

func TestInstallRollbackRestoresOldVersion(t *testing.T) {
	payload := zipPayload(t, map[string]string{
		appExecutableName: "new",
	})
	hash := sha256.Sum256(payload)
	root := t.TempDir()
	installDir := filepath.Join(root, appName)
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldMarker := filepath.Join(installDir, "old.txt")
	if err := os.WriteFile(oldMarker, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	platform := &fakePlatform{defaultDir: installDir, elevated: true, failUninstallWrite: true}
	installer := newTestInstaller(payload, hash, platform)

	result := installer.Install(context.Background(), InstallOptions{}, nil)
	if result.Success || result.Error == "" {
		t.Fatalf("expected failed install, got %+v", result)
	}
	if data, err := os.ReadFile(oldMarker); err != nil || string(data) != "old" {
		t.Fatalf("old install was not restored: data=%q err=%v", data, err)
	}
}

func TestParseCommandLine(t *testing.T) {
	options, err := ParseCommandLine([]string{
		"--silent",
		"--install-dir", `C:\Program Files\NexTunnel`,
		"--no-launch",
		"--no-desktop-shortcut",
		"--log", `C:\temp\installer.log`,
	})
	if err != nil {
		t.Fatalf("ParseCommandLine: %v", err)
	}
	if options.Mode != commandModeInstall || options.Install.LaunchAfterInstall {
		t.Fatalf("unexpected options: %+v", options)
	}
	if options.Install.CreateDesktopShortcut || !options.Install.CreateStartMenuShortcut {
		t.Fatalf("unexpected shortcut options: %+v", options.Install)
	}
}

func TestPlanIncludesWebView2State(t *testing.T) {
	payload := zipPayload(t, map[string]string{appExecutableName: "exe"})
	hash := sha256.Sum256(payload)
	platform := &fakePlatform{defaultDir: t.TempDir(), elevated: true, webView2Ready: true}
	installer := newTestInstaller(payload, hash, platform)

	plan := installer.Plan()
	if !plan.PayloadReady || !plan.WebView2Ready || plan.WebView2Mode != "embedded-bootstrapper" {
		t.Fatalf("unexpected plan: %+v", plan)
	}
}

func newTestInstaller(payload []byte, hash [32]byte, platform *fakePlatform) *Installer {
	manifest := PayloadManifest{
		Version:         "v9.9.9-test",
		Target:          "windows/amd64",
		PayloadFile:     "payload.zip",
		PayloadSHA256:   fmt.Sprintf("%x", hash),
		AppExecutable:   appExecutableName,
		RequiredSpaceMB: 1,
		WintunIncluded:  true,
		Signing:         "unsigned-test",
	}
	return &Installer{source: memoryPayloadSource{manifest: manifest, payload: payload}, platform: platform}
}

func zipPayload(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	for name, contents := range files {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatalf("create zip entry: %v", err)
		}
		if _, err := entry.Write([]byte(contents)); err != nil {
			t.Fatalf("write zip entry: %v", err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close zip: %v", err)
	}
	return buffer.Bytes()
}
