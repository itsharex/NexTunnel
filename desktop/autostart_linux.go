//go:build linux

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func getAutoStartEnabled() (bool, error) {
	targetPath, err := linuxAutostartPath()
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(targetPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read autostart desktop file: %w", err)
	}
	executablePath, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("resolve executable path: %w", err)
	}
	return strings.Contains(string(data), executablePath), nil
}

func setAutoStartEnabled(enabled bool) error {
	targetPath, err := linuxAutostartPath()
	if err != nil {
		return err
	}
	if !enabled {
		if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove autostart desktop file: %w", err)
		}
		return nil
	}
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("create autostart dir: %w", err)
	}
	content := fmt.Sprintf(`[Desktop Entry]
Type=Application
Name=NexTunnel
Exec=%s
Terminal=false
X-GNOME-Autostart-enabled=true
`, executablePath)
	if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write autostart desktop file: %w", err)
	}
	return nil
}

func linuxAutostartPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve config dir: %w", err)
	}
	return filepath.Join(configDir, "autostart", "nextunnel.desktop"), nil
}
