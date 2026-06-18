//go:build !windows && !linux && !darwin

package main

import "fmt"

func getAutoStartEnabled() (bool, error) {
	return false, nil
}

func setAutoStartEnabled(enabled bool) error {
	if enabled {
		return fmt.Errorf("autostart is not supported on this platform")
	}
	return nil
}
