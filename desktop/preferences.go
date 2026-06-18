package main

import (
	"encoding/json"
	"fmt"
	"strings"
)

const (
	settingThemeMode      = "appearance_theme_mode"
	settingMotionLevel    = "appearance_motion_level"
	settingLanguage       = "appearance_language"
	settingAccentColor    = "appearance_accent_color"
	settingAutoConnect    = "general_auto_connect"
	settingMinimizeToTray = "general_minimize_to_tray"
	settingStartMinimized = "general_start_minimized"
	settingIncludeTokens  = "general_export_include_tokens"
	defaultThemeMode      = "dark"
	defaultMotionLevel    = "normal"
	defaultLanguage       = "zh-CN"
	defaultAccentColor    = "#00ffff"
	activityActionImport  = "import_config"
	activityActionExport  = "export_config"
	activityActionPrefs   = "save_preferences"
	activityActionAutoRun = "set_autostart"
	activityActionUpdate  = "check_update"
	activityActionDiag    = "collect_diagnostics"
	configExportVersion   = 1
	maskedSensitiveValue  = "***"
)

// AppearanceSettings 保存桌面端外观偏好。
type AppearanceSettings struct {
	ThemeMode   string `json:"theme_mode"`
	MotionLevel string `json:"motion_level"`
	Language    string `json:"language"`
	AccentColor string `json:"accent_color"`
}

// GeneralSettings 保存桌面端通用行为偏好。
type GeneralSettings struct {
	AutoConnect         bool `json:"auto_connect"`
	MinimizeToTray      bool `json:"minimize_to_tray"`
	StartMinimized      bool `json:"start_minimized"`
	ExportIncludeTokens bool `json:"export_include_tokens"`
	TraySupported       bool `json:"tray_supported"`
}

// ExportConfigOptions 控制导出配置是否包含敏感字段。
type ExportConfigOptions struct {
	IncludeSensitive bool `json:"include_sensitive"`
}

type exportedDesktopConfig struct {
	Version       int                `json:"version"`
	Server        ServerSettings     `json:"server"`
	Appearance    AppearanceSettings `json:"appearance"`
	General       GeneralSettings    `json:"general"`
	Tunnels       []TunnelInfo       `json:"tunnels"`
	FavoritePorts []FavoritePortInfo `json:"favorite_ports"`
	Settings      map[string]string  `json:"settings,omitempty"`
}

// GetAppearanceSettings 返回可持久化的外观设置。
func (a *App) GetAppearanceSettings() AppearanceSettings {
	return AppearanceSettings{
		ThemeMode:   normalizeThemeMode(a.settingOrDefault(settingThemeMode, defaultThemeMode)),
		MotionLevel: normalizeMotionLevel(a.settingOrDefault(settingMotionLevel, defaultMotionLevel)),
		Language:    normalizeLanguage(a.settingOrDefault(settingLanguage, defaultLanguage)),
		AccentColor: normalizeAccentColor(a.settingOrDefault(settingAccentColor, defaultAccentColor)),
	}
}

// SaveAppearanceSettings 持久化外观设置，非法值回退为默认值。
func (a *App) SaveAppearanceSettings(settings AppearanceSettings) error {
	values := map[string]string{
		settingThemeMode:   normalizeThemeMode(settings.ThemeMode),
		settingMotionLevel: normalizeMotionLevel(settings.MotionLevel),
		settingLanguage:    normalizeLanguage(settings.Language),
		settingAccentColor: normalizeAccentColor(settings.AccentColor),
	}
	if err := a.saveSettingsMap(values); err != nil {
		return err
	}
	a.appendActivityLog(activityLog{
		Level:      activityLogLevelInfo,
		Category:   activityLogCategoryOperation,
		Action:     activityActionPrefs,
		TargetType: activityTargetSettings,
		Title:      "外观设置已保存",
		Message:    "主题、语言和动效偏好已持久化。",
		Metadata:   values,
	})
	return nil
}

