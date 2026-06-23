//go:build !windows

package main

import "fmt"

func currentWintunStatus() WintunStatus {
	return WintunStatus{
		Found:          false,
		ArchCompatible: false,
		Installable:    false,
		NeedsAdmin:     false,
		Message:        "Wintun 仅适用于 Windows。",
		Action:         "当前平台不需要 wintun.dll。",
	}
}

func repairWintun(RepairWintunInput) (WintunStatus, error) {
	return currentWintunStatus(), fmt.Errorf("Wintun repair is only available on Windows")
}

func relaunchAsAdminForWintunRepair() error {
	return fmt.Errorf("Wintun repair is only available on Windows")
}

func runWintunRepairCommandIfRequested([]string) bool {
	return false
}
