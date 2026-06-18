package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"runtime"
	"strings"
	"time"
)

const (
	defaultUpdateCheckURL = "https://api.github.com/repos/Lee-zg/NexTunnel/releases/latest"
	updateCheckTimeout    = 6 * time.Second
)

// UpdateInfo 描述一次更新检查结果。
type UpdateInfo struct {
	Available      bool   `json:"available"`
	CurrentVersion string `json:"current_version"`
	LatestVersion  string `json:"latest_version"`
	URL            string `json:"url"`
	Changelog      string `json:"changelog"`
	Error          string `json:"error"`
}

// DiagnosticsInfo 返回脱敏后的诊断文本和关键运行摘要。
type DiagnosticsInfo struct {
	Text             string `json:"text"`
	GeneratedAt      string `json:"generated_at"`
	ConnectionStatus string `json:"connection_status"`
	NATType          string `json:"nat_type"`
}

type githubReleaseResponse struct {
	TagName string `json:"tag_name"`
	HTMLURL string `json:"html_url"`
	Body    string `json:"body"`
}

// CheckForUpdate 检查 GitHub Releases；失败时返回结构化错误，不影响主流程。
func (a *App) CheckForUpdate() (*UpdateInfo, error) {
	info := &UpdateInfo{CurrentVersion: AppVersion}
	ctx, cancel := context.WithTimeout(context.Background(), updateCheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, defaultUpdateCheckURL, nil)
	if err != nil {
		info.Error = err.Error()
		return info, nil
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "NexTunnel/"+AppVersion)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		info.Error = err.Error()
		return info, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		info.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		return info, nil
	}
	var release githubReleaseResponse
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		info.Error = err.Error()
		return info, nil
	}
	info.LatestVersion = release.TagName
	info.URL = release.HTMLURL
	info.Changelog = release.Body
	info.Available = normalizeVersion(release.TagName) != "" && normalizeVersion(release.TagName) != normalizeVersion(AppVersion)
	a.appendActivityLog(activityLog{
		Level:      activityLogLevelInfo,
		Category:   activityLogCategoryOperation,
		Action:     activityActionUpdate,
		TargetType: activityTargetRuntime,
		Title:      "更新检查完成",
		Message:    "已检查 GitHub Releases 最新版本。",
		Metadata: map[string]string{
			"latest_version": info.LatestVersion,
			"available":      fmt.Sprintf("%t", info.Available),
		},
	})
	return info, nil
}

// CollectDiagnostics 汇总脱敏诊断信息，供用户复制到 issue 或工单。
func (a *App) CollectDiagnostics() DiagnosticsInfo {
	runtimeStatus := a.GetRuntimeStatus()
	logs, _ := a.ListActivityLogs(ActivityLogFilter{Limit: 20})
	var buffer bytes.Buffer
	buffer.WriteString("NexTunnel Diagnostics\n")
	buffer.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format(time.RFC3339)))
	buffer.WriteString(fmt.Sprintf("Version: %s\n", AppVersion))
	buffer.WriteString(fmt.Sprintf("OS: %s/%s\n", runtime.GOOS, runtime.GOARCH))
	buffer.WriteString(fmt.Sprintf("Connection: %s\n", runtimeStatus.ConnectionStatus))
	buffer.WriteString(fmt.Sprintf("P2P: %s\n", runtimeStatus.P2PStatus))
	buffer.WriteString(fmt.Sprintf("NAT: %s\n", runtimeStatus.NATType))
	buffer.WriteString(fmt.Sprintf("TUN Platform: %s\n", runtimeStatus.TUN.PlatformName))
	buffer.WriteString(fmt.Sprintf("TUN ProductionMode: %s\n", runtimeStatus.TUN.ProductionMode))
	buffer.WriteString(fmt.Sprintf("VirtualNetwork Applied: %t\n", runtimeStatus.VirtualNetwork.Applied))
	if runtimeStatus.LastError != "" {
		buffer.WriteString(fmt.Sprintf("LastError: %s\n", runtimeStatus.LastError))
	}
	buffer.WriteString("\nRecent Activity:\n")
	for _, log := range logs {
		buffer.WriteString(fmt.Sprintf("- [%s] %s %s: %s\n", log.Level, log.Category, log.Action, log.Title))
	}
	text := buffer.String()
	a.appendActivityLog(activityLog{
		Level:      activityLogLevelInfo,
		Category:   activityLogCategoryOperation,
		Action:     activityActionDiag,
		TargetType: activityTargetRuntime,
		Title:      "诊断信息已生成",
		Message:    "已生成脱敏运行诊断信息。",
	})
	return DiagnosticsInfo{
		Text:             sanitizeDiagnosticsText(text),
		GeneratedAt:      time.Now().Format(time.RFC3339),
		ConnectionStatus: runtimeStatus.ConnectionStatus,
		NATType:          runtimeStatus.NATType,
	}
}

func normalizeVersion(value string) string {
	return strings.TrimPrefix(strings.TrimSpace(strings.ToLower(value)), "v")
}

func sanitizeDiagnosticsText(value string) string {
	replacer := strings.NewReplacer(
		"relay_token", "relay-token-masked",
		"control_plane_token", "control-plane-token-masked",
	)
	return replacer.Replace(value)
}