// GetGeneralSettings 返回通用桌面行为偏好。
func (a *App) GetGeneralSettings() GeneralSettings {
	return GeneralSettings{
		AutoConnect:         parseBoolSetting(a.settingOrDefault(settingAutoConnect, "false")),
		MinimizeToTray:      parseBoolSetting(a.settingOrDefault(settingMinimizeToTray, "false")),
		StartMinimized:      parseBoolSetting(a.settingOrDefault(settingStartMinimized, "false")),
		ExportIncludeTokens: parseBoolSetting(a.settingOrDefault(settingIncludeTokens, "false")),
		TraySupported:       false,
	}
}

// SaveGeneralSettings 持久化通用桌面行为偏好。
func (a *App) SaveGeneralSettings(settings GeneralSettings) error {
	values := map[string]string{
		settingAutoConnect:    fmt.Sprintf("%t", settings.AutoConnect),
		settingMinimizeToTray: fmt.Sprintf("%t", settings.MinimizeToTray),
		settingStartMinimized: fmt.Sprintf("%t", settings.StartMinimized),
		settingIncludeTokens:  fmt.Sprintf("%t", settings.ExportIncludeTokens),
	}
	if err := a.saveSettingsMap(values); err != nil {
		return err
	}
	a.appendActivityLog(activityLog{
		Level:      activityLogLevelInfo,
		Category:   activityLogCategoryOperation,
		Action:     activityActionPrefs,
		TargetType: activityTargetSettings,
		Title:      "通用设置已保存",
		Message:    "自动连接、托盘和导出偏好已持久化。",
		Metadata:   values,
	})
	return nil
}

// ExportConfig 导出当前桌面端配置，默认脱敏 token。
func (a *App) ExportConfig(options ExportConfigOptions) (string, error) {
	if a.store == nil {
		return "", fmt.Errorf("config store is not ready")
	}
	tunnels, err := a.GetTunnels()
	if err != nil {
		return "", err
	}
	ports, err := a.ListFavoritePorts()
	if err != nil {
		return "", err
	}
	rawSettings, err := a.store.ListSettings()
	if err != nil {
		return "", err
	}

	serverSettings := a.GetServerSettings()
	if !options.IncludeSensitive {
		serverSettings.RelayToken = ""
		serverSettings.ControlPlaneToken = ""
		delete(rawSettings, settingRelayToken)
		delete(rawSettings, settingControlPlaneToken)
	}

	payload := exportedDesktopConfig{
		Version:       configExportVersion,
		Server:        serverSettings,
		Appearance:    a.GetAppearanceSettings(),
		General:       a.GetGeneralSettings(),
		Tunnels:       tunnels,
		FavoritePorts: ports,
		Settings:      rawSettings,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshal config export: %w", err)
	}
	a.appendActivityLog(activityLog{
		Level:      activityLogLevelInfo,
		Category:   activityLogCategorySecurity,
		Action:     activityActionExport,
		TargetType: activityTargetSettings,
		Title:      "配置已导出",
		Message:    "桌面端配置已序列化为 JSON。",
		Metadata: map[string]string{
			"include_sensitive": fmt.Sprintf("%t", options.IncludeSensitive),
			"tunnels":           fmt.Sprintf("%d", len(tunnels)),
			"favorite_ports":    fmt.Sprintf("%d", len(ports)),
		},
	})
	return string(data), nil
}

