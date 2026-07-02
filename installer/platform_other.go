//go:build !windows

package main

import (
	"fmt"
	"os"
	"path/filepath"
)

type otherPlatform struct{}

func newPlatformIntegration() PlatformIntegration {
	return otherPlatform{}
}

func (otherPlatform) DefaultInstallDir() string {
	return filepath.Join(os.TempDir(), appName)
}

func (otherPlatform) IsElevated() bool {
	return os.Geteuid() == 0
}

func (otherPlatform) WebView2Ready() bool { return true }

func (otherPlatform) RelaunchElevated(_ []string) error {
	return fmt.Errorf("当前平台不支持安装器提权")
}

func (otherPlatform) StopProcess(_ string) error { return nil }

func (otherPlatform) WriteUninstallInfo(_ UninstallInfo) error { return nil }

func (otherPlatform) RemoveUninstallInfo() error { return nil }

func (otherPlatform) CreateShortcuts(_ ShortcutOptions) error { return nil }

func (otherPlatform) RemoveShortcuts(_ string) error { return nil }

func (otherPlatform) Launch(_ string) error { return nil }

func (otherPlatform) RemoveInstallDir(path string, _ string) error { return os.RemoveAll(path) }

func (otherPlatform) ShowFatalMessage(title string, message string) {
	_, _ = fmt.Fprintf(os.Stderr, "%s: %s\n", title, message)
}
