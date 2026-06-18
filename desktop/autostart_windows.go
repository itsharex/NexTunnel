//go:build windows

package main

import (
	"fmt"
	"os"

	"golang.org/x/sys/windows/registry"
)

const windowsRunKey = `Software\Microsoft\Windows\CurrentVersion\Run`

func getAutoStartEnabled() (bool, error) {
	key, err := registry.OpenKey(registry.CURRENT_USER, windowsRunKey, registry.QUERY_VALUE)
	if err != nil {
		return false, fmt.Errorf("open Run registry key: %w", err)
	}
	defer key.Close()
	value, _, err := key.GetStringValue("NexTunnel")
	if err == registry.ErrNotExist {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read Run registry value: %w", err)
	}
	executablePath, err := os.Executable()
	if err != nil {
		return false, fmt.Errorf("resolve executable path: %w", err)
	}
	return value == quoteCommandPath(executablePath), nil
}

func setAutoStartEnabled(enabled bool) error {
	key, _, err := registry.CreateKey(registry.CURRENT_USER, windowsRunKey, registry.SET_VALUE|registry.QUERY_VALUE)
	if err != nil {
		return fmt.Errorf("open Run registry key: %w", err)
	}
	defer key.Close()
	if !enabled {
		if err := key.DeleteValue("NexTunnel"); err != nil && err != registry.ErrNotExist {
			return fmt.Errorf("delete Run registry value: %w", err)
		}
		return nil
	}
	executablePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable path: %w", err)
	}
	if err := key.SetStringValue("NexTunnel", quoteCommandPath(executablePath)); err != nil {
		return fmt.Errorf("write Run registry value: %w", err)
	}
	return nil
}

func quoteCommandPath(path string) string {
	return `"` + path + `"`
}
