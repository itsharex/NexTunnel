package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type Installer struct {
	source   PayloadSource
	platform PlatformIntegration
	selfPath string
	mu       sync.Mutex
}

func NewInstaller(source PayloadSource) *Installer {
	selfPath, _ := os.Executable()
	return &Installer{
		source:   source,
		platform: newPlatformIntegration(),
		selfPath: selfPath,
	}
}

func (i *Installer) Plan() InstallPlan {
	manifest, err := i.source.Manifest()
	plan := InstallPlan{
		Version:         AppVersion,
		Target:          "windows/amd64",
		DefaultDir:      i.platform.DefaultInstallDir(),
		AppExecutable:   appExecutableName,
		RequiresAdmin:   true,
		IsAdmin:         i.platform.IsElevated(),
		WebView2Ready:   i.platform.WebView2Ready(),
		WebView2Mode:    "embedded-bootstrapper",
		RequiredSpaceMB: defaultRequiredSpaceMB,
		WintunIncluded:  false,
		Signing:         "unsigned-alpha",
		PayloadReady:    false,
		PayloadSHA256:   "",
	}
	if err != nil {
		plan.Error = err.Error()
		return plan
	}
	plan.Version = manifest.Version
	plan.Target = manifest.Target
	plan.AppExecutable = manifest.AppExecutable
	plan.RequiredSpaceMB = manifest.RequiredSpaceMB
	plan.WintunIncluded = manifest.WintunIncluded
	plan.Signing = manifest.Signing
	plan.PayloadSHA256 = manifest.PayloadSHA256
	plan.PayloadReady = manifest.PayloadFile != "" && manifest.PayloadSHA256 != ""
	return plan
}

func (i *Installer) Install(ctx context.Context, options InstallOptions, report ProgressReporter) InstallResult {
	i.mu.Lock()
	defer i.mu.Unlock()

	manifest, err := i.source.Manifest()
	if err != nil {
		return failedResult(err)
	}
	options = i.normalizeOptions(options)
	if err := validateInstallDir(options.InstallDir); err != nil {
		return failedResult(err)
	}
	if !i.platform.IsElevated() {
		return failedResult(fmt.Errorf("安装 NexTunnel 需要管理员权限"))
	}

	reportProgress(report, installPhasePreparing, 4, "正在准备安装计划", "")
	payload, err := i.source.PayloadBytes(manifest)
	if err != nil {
		return failedResult(err)
	}
	reportProgress(report, installPhaseValidating, 12, "正在校验安装包完整性", "")
	if err := assertPayloadHash(payload, manifest.PayloadSHA256); err != nil {
		return failedResult(err)
	}
	if err := ctx.Err(); err != nil {
		return failedResult(err)
	}

	// 先解压到同卷 staging，后续用目录 rename 降低升级中断风险。
	parentDir := filepath.Dir(options.InstallDir)
	if err := os.MkdirAll(parentDir, 0o755); err != nil {
		return failedResult(fmt.Errorf("创建安装父目录：%w", err))
	}
	stagingDir, err := os.MkdirTemp(parentDir, ".nextunnel-staging-")
	if err != nil {
		return failedResult(fmt.Errorf("创建临时安装目录：%w", err))
	}
	backupDir := ""
	installCompleted := false
	defer func() {
		if !installCompleted {
			_ = os.RemoveAll(stagingDir)
		}
	}()

	reportProgress(report, installPhaseExtracting, 18, "正在展开应用文件", "")
	if err := safeExtractZip(payload, stagingDir, report); err != nil {
		return failedResult(err)
	}
	appPath := filepath.Join(options.InstallDir, manifest.AppExecutable)
	stagedAppPath := filepath.Join(stagingDir, manifest.AppExecutable)
	if _, err := os.Stat(stagedAppPath); err != nil {
		return failedResult(fmt.Errorf("payload 缺少主程序 %s: %w", manifest.AppExecutable, err))
	}
	if err := i.copySelfInstaller(stagingDir); err != nil {
		return failedResult(err)
	}
	if err := ctx.Err(); err != nil {
		return failedResult(err)
	}

	// 替换前保留旧目录，系统集成失败时也能回滚到上一版。
	reportProgress(report, installPhaseReplacing, 66, "正在替换旧版本", "")
	if err := i.platform.StopProcess(manifest.AppExecutable); err != nil {
		return failedResult(err)
	}
	if _, err := os.Stat(options.InstallDir); err == nil {
		backupDir = options.InstallDir + ".backup-" + time.Now().Format("20060102150405")
		if err := os.Rename(options.InstallDir, backupDir); err != nil {
			return failedResult(fmt.Errorf("备份旧版本目录：%w", err))
		}
	} else if err != nil && !os.IsNotExist(err) {
		return failedResult(fmt.Errorf("检查旧版本目录：%w", err))
	}
	if err := os.Rename(stagingDir, options.InstallDir); err != nil {
		i.rollbackInstall(options.InstallDir, backupDir, report)
		return failedResult(fmt.Errorf("替换安装目录：%w", err))
	}

	reportProgress(report, installPhaseIntegrating, 82, "正在写入系统集成信息", "")
	if err := i.writeSystemIntegration(options, manifest, appPath); err != nil {
		i.rollbackInstall(options.InstallDir, backupDir, report)
		return failedResult(err)
	}
	if backupDir != "" {
		_ = os.RemoveAll(backupDir)
	}
	installCompleted = true
	if options.LaunchAfterInstall {
		if err := i.platform.Launch(appPath); err != nil {
			return InstallResult{Success: true, Version: manifest.Version, AppPath: appPath, Error: fmt.Sprintf("安装完成，但启动失败：%v", err)}
		}
	}
	reportProgress(report, installPhaseComplete, 100, "NexTunnel 安装完成", "")
	return InstallResult{Success: true, Version: manifest.Version, AppPath: appPath}
}

