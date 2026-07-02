package main

type UninstallInfo struct {
	DisplayName     string
	DisplayVersion  string
	Publisher       string
	InstallDir      string
	ExecutablePath  string
	UninstallerPath string
	EstimatedSizeKB uint64
}

type ShortcutOptions struct {
	Name        string
	TargetPath  string
	WorkingDir  string
	Description string
	Desktop     bool
	StartMenu   bool
}

type PlatformIntegration interface {
	DefaultInstallDir() string
	IsElevated() bool
	WebView2Ready() bool
	RelaunchElevated(args []string) error
	StopProcess(executableName string) error
	WriteUninstallInfo(info UninstallInfo) error
	RemoveUninstallInfo() error
	CreateShortcuts(options ShortcutOptions) error
	RemoveShortcuts(name string) error
	Launch(path string) error
	RemoveInstallDir(path string, selfPath string) error
	ShowFatalMessage(title string, message string)
}
