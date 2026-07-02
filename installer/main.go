package main

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"log"
	"os"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

//go:embed all:frontend/dist
var embeddedFrontend embed.FS

func main() {
	commandOptions, err := ParseCommandLine(os.Args[1:])
	if err != nil {
		newPlatformIntegration().ShowFatalMessage("NexTunnel 安装器", err.Error())
		os.Exit(64)
	}
	if commandOptions.ShowVersion {
		fmt.Println(AppVersion)
		return
	}

	platform := newPlatformIntegration()
	if !platform.IsElevated() {
		if err := platform.RelaunchElevated(commandOptions.OriginalArgs); err != nil {
			platform.ShowFatalMessage("NexTunnel 安装器", "安装器需要管理员权限，提权启动失败："+err.Error())
			os.Exit(1)
		}
		return
	}
	if commandOptions.Mode != commandModeGUI {
		os.Exit(runHeadless(commandOptions))
	}

	app := NewApp()
	assets, err := fs.Sub(embeddedFrontend, "frontend/dist")
	if err != nil {
		platform.ShowFatalMessage("NexTunnel 安装器", "安装器前端资源缺失，请重新构建安装器："+err.Error())
		os.Exit(1)
	}
	if _, err := fs.Stat(assets, "index.html"); err != nil {
		platform.ShowFatalMessage("NexTunnel 安装器", "安装器前端入口缺失，请先构建 installer 前端："+err.Error())
		os.Exit(1)
	}
	err = wails.Run(&options.App{
		Title:         "NexTunnel 安装器",
		Width:         980,
		Height:        620,
		MinWidth:      980,
		MinHeight:     620,
		Frameless:     true,
		DisableResize: true,
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		BackgroundColour: &options.RGBA{R: 5, G: 8, B: 20, A: 255},
		CSSDragProperty:  "--wails-draggable",
		CSSDragValue:     "drag",
		OnStartup:        app.startup,
		OnBeforeClose:    app.beforeClose,
		Bind: []interface{}{
			app,
		},
	})
	if err != nil {
		log.Fatal("Error:", err.Error())
	}
}

func runHeadless(commandOptions CommandOptions) int {
	installer := NewInstaller(newEmbeddedPayloadSource())
	report := func(progress InstallProgress) {
		if commandOptions.LogPath != "" {
			appendLogLine(commandOptions.LogPath, fmt.Sprintf("%s %d%% %s %s", progress.Phase, progress.Percent, progress.Message, progress.Error))
		}
	}
	var result InstallResult
	switch commandOptions.Mode {
	case commandModeUninstall:
		result = installer.Uninstall(context.Background(), commandOptions.Install.InstallDir, report)
	case commandModeRepair, commandModeInstall:
		result = installer.Install(context.Background(), commandOptions.Install, report)
	default:
		result = failedResult(fmt.Errorf("不支持的命令模式：%s", commandOptions.Mode))
	}
	if !result.Success {
		if commandOptions.LogPath != "" {
			appendLogLine(commandOptions.LogPath, "ERROR "+result.Error)
		}
		return 1
	}
	return 0
}

func appendLogLine(path string, line string) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return
	}
	defer file.Close()
	_, _ = file.WriteString(line + "\n")
}