func (i *Installer) Uninstall(ctx context.Context, installDir string, report ProgressReporter) InstallResult {
	i.mu.Lock()
	defer i.mu.Unlock()

	if strings.TrimSpace(installDir) == "" {
		installDir = i.platform.DefaultInstallDir()
	}
	if err := validateInstallDir(installDir); err != nil {
		return failedResult(err)
	}
	if !i.platform.IsElevated() {
		return failedResult(fmt.Errorf("卸载 NexTunnel 需要管理员权限"))
	}
	reportProgress(report, installPhaseUninstalling, 20, "正在停止 NexTunnel", "")
	_ = i.platform.StopProcess(appExecutableName)
	if err := ctx.Err(); err != nil {
		return failedResult(err)
	}
	reportProgress(report, installPhaseUninstalling, 55, "正在移除系统集成信息", "")
	if err := i.platform.RemoveShortcuts(appName); err != nil {
		return failedResult(err)
	}
	if err := i.platform.RemoveUninstallInfo(); err != nil {
		return failedResult(err)
	}
	reportProgress(report, installPhaseUninstalling, 78, "正在删除安装目录", "")
	if err := i.platform.RemoveInstallDir(installDir, i.selfPath); err != nil {
		return failedResult(err)
	}
	reportProgress(report, installPhaseComplete, 100, "NexTunnel 已卸载", "")
	return InstallResult{Success: true, Version: AppVersion, AppPath: filepath.Join(installDir, appExecutableName)}
}

func (i *Installer) normalizeOptions(options InstallOptions) InstallOptions {
	if strings.TrimSpace(options.InstallDir) == "" {
		options.InstallDir = i.platform.DefaultInstallDir()
	}
	return options
}

func (i *Installer) copySelfInstaller(stagingDir string) error {
	if i.selfPath == "" {
		return nil
	}
	targetPath := filepath.Join(stagingDir, installerExecutableName)
	source, err := os.Open(i.selfPath)
	if err != nil {
		return fmt.Errorf("读取安装器自身：%w", err)
	}
	defer source.Close()
	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("复制安装器自身：%w", err)
	}
	defer target.Close()
	if _, err := io.Copy(target, source); err != nil {
		return fmt.Errorf("复制安装器自身：%w", err)
	}
	return nil
}

func (i *Installer) writeSystemIntegration(options InstallOptions, manifest PayloadManifest, appPath string) error {
	sizeKB, _ := directorySizeKB(options.InstallDir)
	uninstallerPath := filepath.Join(options.InstallDir, installerExecutableName)
	if err := i.platform.WriteUninstallInfo(UninstallInfo{
		DisplayName:     appName,
		DisplayVersion:  manifest.Version,
		Publisher:       "NexTunnel Contributors",
		InstallDir:      options.InstallDir,
		ExecutablePath:  appPath,
		UninstallerPath: uninstallerPath,
		EstimatedSizeKB: sizeKB,
	}); err != nil {
		return err
	}
	return i.platform.CreateShortcuts(ShortcutOptions{
		Name:        appName,
		TargetPath:  appPath,
		WorkingDir:  options.InstallDir,
		Description: "NexTunnel desktop client",
		Desktop:     options.CreateDesktopShortcut,
		StartMenu:   options.CreateStartMenuShortcut,
	})
}

func (i *Installer) rollbackInstall(installDir string, backupDir string, report ProgressReporter) {
	reportProgress(report, installPhaseRollback, 90, "安装失败，正在回滚旧版本", "")
	_ = os.RemoveAll(installDir)
	if backupDir != "" {
		_ = os.Rename(backupDir, installDir)
	}
}

func validateInstallDir(installDir string) error {
	installDir = strings.TrimSpace(installDir)
	if installDir == "" {
		return fmt.Errorf("安装目录不能为空")
	}
	if !filepath.IsAbs(installDir) {
		return fmt.Errorf("安装目录必须是绝对路径")
	}
	cleanPath := filepath.Clean(installDir)
	root := filepath.VolumeName(cleanPath) + string(filepath.Separator)
	if cleanPath == root || cleanPath == filepath.VolumeName(cleanPath) {
		return fmt.Errorf("安装目录不能是磁盘根目录")
	}
	return nil
}

func directorySizeKB(root string) (uint64, error) {
	var total uint64
	err := filepath.WalkDir(root, func(_ string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		total += uint64(info.Size())
		return nil
	})
	if total == 0 {
		return 0, err
	}
	return (total + 1023) / 1024, err
}

func reportProgress(report ProgressReporter, phase string, percent int, message string, errorMessage string) {
	if report != nil {
		report(InstallProgress{Phase: phase, Percent: percent, Message: message, Error: errorMessage})
	}
}

func failedResult(err error) InstallResult {
	if err == nil {
		return InstallResult{Success: false, Version: AppVersion}
	}
	return InstallResult{Success: false, Version: AppVersion, Error: err.Error()}
}
