package command

import (
	"fmt"
	"net/url"

	"github.com/nextunnel/cli/internal/api"
	"github.com/nextunnel/cli/internal/configstore"
	"github.com/spf13/cobra"
)

func newRemoteCommand(outputFormat *string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remote",
		Short: "管理远端 Dashboard / Control Plane",
	}
	cmd.AddCommand(
		newRemoteLoginCommand(),
		newRemoteHealthCommand(outputFormat),
		newRemoteNodeCommand(outputFormat),
		newRemoteACLCommand(outputFormat),
		newRemoteAlertCommand(outputFormat),
	)
	return cmd
}

func newRemoteLoginCommand() *cobra.Command {
	var dashboardURL string
	var username string
	var password string
	var contextName string
	command := &cobra.Command{
		Use:   "login",
		Short: "登录 Dashboard 并保存上下文 token",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if dashboardURL == "" || username == "" || password == "" {
				return fmt.Errorf("--dashboard、--username 和 --password 均必填")
			}
			client := api.NewClient(dashboardURL, "")
			var login struct {
				Token string `json:"token"`
			}
			if err := client.Post("/api/v1/auth/login", map[string]string{"username": username, "password": password}, &login); err != nil {
				return err
			}
			if login.Token == "" {
				return fmt.Errorf("dashboard login did not return token")
			}
			if contextName == "" {
				contextName = "default"
			}
			store, err := configstore.LoadDefault()
			if err != nil {
				return err
			}
			ctx := store.Contexts[contextName]
			ctx.Name = contextName
			ctx.Dashboard = dashboardURL
			ctx.DashboardToken = login.Token
			store.Contexts[contextName] = ctx
			store.CurrentContext = contextName
			if err := configstore.SaveDefault(store); err != nil {
				return err
			}
			_, err = fmt.Fprintf(commandOutput(cmd), "已保存上下文：%s\n", contextName)
			return err
		},
	}
	command.Flags().StringVar(&dashboardURL, "dashboard", "", "Dashboard 基址，例如 http://127.0.0.1:8080")
	command.Flags().StringVar(&username, "username", "", "Dashboard 用户名")
	command.Flags().StringVar(&password, "password", "", "Dashboard 密码")
	command.Flags().StringVar(&contextName, "context", "default", "保存的上下文名称")
	return command
}

func newRemoteHealthCommand(outputFormat *string) *cobra.Command {
	var target string
	command := &cobra.Command{
		Use:   "health",
		Short: "检查远端健康状态",
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx, err := configstore.CurrentContext()
			if err != nil {
				return err
			}
			result := map[string]string{}
			if target == "" || target == "control-plane" {
				if ctx.ControlPlane != "" {
					if err := api.NewClient(ctx.ControlPlane, ctx.ControlToken).Get("/healthz", &result); err != nil {
						return err
					}
					result["control_plane"] = "ok"
				}
			}
			if target == "" || target == "dashboard" {
				if ctx.Dashboard != "" {
					if err := api.NewClient(ctx.Dashboard, ctx.DashboardToken).Get("/api/v1/health", &result); err != nil {
						return err
					}
					result["dashboard"] = "ok"
				}
			}
			if len(result) == 0 {
				return fmt.Errorf("当前上下文未配置可检查的远端地址")
			}
			return writeData(commandOutput(cmd), *outputFormat, result)
		},
	}
	command.Flags().StringVar(&target, "target", "", "目标：control-plane 或 dashboard")
	return command
}

func newRemoteNodeCommand(outputFormat *string) *cobra.Command {
	cmd := &cobra.Command{Use: "node", Short: "管理节点"}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "列出 Control Plane 节点",
			RunE: func(cmd *cobra.Command, _ []string) error {
				ctx, err := configstore.CurrentContext()
				if err != nil {
					return err
				}
				var nodes []map[string]any
				client, err := newControlPlaneClient(ctx)
				if err != nil {
					return err
				}
				if err := client.Get("/api/v1/nodes", &nodes); err != nil {
					return err
				}
				return writeData(commandOutput(cmd), *outputFormat, nodes)
			},
		},
		&cobra.Command{
			Use:   "inspect NODE_ID",
			Short: "查看节点详情",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				ctx, err := configstore.CurrentContext()
				if err != nil {
					return err
				}
				var node map[string]any
				client, err := newControlPlaneClient(ctx)
				if err != nil {
					return err
				}
				if err := client.Get("/api/v1/nodes/"+url.PathEscape(args[0]), &node); err != nil {
					return err
				}
				return writeData(commandOutput(cmd), *outputFormat, node)
			},
		},
	)
	return cmd
}

func newRemoteACLCommand(outputFormat *string) *cobra.Command {
	cmd := &cobra.Command{Use: "acl", Short: "管理 ACL"}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "列出 ACL 规则",
			RunE: func(cmd *cobra.Command, _ []string) error {
				ctx, err := configstore.CurrentContext()
				if err != nil {
					return err
				}
				var rules []map[string]any
				client, err := newControlPlaneClient(ctx)
				if err != nil {
					return err
				}
				if err := client.Get("/api/v1/acl", &rules); err != nil {
					return err
				}
				return writeData(commandOutput(cmd), *outputFormat, rules)
			},
		},
	)
	return cmd
}

func newRemoteAlertCommand(outputFormat *string) *cobra.Command {
	cmd := &cobra.Command{Use: "alert", Short: "管理 Dashboard 告警"}
	cmd.AddCommand(
		&cobra.Command{
			Use:   "list",
			Short: "列出告警",
			RunE: func(cmd *cobra.Command, _ []string) error {
				ctx, err := configstore.CurrentContext()
				if err != nil {
					return err
				}
				var alerts []map[string]any
				client, err := newDashboardClient(ctx)
				if err != nil {
					return err
				}
				if err := client.Get("/api/v1/alerts", &alerts); err != nil {
					return err
				}
				return writeData(commandOutput(cmd), *outputFormat, alerts)
			},
		},
		&cobra.Command{
			Use:   "ack ALERT_ID",
			Short: "确认告警",
			Args:  cobra.ExactArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				ctx, err := configstore.CurrentContext()
				if err != nil {
					return err
				}
				var result map[string]any
				client, err := newDashboardClient(ctx)
				if err != nil {
					return err
				}
				if err := client.Post("/api/v1/alerts/"+url.PathEscape(args[0])+"/ack", map[string]string{}, &result); err != nil {
					return err
				}
				return writeData(commandOutput(cmd), *outputFormat, result)
			},
		},
	)
	return cmd
}

func newControlPlaneClient(ctx configstore.Context) (*api.Client, error) {
	if ctx.ControlPlane == "" {
		return nil, fmt.Errorf("当前上下文未配置 Control Plane 地址，请先执行 nextunnel config set-context <name> --server <url> --token <token>")
	}
	return api.NewClient(ctx.ControlPlane, ctx.ControlToken), nil
}

func newDashboardClient(ctx configstore.Context) (*api.Client, error) {
	if ctx.Dashboard == "" {
		return nil, fmt.Errorf("当前上下文未配置 Dashboard 地址，请先执行 nextunnel config set-context <name> --dashboard <url> --dashboard-token <token>，或使用 nextunnel remote login")
	}
	return api.NewClient(ctx.Dashboard, ctx.DashboardToken), nil
}
