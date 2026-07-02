package main

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	neturl "net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultUpdateCheckURL      = "https://api.github.com/repos/Lee-zg/NexTunnel/releases/latest"
	updateCheckTimeout         = 6 * time.Second
	updateInstallTimeout       = 2 * time.Minute
	updateChecksumMaxBytes     = 4096
	updateWindowsInstallerHint = "windows-amd64-installer.exe"
)

var (
	updateCheckURL             = defaultUpdateCheckURL
	updateCheckHTTPClient      = http.DefaultClient
	updateDownloadHTTPClient   = http.DefaultClient
	allowedUpdateDownloadHosts = []string{"github.com"}
	currentRuntimeGOOS         = runtime.GOOS
	updateInstallerStarter     = startUpdateInstaller
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
	TagName string               `json:"tag_name"`
	HTMLURL string               `json:"html_url"`
	Body    string               `json:"body"`
	Assets  []githubReleaseAsset `json:"assets"`
}

type githubReleaseAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// UpdateInstallInfo 描述一次更新安装器下载和启动结果。
type UpdateInstallInfo struct {
	Started  bool   `json:"started"`
	FilePath string `json:"file_path"`
	Error    string `json:"error"`
}

// CheckForUpdate 检查 GitHub Releases；失败时返回结构化错误，不影响主流程。
func (a *App) CheckForUpdate() (*UpdateInfo, error) {
	info := &UpdateInfo{CurrentVersion: AppVersion}
	ctx, cancel := context.WithTimeout(context.Background(), updateCheckTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, updateCheckURL, nil)
	if err != nil {
		info.Error = err.Error()
		a.recordUpdateCheckFailure(info.Error)
		return info, nil
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "NexTunnel/"+AppVersion)

	resp, err := updateCheckHTTPClient.Do(req)
	if err != nil {
		info.Error = err.Error()
		a.recordUpdateCheckFailure(info.Error)
		return info, nil
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		info.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
		a.recordUpdateCheckFailure(info.Error)
		return info, nil
	}
	var release githubReleaseResponse
	if err := json.NewDecoder(io.LimitReader(resp.Body, 2<<20)).Decode(&release); err != nil {
		info.Error = err.Error()
		a.recordUpdateCheckFailure(info.Error)
		return info, nil
	}
	info = updateInfoFromRelease(AppVersion, release)
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

// InstallUpdate 下载已校验的 Windows 安装器并启动；失败时返回结构化错误。
func (a *App) InstallUpdate(rawURL string) (*UpdateInstallInfo, error) {
	info := &UpdateInstallInfo{}
	if currentRuntimeGOOS != "windows" {
		info.Error = "当前平台暂不支持应用内启动安装器，请打开发布页手动下载。"
		a.recordUpdateInstallFailure(info.Error)
		return info, nil
	}
	installerURL, err := validateUpdateInstallerURL(rawURL)
	if err != nil {
		info.Error = err.Error()
		a.recordUpdateInstallFailure(info.Error)
		return info, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), updateInstallTimeout)
	defer cancel()
	installerPath, err := downloadVerifiedUpdateInstaller(ctx, installerURL)
	if err != nil {
		info.Error = err.Error()
		a.recordUpdateInstallFailure(info.Error)
		return info, nil
	}
	if err := updateInstallerStarter(installerPath); err != nil {
		info.Error = err.Error()
		info.FilePath = installerPath
		a.recordUpdateInstallFailure(info.Error)
		return info, nil
	}

	info.Started = true
	info.FilePath = installerPath
	a.appendActivityLog(activityLog{
		Level:      activityLogLevelInfo,
		Category:   activityLogCategoryOperation,
		Action:     activityActionUpdate,
		TargetType: activityTargetRuntime,
		Title:      "更新安装器已启动",
		Message:    "已下载并启动通过 SHA256 校验的更新安装器。",
		Metadata: map[string]string{
			"file_path": installerPath,
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

func updateInfoFromRelease(currentVersion string, release githubReleaseResponse) *UpdateInfo {
	info := &UpdateInfo{
		CurrentVersion: currentVersion,
		LatestVersion:  strings.TrimSpace(release.TagName),
		URL:            strings.TrimSpace(release.HTMLURL),
		Changelog:      strings.TrimSpace(release.Body),
	}
	if assetURL := selectWindowsInstallerAssetURL(release.Assets); assetURL != "" {
		info.URL = assetURL
	}
	info.Available = isUpdateAvailable(currentVersion, release.TagName)
	return info
}

func selectWindowsInstallerAssetURL(assets []githubReleaseAsset) string {
	type assetCandidate struct {
		score int
		url   string
	}
	candidates := make([]assetCandidate, 0, len(assets))
	for _, asset := range assets {
		name := strings.ToLower(strings.TrimSpace(asset.Name))
		downloadURL := strings.TrimSpace(asset.BrowserDownloadURL)
		if name == "" || downloadURL == "" || !strings.HasSuffix(name, ".exe") {
			continue
		}
		score := 0
		switch {
		case strings.Contains(name, updateWindowsInstallerHint):
			score = 100
		case strings.Contains(name, "windows") && strings.Contains(name, "amd64") && strings.Contains(name, "installer"):
			score = 80
		case strings.Contains(name, "installer"):
			score = 60
		default:
			score = 20
		}
		candidates = append(candidates, assetCandidate{score: score, url: downloadURL})
	}
	if len(candidates) == 0 {
		return ""
	}
	sort.SliceStable(candidates, func(leftIndex, rightIndex int) bool {
		return candidates[leftIndex].score > candidates[rightIndex].score
	})
	return candidates[0].url
}

func isUpdateAvailable(currentVersion, latestVersion string) bool {
	return compareSemanticVersions(latestVersion, currentVersion) > 0
}

type semanticVersion struct {
	numbers    [3]int
	prerelease []string
	valid      bool
}

func compareSemanticVersions(leftVersion, rightVersion string) int {
	left := parseSemanticVersion(leftVersion)
	right := parseSemanticVersion(rightVersion)
	if !left.valid || !right.valid {
		return 0
	}
	for index := range left.numbers {
		if left.numbers[index] > right.numbers[index] {
			return 1
		}
		if left.numbers[index] < right.numbers[index] {
			return -1
		}
	}
	return comparePrereleaseIdentifiers(left.prerelease, right.prerelease)
}

func parseSemanticVersion(value string) semanticVersion {
	normalized := normalizeVersion(value)
	normalized = strings.SplitN(normalized, "+", 2)[0]
	if normalized == "" {
		return semanticVersion{}
	}
	core := normalized
	prerelease := []string{}
	if strings.Contains(normalized, "-") {
		parts := strings.SplitN(normalized, "-", 2)
		core = parts[0]
		if strings.TrimSpace(parts[1]) == "" {
			return semanticVersion{}
		}
		prerelease = strings.Split(parts[1], ".")
	}
	coreParts := strings.Split(core, ".")
	if len(coreParts) != 3 {
		return semanticVersion{}
	}
	parsed := semanticVersion{prerelease: prerelease, valid: true}
	for index, part := range coreParts {
		number, err := strconv.Atoi(part)
		if err != nil || number < 0 {
			return semanticVersion{}
		}
		parsed.numbers[index] = number
	}
	return parsed
}

func comparePrereleaseIdentifiers(left, right []string) int {
	if len(left) == 0 && len(right) == 0 {
		return 0
	}
	if len(left) == 0 {
		return 1
	}
	if len(right) == 0 {
		return -1
	}
	for index := 0; index < len(left) && index < len(right); index++ {
		comparison := comparePrereleaseIdentifier(left[index], right[index])
		if comparison != 0 {
			return comparison
		}
	}
	if len(left) > len(right) {
		return 1
	}
	if len(left) < len(right) {
		return -1
	}
	return 0
}

func comparePrereleaseIdentifier(left, right string) int {
	leftNumber, leftIsNumeric := parseNumericPrereleaseIdentifier(left)
	rightNumber, rightIsNumeric := parseNumericPrereleaseIdentifier(right)
	switch {
	case leftIsNumeric && rightIsNumeric:
		if leftNumber > rightNumber {
			return 1
		}
		if leftNumber < rightNumber {
			return -1
		}
		return 0
	case leftIsNumeric:
		return -1
	case rightIsNumeric:
		return 1
	default:
		return strings.Compare(left, right)
	}
}

func parseNumericPrereleaseIdentifier(value string) (int, bool) {
	if value == "" {
		return 0, false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return 0, false
		}
	}
	number, err := strconv.Atoi(value)
	if err != nil {
		return 0, false
	}
	return number, true
}

func validateUpdateInstallerURL(rawURL string) (*neturl.URL, error) {
	parsedURL, err := neturl.Parse(strings.TrimSpace(rawURL))
	if err != nil {
		return nil, fmt.Errorf("更新下载链接无效：%w", err)
	}
	if parsedURL.Scheme != "https" {
		return nil, fmt.Errorf("更新下载链接必须使用 HTTPS")
	}
	if !isAllowedUpdateDownloadHost(parsedURL.Hostname()) {
		return nil, fmt.Errorf("更新下载链接必须来自 GitHub Releases")
	}
	if !strings.EqualFold(path.Ext(parsedURL.Path), ".exe") {
		return nil, fmt.Errorf("更新下载链接必须指向 Windows 安装器")
	}
	if strings.TrimSpace(path.Base(parsedURL.Path)) == "" {
		return nil, fmt.Errorf("更新下载链接缺少文件名")
	}
	return parsedURL, nil
}

func isAllowedUpdateDownloadHost(hostname string) bool {
	for _, allowedHost := range allowedUpdateDownloadHosts {
		if strings.EqualFold(hostname, allowedHost) {
			return true
		}
	}
	return false
}

func downloadVerifiedUpdateInstaller(ctx context.Context, installerURL *neturl.URL) (string, error) {
	checksumURL := *installerURL
	checksumURL.Path = checksumURL.Path + ".sha256"
	expectedHash, err := downloadUpdateChecksum(ctx, checksumURL.String())
	if err != nil {
		return "", err
	}

	fileName, err := safeInstallerFileName(installerURL)
	if err != nil {
		return "", err
	}
	tempDir, err := os.MkdirTemp("", "nextunnel-update-*")
	if err != nil {
		return "", fmt.Errorf("创建更新临时目录失败：%w", err)
	}
	installerPath := filepath.Join(tempDir, fileName)

	actualHash, err := downloadUpdateFile(ctx, installerURL.String(), installerPath)
	if err != nil {
		return "", err
	}
	if !strings.EqualFold(actualHash, expectedHash) {
		return "", fmt.Errorf("更新安装器 SHA256 校验失败，期望 %s，实际 %s", expectedHash, actualHash)
	}
	return installerPath, nil
}

func downloadUpdateChecksum(ctx context.Context, checksumURL string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, checksumURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建校验文件请求失败：%w", err)
	}
	resp, err := updateDownloadHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载校验文件失败：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("下载校验文件失败：HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, updateChecksumMaxBytes))
	if err != nil {
		return "", fmt.Errorf("读取校验文件失败：%w", err)
	}
	fields := strings.Fields(string(data))
	if len(fields) == 0 || !isSHA256Hex(fields[0]) {
		return "", fmt.Errorf("校验文件格式无效")
	}
	return strings.ToLower(fields[0]), nil
}

func downloadUpdateFile(ctx context.Context, downloadURL, targetPath string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	if err != nil {
		return "", fmt.Errorf("创建更新下载请求失败：%w", err)
	}
	resp, err := updateDownloadHTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("下载更新安装器失败：%w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("下载更新安装器失败：HTTP %d", resp.StatusCode)
	}
	file, err := os.OpenFile(targetPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return "", fmt.Errorf("创建更新安装器文件失败：%w", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(file, io.TeeReader(resp.Body, hasher)); err != nil {
		return "", fmt.Errorf("写入更新安装器失败：%w", err)
	}
	return hex.EncodeToString(hasher.Sum(nil)), nil
}

func safeInstallerFileName(installerURL *neturl.URL) (string, error) {
	fileName, err := neturl.PathUnescape(path.Base(installerURL.Path))
	if err != nil {
		return "", fmt.Errorf("解析更新安装器文件名失败：%w", err)
	}
	fileName = filepath.Base(fileName)
	if fileName == "." || fileName == string(filepath.Separator) || !strings.HasSuffix(strings.ToLower(fileName), ".exe") {
		return "", fmt.Errorf("更新安装器文件名无效")
	}
	return fileName, nil
}

func isSHA256Hex(value string) bool {
	if len(value) != sha256.Size*2 {
		return false
	}
	_, err := hex.DecodeString(value)
	return err == nil
}

func startUpdateInstaller(installerPath string) error {
	cmd := exec.Command(installerPath)
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动更新安装器失败：%w", err)
	}
	if cmd.Process != nil {
		return cmd.Process.Release()
	}
	return nil
}

func (a *App) recordUpdateCheckFailure(message string) {
	a.appendActivityLog(activityLog{
		Level:      activityLogLevelWarn,
		Category:   activityLogCategoryOperation,
		Action:     activityActionUpdate,
		TargetType: activityTargetRuntime,
		Title:      "更新检查失败",
		Message:    message,
	})
}

func (a *App) recordUpdateInstallFailure(message string) {
	a.appendActivityLog(activityLog{
		Level:      activityLogLevelWarn,
		Category:   activityLogCategoryOperation,
		Action:     activityActionUpdate,
		TargetType: activityTargetRuntime,
		Title:      "更新安装未启动",
		Message:    message,
	})
}

func sanitizeDiagnosticsText(value string) string {
	replacer := strings.NewReplacer(
		"relay_token", "relay-token-masked",
		"control_plane_token", "control-plane-token-masked",
	)
	sanitized := replacer.Replace(value)
	// 诊断文本可直接贴到 issue，因此统一遮蔽常见凭据形态。
	sanitizers := []struct {
		pattern     *regexp.Regexp
		replacement string
	}{
		{regexp.MustCompile(`(?i)(bearer\s+)[A-Za-z0-9._~+/=-]+`), `${1}<redacted>`},
		{regexp.MustCompile(`(?i)((?:relay-token-masked|control-plane-token-masked)\s*[:=]\s*)\S+`), `${1}<redacted>`},
		{regexp.MustCompile(`(?i)((?:token|secret|password|key)\s*[:=]\s*)\S+`), `${1}<redacted>`},
		{regexp.MustCompile(`(?i)((?:relay|control[-_]?plane|auth|access|refresh)[-_]?(?:token|secret|password|key)\s*[:=]\s*)\S+`), `${1}<redacted>`},
		{regexp.MustCompile(`([A-Za-z][A-Za-z0-9+.-]*://[^\s/@:]+:)[^\s/@]+(@)`), `${1}<redacted>${2}`},
	}
	for _, sanitizer := range sanitizers {
		sanitized = sanitizer.pattern.ReplaceAllString(sanitized, sanitizer.replacement)
	}
	return sanitized
}