// ImportConfig 导入 JSON 配置，并复用现有校验逻辑保证数据稳健。
func (a *App) ImportConfig(data string) error {
	if a.store == nil {
		return fmt.Errorf("config store is not ready")
	}
	var payload exportedDesktopConfig
	if err := json.Unmarshal([]byte(data), &payload); err != nil {
		return fmt.Errorf("decode config import: %w", err)
	}
	if payload.Version <= 0 || payload.Version > configExportVersion {
		return fmt.Errorf("unsupported config version: %d", payload.Version)
	}
	// 默认导出会脱敏 token；导回时保留本机已有敏感字段，避免误清空连接凭据。
	currentServerSettings := a.GetServerSettings()
	if strings.TrimSpace(payload.Server.RelayToken) == "" {
		payload.Server.RelayToken = currentServerSettings.RelayToken
	}
	if strings.TrimSpace(payload.Server.ControlPlaneToken) == "" {
		payload.Server.ControlPlaneToken = currentServerSettings.ControlPlaneToken
	}
	if err := a.SaveServerSettings(payload.Server); err != nil {
		return err
	}
	if err := a.SaveAppearanceSettings(payload.Appearance); err != nil {
		return err
	}
	if err := a.SaveGeneralSettings(payload.General); err != nil {
		return err
	}
	for _, tunnelInfo := range payload.Tunnels {
		if err := validateCreateTunnelInput(CreateTunnelInput{
			Name:       tunnelInfo.Name,
			ProxyType:  tunnelInfo.ProxyType,
			LocalAddr:  tunnelInfo.LocalAddr,
			LocalPort:  tunnelInfo.LocalPort,
			RemotePort: tunnelInfo.RemotePort,
		}); err != nil {
			return fmt.Errorf("validate tunnel %q: %w", tunnelInfo.Name, err)
		}
		existing, err := a.store.GetByName(tunnelInfo.Name)
		if err != nil {
			return err
		}
		if existing != nil {
			existing.ProxyType = tunnelInfo.ProxyType
			existing.LocalAddr = tunnelInfo.LocalAddr
			existing.LocalPort = tunnelInfo.LocalPort
			existing.RemotePort = tunnelInfo.RemotePort
			if err := a.store.Update(existing); err != nil {
				return err
			}
			continue
		}
		if _, err := a.CreateTunnel(CreateTunnelInput{
			Name:       tunnelInfo.Name,
			ProxyType:  tunnelInfo.ProxyType,
			LocalAddr:  tunnelInfo.LocalAddr,
			LocalPort:  tunnelInfo.LocalPort,
			RemotePort: tunnelInfo.RemotePort,
		}); err != nil {
			return err
		}
	}
	for _, port := range payload.FavoritePorts {
		if _, err := a.SaveFavoritePort(FavoritePortInput{
			ID:          port.ID,
			Name:        port.Name,
			Category:    port.Category,
			Port:        port.Port,
			Protocol:    port.Protocol,
			Description: port.Description,
			Enabled:     port.Enabled,
		}); err != nil {
			return err
		}
	}
	a.appendActivityLog(activityLog{
		Level:      activityLogLevelInfo,
		Category:   activityLogCategorySecurity,
		Action:     activityActionImport,
		TargetType: activityTargetSettings,
		Title:      "配置已导入",
		Message:    "桌面端配置已从 JSON 导入。",
		Metadata: map[string]string{
			"tunnels":        fmt.Sprintf("%d", len(payload.Tunnels)),
			"favorite_ports": fmt.Sprintf("%d", len(payload.FavoritePorts)),
		},
	})
	return nil
}

func (a *App) saveSettingsMap(values map[string]string) error {
	if a.store == nil {
		err := fmt.Errorf("config store is not ready")
		a.recordError(err)
		return err
	}
	for key, value := range values {
		if err := a.store.SetSetting(key, value); err != nil {
			a.recordError(err)
			return fmt.Errorf("save setting %s: %w", key, err)
		}
	}
	a.clearError()
	return nil
}

func parseBoolSetting(value string) bool {
	return strings.EqualFold(strings.TrimSpace(value), "true")
}

func normalizeThemeMode(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "system", "light", "dark":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return defaultThemeMode
	}
}

func normalizeMotionLevel(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "normal", "reduced":
		return strings.ToLower(strings.TrimSpace(value))
	default:
		return defaultMotionLevel
	}
}

func normalizeLanguage(value string) string {
	switch strings.TrimSpace(value) {
	case "zh-CN", "en-US":
		return strings.TrimSpace(value)
	default:
		return defaultLanguage
	}
}

func normalizeAccentColor(value string) string {
	trimmed := strings.TrimSpace(value)
	if len(trimmed) != 7 || !strings.HasPrefix(trimmed, "#") {
		return defaultAccentColor
	}
	for _, r := range trimmed[1:] {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')) {
			return defaultAccentColor
		}
	}
	return trimmed
}

func maskSensitiveValue(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	return maskedSensitiveValue
}
