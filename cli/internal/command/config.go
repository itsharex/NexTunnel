package command

import (
	"fmt"

	"github.com/nextunnel/cli/internal/configstore"
	"github.com/spf13/cobra"
)

func newConfigCommand(outputFormat *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "管理 NexTunnel CLI 上下文配置",
	}
	cmd.AddCommand(
		newConfigPathCommand(outputFormat),
		newConfigSetContextCommand(),
		newConfigUseContextCommand(),
		newConfigCurrentContextCommand(outputFormat),
		newConfigListContextsCommand(outputFormat),
	)
	return cmd
}

func newConfigPathCommand(outputFormat *string) *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "显示 CLI 配置文件路径",
		RunE: func(cmd *cobra.Command, _ []string) error {
			path, err := configstore.DefaultPath()
			if err != nil {
				return err
			}
			return writeData(commandOutput(cmd), *outputFormat, map[string]string{"config_path": path})
		},
	}
}

func newConfigSetContextCommand() *cobra.Command {
	var serverURL string
	var token string
	var dashboardURL string
	var dashboardToken string
	command := &cobra.Command{
		Use:   "set-context NAME",
		Short: "新增或更新远端上下文",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if serverURL == "" && dashboardURL == "" {
				return fmt.Errorf("至少需要指定 --server 或 --dashboard")
			}
			store, err := configstore.LoadDefault()
			if err != nil {
				return err
			}
			store.Contexts[args[0]] = configstore.Context{
				Name:           args[0],
				ControlPlane:   serverURL,
				ControlToken:   token,
				Dashboard:      dashboardURL,
				DashboardToken: dashboardToken,
			}
			if store.CurrentContext == "" {
				store.CurrentContext = args[0]
			}
			return configstore.SaveDefault(store)
		},
	}
	command.Flags().StringVar(&serverURL, "server", "", "Control Plane 基址，例如 http://127.0.0.1:9090")
	command.Flags().StringVar(&token, "token", "", "Control Plane Bearer Token")
	command.Flags().StringVar(&dashboardURL, "dashboard", "", "Dashboard 基址，例如 http://127.0.0.1:8080")
	command.Flags().StringVar(&dashboardToken, "dashboard-token", "", "Dashboard Bearer Token")
	return command
}

func newConfigUseContextCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "use-context NAME",
		Short: "切换当前上下文",
		Args:  cobra.ExactArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			store, err := configstore.LoadDefault()
			if err != nil {
				return err
			}
			if _, ok := store.Contexts[args[0]]; !ok {
				return fmt.Errorf("context not found: %s", args[0])
			}
			store.CurrentContext = args[0]
			return configstore.SaveDefault(store)
		},
	}
}

func newConfigCurrentContextCommand(outputFormat *string) *cobra.Command {
	return &cobra.Command{
		Use:   "current-context",
		Short: "显示当前上下文",
		RunE: func(cmd *cobra.Command, _ []string) error {
			store, err := configstore.LoadDefault()
			if err != nil {
				return err
			}
			return writeData(commandOutput(cmd), *outputFormat, map[string]string{"current_context": store.CurrentContext})
		},
	}
}

func newConfigListContextsCommand(outputFormat *string) *cobra.Command {
	return &cobra.Command{
		Use:   "get-contexts",
		Short: "列出所有上下文",
		RunE: func(cmd *cobra.Command, _ []string) error {
			store, err := configstore.LoadDefault()
			if err != nil {
				return err
			}
			return writeData(commandOutput(cmd), *outputFormat, store.Contexts)
		},
	}
}
