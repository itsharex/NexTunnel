package main

import (
	"fmt"
	"strings"
)

type CommandMode string

const (
	commandModeGUI       CommandMode = "gui"
	commandModeInstall   CommandMode = "install"
	commandModeRepair    CommandMode = "repair"
	commandModeUninstall CommandMode = "uninstall"
)

type CommandOptions struct {
	Mode         CommandMode
	Install      InstallOptions
	LogPath      string
	ShowVersion  bool
	OriginalArgs []string
}

func ParseCommandLine(args []string) (CommandOptions, error) {
	options := CommandOptions{
		Mode: commandModeGUI,
		Install: InstallOptions{
			CreateDesktopShortcut:   true,
			CreateStartMenuShortcut: true,
			LaunchAfterInstall:      true,
		},
		OriginalArgs: append([]string(nil), args...),
	}
	for index := 0; index < len(args); index++ {
		arg := strings.TrimSpace(args[index])
		switch arg {
		case "":
			continue
		case "--silent":
			if options.Mode == commandModeGUI {
				options.Mode = commandModeInstall
			}
		case "--repair":
			options.Mode = commandModeRepair
		case "--uninstall":
			options.Mode = commandModeUninstall
		case "--install-dir":
			index++
			if index >= len(args) {
				return options, fmt.Errorf("--install-dir 需要参数")
			}
			options.Install.InstallDir = args[index]
		case "--log":
			index++
			if index >= len(args) {
				return options, fmt.Errorf("--log 需要参数")
			}
			options.LogPath = args[index]
		case "--no-launch":
			options.Install.LaunchAfterInstall = false
		case "--no-desktop-shortcut":
			options.Install.CreateDesktopShortcut = false
		case "--no-start-menu-shortcut":
			options.Install.CreateStartMenuShortcut = false
		case "--version":
			options.ShowVersion = true
		default:
			return options, fmt.Errorf("未知参数：%s", arg)
		}
	}
	return options, nil
}
