package command

import (
	"fmt"
	"os/exec"
	"runtime"

	"github.com/nextunnel/cli/internal/desktop"
	"github.com/spf13/cobra"
)

func newDesktopCommand(outputFormat *string) *cobra.Command {
	var controlFile string
	cmd := &cobra.Command{
		Use:   "desktop",
		Short: "管理本机 NexTunnel 桌面端",
	}
	cmd.PersistentFlags().StringVar(&controlFile, "control-file", "", "桌面端控制文件路径")
	cmd.AddCommand(
		newDesktopOpenCommand(),
		newDesktopStatusCommand(outputFormat, &controlFile),
		newDesktopConnectCommand(outputFormat, &controlFile),
		newDesktopDisconnectCommand(outputFormat, &controlFile),
		newDesktopNATCommand(outputFormat, &controlFile),
		newDesktopNetworkCommand(outputFormat, &controlFile),
		newDesktopSettingsCommand(outputFormat, &controlFile),
	)
	return cmd
}

func newDesktopOpenCommand() *cobra.Command {
	var binary string
	command := &cobra.Command{
		Use:   "open",
		Short: "启动桌面端应用",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if binary == "" {
				binary = defaultDesktopBinary()
			}
			if binary == "" {
				return fmt.Errorf("未找到桌面端可执行文件，请使用 --binary 指定路径")
			}
			start := exec.Command(binary)
			if err := start.Start(); err != nil {
				return err
			}
			_, err := fmt.Fprintf(commandOutput(cmd), "桌面端已启动 pid=%d\n", start.Process.Pid)
			return err
		},
	}
	command.Flags().StringVar(&binary, "binary", "", "桌面端可执行文件路径")
	return command
}

func newDesktopStatusCommand(outputFormat *string, controlFile *string) *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "查看桌面端运行状态",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := desktop.NewClient(*controlFile)
			if err != nil {
				return err
			}
			var status map[string]any
			if err := client.Get("/api/v1/status", &status); err != nil {
				return err
			}
			return writeData(commandOutput(cmd), *outputFormat, status)
		},
	}
}

func newDesktopConnectCommand(outputFormat *string, controlFile *string) *cobra.Command {
	var relay string
	var token string
	command := &cobra.Command{
		Use:   "connect",
		Short: "连接 Relay",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if relay == "" {
				return fmt.Errorf("--relay 必填")
			}
			client, err := desktop.NewClient(*controlFile)
			if err != nil {
				return err
			}
			var status map[string]any
			if err := client.Post("/api/v1/connect", map[string]string{"server_addr": relay, "auth_token": token}, &status); err != nil {
				return err
			}
			return writeData(commandOutput(cmd), *outputFormat, status)
		},
	}
	command.Flags().StringVar(&relay, "relay", "", "Relay 地址，例如 127.0.0.1:7000")
	command.Flags().StringVar(&token, "token", "", "Relay 认证 token")
	return command
}

func newDesktopDisconnectCommand(outputFormat *string, controlFile *string) *cobra.Command {
	return &cobra.Command{
		Use:   "disconnect",
		Short: "断开 Relay",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := desktop.NewClient(*controlFile)
			if err != nil {
				return err
			}
			var status map[string]any
			if err := client.Post("/api/v1/disconnect", map[string]string{}, &status); err != nil {
				return err
			}
			return writeData(commandOutput(cmd), *outputFormat, status)
		},
	}
}

func newDesktopNATCommand(outputFormat *string, controlFile *string) *cobra.Command {
	cmd := &cobra.Command{Use: "nat", Short: "桌面端 NAT 诊断"}
	cmd.AddCommand(&cobra.Command{
		Use:   "detect",
		Short: "触发 NAT 检测",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := desktop.NewClient(*controlFile)
			if err != nil {
				return err
			}
			var result map[string]any
			if err := client.Post("/api/v1/nat/detect", map[string]string{}, &result); err != nil {
				return err
			}
			return writeData(commandOutput(cmd), *outputFormat, result)
		},
	})
	return cmd
}

func newDesktopNetworkCommand(outputFormat *string, controlFile *string) *cobra.Command {
	cmd := &cobra.Command{Use: "network", Short: "桌面端虚拟网络控制"}
	cmd.AddCommand(desktopPostCommand("apply", "应用虚拟网络路由", "/api/v1/network/apply", outputFormat, controlFile))
	cmd.AddCommand(desktopPostCommand("reset", "回滚虚拟网络路由", "/api/v1/network/reset", outputFormat, controlFile))
	return cmd
}

func newDesktopSettingsCommand(outputFormat *string, controlFile *string) *cobra.Command {
	var relay string
	var relayToken string
	var controlPlane string
	var controlToken string
	var stun string
	cmd := &cobra.Command{Use: "settings", Short: "桌面端连接设置"}
	cmd.AddCommand(&cobra.Command{
		Use:   "get",
		Short: "读取设置",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := desktop.NewClient(*controlFile)
			if err != nil {
				return err
			}
			var settings map[string]any
			if err := client.Get("/api/v1/settings", &settings); err != nil {
				return err
			}
			return writeData(commandOutput(cmd), *outputFormat, settings)
		},
	})
	set := &cobra.Command{
		Use:   "set",
		Short: "保存设置",
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := desktop.NewClient(*controlFile)
			if err != nil {
				return err
			}
			payload := map[string]string{}
			if relay != "" {
				payload["relay_addr"] = relay
			}
			if relayToken != "" {
				payload["relay_token"] = relayToken
			}
			if controlPlane != "" {
				payload["control_plane_url"] = controlPlane
			}
			if controlToken != "" {
				payload["control_plane_token"] = controlToken
			}
			if stun != "" {
				payload["stun_server"] = stun
				payload["stun_alt_server"] = stun
			}
			var result map[string]any
			if err := client.Post("/api/v1/settings", payload, &result); err != nil {
				return err
			}
			return writeData(commandOutput(cmd), *outputFormat, result)
		},
	}
	set.Flags().StringVar(&relay, "relay", "", "Relay 地址")
	set.Flags().StringVar(&relayToken, "relay-token", "", "Relay token")
	set.Flags().StringVar(&controlPlane, "control-plane", "", "Control Plane URL")
	set.Flags().StringVar(&controlToken, "control-token", "", "Control Plane token")
	set.Flags().StringVar(&stun, "stun", "", "STUN 服务器")
	cmd.AddCommand(set)
	return cmd
}

func desktopPostCommand(use, short, path string, outputFormat *string, controlFile *string) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		RunE: func(cmd *cobra.Command, _ []string) error {
			client, err := desktop.NewClient(*controlFile)
			if err != nil {
				return err
			}
			var result map[string]any
			if err := client.Post(path, map[string]string{}, &result); err != nil {
				return err
			}
			return writeData(commandOutput(cmd), *outputFormat, result)
		},
	}
}

func defaultDesktopBinary() string {
	if runtime.GOOS == "windows" {
		return ""
	}
	return "nextunnel"
}
