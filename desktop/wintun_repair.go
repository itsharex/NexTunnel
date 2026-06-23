package main

import (
	"fmt"
	"strings"
)

const (
	defaultWintunDownloadURL = "https://www.wintun.net/builds/wintun-0.14.1.zip"
	defaultWintunSHA256      = "07c256185d6ee3652e09fa55c0b673e2624b565e02c4b9091c79ca7d2f24ef51"

	wintunRepairSourceBundled  = "bundled"
	wintunRepairSourceDownload = "download"
)

// WintunStatus 描述 Windows 真实 TUN 运行所需 DLL 的当前状态。
type WintunStatus struct {
	Found          bool   `json:"found"`
	Path           string `json:"path"`
	ArchCompatible bool   `json:"arch_compatible"`
	Installable    bool   `json:"installable"`
	NeedsAdmin     bool   `json:"needs_admin"`
	Message        string `json:"message"`
	Action         string `json:"action"`
}

// RepairWintunInput 描述应用内修复 Wintun 的来源。
type RepairWintunInput struct {
	Source string `json:"source"`
}

func normalizeWintunRepairSource(source string) (string, error) {
	normalizedSource := strings.TrimSpace(strings.ToLower(source))
	if normalizedSource == "" {
		return wintunRepairSourceDownload, nil
	}
	switch normalizedSource {
	case wintunRepairSourceBundled, wintunRepairSourceDownload:
		return normalizedSource, nil
	default:
		return "", fmt.Errorf("unsupported Wintun repair source: %s", source)
	}
}

// GetWintunStatus 返回当前 wintun.dll 查找、架构和可修复状态。
func (a *App) GetWintunStatus() WintunStatus {
	return currentWintunStatus()
}

// RepairWintun 下载或复制官方 wintun.dll 到 NexTunnel EXE 同目录。
func (a *App) RepairWintun(input RepairWintunInput) (WintunStatus, error) {
	status, err := repairWintun(input)
	if err != nil {
		a.recordError(err)
		return status, err
	}
	a.clearError()
	a.appendActivityLog(activityLog{
		Level:      activityLogLevelInfo,
		Category:   activityLogCategorySecurity,
		Action:     activityActionRepairWintun,
		TargetType: activityTargetWintun,
		Title:      "Wintun 修复完成",
		Message:    status.Message,
		Metadata: map[string]string{
			"path": status.Path,
		},
	})
	return status, nil
}

// RelaunchAsAdminForWintunRepair 使用系统 UAC 重新拉起隐藏修复进程。
func (a *App) RelaunchAsAdminForWintunRepair() error {
	if err := relaunchAsAdminForWintunRepair(); err != nil {
		a.recordError(err)
		return err
	}
	a.clearError()
	a.appendActivityLog(activityLog{
		Level:      activityLogLevelInfo,
		Category:   activityLogCategorySecurity,
		Action:     activityActionRepairWintun,
		TargetType: activityTargetWintun,
		Title:      "已请求管理员修复 Wintun",
		Message:    "已通过 UAC 请求以管理员权限执行 Wintun 修复。",
	})
	return nil
}
