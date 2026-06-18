//go:build darwin

package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func getAutoStartEnabled() (bool, error) {
	targetPath, err := darwinLaunchAgentPath()
	if err != nil {
		return false, err
	}
	data, err := os.ReadFile(targetPath)
	if os.IsNotExist(err) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read LaunchAgent plist: %w", err)
	}
	executablePath, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("resolve executable path: %w", err)
	}
	return strings.Contains(string(data), executablePath), nil
}

func setAutoStartEnabled(enabled bool) error {
	targetPath, err := darwinLaunchAgentPath()
	if err != nil {
		return err
	}
	if !enabled {
		if err := os.Remove(targetPath); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove LaunchAgent plist: %w", err)
		}
		return nil
	}
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return fmt.Errorf("create LaunchAgents dir: %w", err)
	}
	content := fmt.Sprintf(`<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>Label</key>
  <string>com.nextunnel.desktop</string>
  <key>ProgramArguments</key>
  <array>
    <string>%s</string>
  </array>
  <key>RunAtLoad</key>
  <true/>
</dict>
</plist>
`, executablePath)
	if err := os.WriteFile(targetPath, []byte(content), 0644); err != nil {
		return fmt.Errorf("write LaunchAgent plist: %w", err)
	}
	return nil
}

func darwinLaunchAgentPath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(homeDir, "Library", "LaunchAgents", "com.nextunnel.desktop.plist"), nil
}
