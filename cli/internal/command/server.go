package command

import (
	"fmt"
	"runtime"

	"github.com/nextunnel/cli/internal/system"
	"github.com/spf13/cobra"
)

func newServerCommand(outputFormat *string) *cobra.Command {
	var paths system.Paths
	paths = system.DefaultPaths()
	cmd := &cobra.Command{
		Use:   "server",
		Short: "管理本机 NexTunnel 服务端",
	}
	cmd.PersistentFlags().StringVar(&paths.InstallDir, "install-dir", paths.InstallDir, "服务端安装目录")
	cmd.PersistentFlags().StringVar(&paths.ConfigDir, "config-dir", paths.ConfigDir, "服务端配置目录")
	cmd.PersistentFlags().StringVar(&paths.DataDir, "data-dir", paths.DataDir, "服务端数据目录")
	cmd.PersistentFlags().StringVar(&paths.BinDir, "bin-dir", paths.BinDir, "服务端二进制目录")
	cmd.PersistentFlags().StringVar(&paths.LogDir, "log-dir", paths.LogDir, "服务端日志目录")
	cmd.PersistentFlags().StringVar(&paths.RunDir, "run-dir", paths.RunDir, "服务端运行态目录")
	cmd.PersistentFlags().StringVar(&paths.EnvPath, "env-file", paths.EnvPath, "server.env 路径")
	cmd.AddCommand(
		newServerInstallerCommand("install", "安装并启动服务端"),
		serverActionCommand("up", "启动本机服务端", func() error { return system.StartServer(paths) }),
		serverActionCommand("down", "停止本机服务端", func() error { return system.StopServer(paths) }),
		serverActionCommand("restart", "重启本机服务端", func() error { return system.RestartServer(paths) }),
		newServerInstallerCommand("update", "更新服务端发布包"),
		newServerInstallerCommand("uninstall", "卸载服务端"),
		newServerStatusCommand(outputFormat, &paths),
		newServerHealthCommand(outputFormat, &paths),
		newServerLogsCommand(&paths),
		newServerPathsCommand(outputFormat, &paths),
	)
	return cmd
}

func newServerInstallerCommand(action, short string) *cobra.Command {
	var version string
	var packageURL string
	var packageSHA256 string
	var nonInteractive bool
	var force bool
	var purge bool
	command := &cobra.Command{
		Use:   action,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			args := []string{}
			if version != "" {
				args = appendInstallerFlag(args, "version", version, "Version")
			}
			if packageURL != "" {
				args = appendInstallerFlag(args, "package-url", packageURL, "PackageUrl")
			}
			if packageSHA256 != "" {
				args = appendInstallerFlag(args, "sha256", packageSHA256, "PackageSha256")
			}
			if nonInteractive {
				args = appendBooleanInstallerFlag(args, "non-interactive", "NonInteractive")
			}
			if force {
				args = appendBooleanInstallerFlag(args, "force", "Force")
			}
			if purge {
				args = appendBooleanInstallerFlag(args, "purge", "Purge")
			}
			if err := system.RunInstaller(action, args); err != nil {
				return err
			}
			_, err := fmt.Fprintf(commandOutput(cmd), "%s 完成\n", short)
			return err
		},
	}
	command.Flags().StringVar(&version, "version", "", "Release 版本，例如 v0.2.1-alpha")
	command.Flags().StringVar(&packageURL, "package-url", "", "服务端发布包 URL 或本地路径")
	command.Flags().StringVar(&packageSHA256, "sha256", "", "发布包 SHA256")
	command.Flags().BoolVar(&nonInteractive, "non-interactive", false, "非交互执行")
	command.Flags().BoolVar(&force, "force", false, "强制重新生成配置")
	command.Flags().BoolVar(&purge, "purge", false, "卸载时删除配置和数据")
	return command
}

func appendInstallerFlag(args []string, bashName, powerShellName, value string) []string {
	if runtime.GOOS == "windows" {
		return append(args, "-"+powerShellName, value)
	}
	return append(args, "--"+bashName, value)
}

func appendBooleanInstallerFlag(args []string, bashName, powerShellName string) []string {
	if runtime.GOOS == "windows" {
		return append(args, "-"+powerShellName)
	}
	return append(args, "--"+bashName)
}

func serverActionCommand(use, short string, run func() error) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			if err := run(); err != nil {
				return err
			}
			_, err := fmt.Fprintf(commandOutput(cmd), "%s 完成\n", short)
			return err
		},
	}
}

func newServerStatusCommand(outputFormat *string, paths *system.Paths) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "查看本机服务端进程状态",
		RunE: func(cmd *cobra.Command, _ []string) error {
			status, err := system.ServerStatus(*paths)
			if err != nil {
				return err
			}
			return writeData(commandOutput(cmd), *outputFormat, status)
		},
	}
}

func newServerHealthCommand(outputFormat *string, paths *system.Paths) *cobra.Command {
	return &cobra.Command{
		Use:   "health",
		Short: "检查本机服务端健康状态",
		RunE: func(cmd *cobra.Command, _ []string) error {
			result, err := system.Health(*paths)
			if err != nil {
				return err
			}
			return writeData(commandOutput(cmd), *outputFormat, result)
		},
	}
}

func newServerLogsCommand(paths *system.Paths) *cobra.Command {
	var follow bool
	cmd := &cobra.Command{
		Use:   "logs",
		Short: "查看本机服务端日志",
		RunE: func(_ *cobra.Command, _ []string) error {
			return system.TailServerLogs(*paths, follow)
		},
	}
	cmd.Flags().BoolVarP(&follow, "follow", "f", false, "持续跟随日志")
	return cmd
}

func newServerPathsCommand(outputFormat *string, paths *system.Paths) *cobra.Command {
	return &cobra.Command{
		Use:   "paths",
		Short: "显示服务端默认路径",
		RunE: func(cmd *cobra.Command, _ []string) error {
			return writeData(commandOutput(cmd), *outputFormat, paths)
		},
	}
}
