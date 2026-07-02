package main

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

type App struct {
	ctx       context.Context
	installer *Installer
	cancelMu  sync.Mutex
	cancel    context.CancelFunc
}

func NewApp() *App {
	return &App{installer: NewInstaller(newEmbeddedPayloadSource())}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

func (a *App) beforeClose(_ context.Context) bool {
	a.cancelMu.Lock()
	defer a.cancelMu.Unlock()
	if a.cancel == nil {
		return false
	}
	// 系统关闭事件也必须先请求取消，避免安装中途直接退出。
	a.cancel()
	return true
}

func (a *App) GetInstallPlan() InstallPlan {
	return a.installer.Plan()
}

func (a *App) StartInstall(options InstallOptions) InstallResult {
	ctx, cancel := context.WithCancel(context.Background())
	a.setCancel(cancel)
	defer a.clearCancel()
	return a.installer.Install(ctx, options, a.emitProgress)
}

func (a *App) CancelInstall() {
	a.cancelMu.Lock()
	defer a.cancelMu.Unlock()
	if a.cancel != nil {
		a.cancel()
	}
}

func (a *App) SelectInstallDir(currentDir string) string {
	if a.ctx == nil {
		return ""
	}
	defaultDir := existingDialogDirectory(currentDir)
	if defaultDir == "" && a.installer != nil {
		defaultDir = existingDialogDirectory(a.installer.platform.DefaultInstallDir())
	}
	selectedDir, err := runtime.OpenDirectoryDialog(a.ctx, runtime.OpenDialogOptions{
		Title:            "选择 NexTunnel 安装目录",
		DefaultDirectory: defaultDir,
	})
	if err != nil {
		return ""
	}
	return selectedDir
}

func (a *App) StartUninstall() InstallResult {
	ctx, cancel := context.WithCancel(context.Background())
	a.setCancel(cancel)
	defer a.clearCancel()
	return a.installer.Uninstall(ctx, "", a.emitProgress)
}

func (a *App) setCancel(cancel context.CancelFunc) {
	a.cancelMu.Lock()
	defer a.cancelMu.Unlock()
	a.cancel = cancel
}

func (a *App) clearCancel() {
	a.cancelMu.Lock()
	defer a.cancelMu.Unlock()
	a.cancel = nil
}

func (a *App) emitProgress(progress InstallProgress) {
	if a.ctx != nil {
		runtime.EventsEmit(a.ctx, progressEventName, progress)
	}
}

func existingDialogDirectory(rawPath string) string {
	path := strings.TrimSpace(rawPath)
	for path != "" {
		cleanPath := filepath.Clean(path)
		info, err := os.Stat(cleanPath)
		if err == nil && info.IsDir() {
			return cleanPath
		}
		parentPath := filepath.Dir(cleanPath)
		if parentPath == "." || parentPath == cleanPath {
			return ""
		}
		// Wails 要求 DefaultDirectory 已存在，因此安装目标不存在时回退到父目录。
		path = parentPath
	}
	return ""
}
