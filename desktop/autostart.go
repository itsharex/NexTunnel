package main

import "fmt"

// GetAutoStartEnabled 返回当前用户级开机自启状态；读取失败时记录错误并返回 false。
func (a *App) GetAutoStartEnabled() bool {
	enabled, err := getAutoStartEnabled()
	if err != nil {
		a.recordError(err)
		return false
	}
	return enabled
}

// SetAutoStartEnabled 设置当前用户级开机自启，不申请系统级权限。
func (a *App) SetAutoStartEnabled(enabled bool) error {
	if err := setAutoStartEnabled(enabled); err != nil {
		a.recordError(err)
		return err
	}
	a.appendActivityLog(activityLog{
		Level:      activityLogLevelInfo,
		Category:   activityLogCategorySecurity,
		Action:     activityActionAutoRun,
		TargetType: activityTargetSettings,
		Title:      "开机自启设置已更新",
		Message:    fmt.Sprintf("当前用户级开机自启已设置为 %t。", enabled),
		Metadata: map[string]string{
			"enabled": fmt.Sprintf("%t", enabled),
		},
	})
	return nil
}
