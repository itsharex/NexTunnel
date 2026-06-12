package virtualnet

import (
	"fmt"
	"strings"
)

type systemCommand struct {
	name string
	args []string
}

func (c systemCommand) String() string {
	if len(c.args) == 0 {
		return c.name
	}
	return c.name + " " + strings.Join(c.args, " ")
}

func buildApplyCommands(goos string, cfg Config) ([]systemCommand, error) {
	switch goos {
	case "windows":
		return buildWindowsApplyCommands(cfg), nil
	case "linux":
		return buildLinuxApplyCommands(cfg), nil
	case "darwin":
		return buildDarwinApplyCommands(cfg), nil
	default:
		return nil, fmt.Errorf("virtual network apply is unsupported on %s", goos)
	}
}

func buildResetCommands(goos string, state State) ([]systemCommand, error) {
	switch goos {
	case "windows":
		return buildWindowsResetCommands(state), nil
	case "linux":
		return buildLinuxResetCommands(state), nil
	case "darwin":
		return buildDarwinResetCommands(state), nil
	default:
		return nil, fmt.Errorf("virtual network reset is unsupported on %s", goos)
	}
}

func buildWindowsApplyCommands(cfg Config) []systemCommand {
	commands := []systemCommand{
		{
			name: "netsh",
			args: []string{"interface", "ipv4", "set", "subinterface", cfg.Interface, fmt.Sprintf("mtu=%d", cfg.MTU), "store=active"},
		},
		{
			name: "netsh",
			args: []string{"interface", "ip", "set", "address", fmt.Sprintf("name=%s", cfg.Interface), "static", cfg.VirtualIP, maskFromCIDR(cfg.Subnet)},
		},
	}
	for _, route := range cfg.Routes {
		commands = append(commands, systemCommand{
			name: "netsh",
			args: []string{"interface", "ipv4", "add", "route", fmt.Sprintf("prefix=%s", route.Destination), fmt.Sprintf("interface=%s", cfg.Interface), fmt.Sprintf("nexthop=%s", route.Gateway), fmt.Sprintf("metric=%d", route.Metric), "store=active"},
		})
	}
	return commands
}

func buildWindowsResetCommands(state State) []systemCommand {
	commands := make([]systemCommand, 0, len(state.Routes))
	for _, route := range state.Routes {
		commands = append(commands, systemCommand{
			name: "netsh",
			args: []string{"interface", "ipv4", "delete", "route", fmt.Sprintf("prefix=%s", route.Destination), fmt.Sprintf("interface=%s", state.Interface)},
		})
	}
	return commands
}

func buildLinuxApplyCommands(cfg Config) []systemCommand {
	commands := []systemCommand{
		{name: "ip", args: []string{"link", "set", "dev", cfg.Interface, "mtu", fmt.Sprintf("%d", cfg.MTU)}},
		{name: "ip", args: []string{"addr", "replace", cfg.VirtualIP + cidrSuffix(cfg.Subnet), "dev", cfg.Interface}},
		{name: "ip", args: []string{"link", "set", "dev", cfg.Interface, "up"}},
	}
	for _, route := range cfg.Routes {
		commands = append(commands, systemCommand{
			name: "ip",
			args: []string{"route", "replace", route.Destination, "via", route.Gateway, "dev", cfg.Interface, "metric", fmt.Sprintf("%d", route.Metric)},
		})
	}
	return commands
}

func buildLinuxResetCommands(state State) []systemCommand {
	commands := make([]systemCommand, 0, len(state.Routes))
	for _, route := range state.Routes {
		commands = append(commands, systemCommand{name: "ip", args: []string{"route", "del", route.Destination, "dev", state.Interface}})
	}
	return commands
}

func buildDarwinApplyCommands(cfg Config) []systemCommand {
	commands := []systemCommand{
		{name: "ifconfig", args: []string{cfg.Interface, "mtu", fmt.Sprintf("%d", cfg.MTU)}},
		{name: "ifconfig", args: []string{cfg.Interface, "inet", cfg.VirtualIP, cfg.Gateway, "netmask", maskFromCIDR(cfg.Subnet), "up"}},
	}
	for _, route := range cfg.Routes {
		commands = append(commands, systemCommand{name: "route", args: []string{"add", "-net", route.Destination, route.Gateway}})
	}
	return commands
}

func buildDarwinResetCommands(state State) []systemCommand {
	commands := make([]systemCommand, 0, len(state.Routes))
	for _, route := range state.Routes {
		commands = append(commands, systemCommand{name: "route", args: []string{"delete", "-net", route.Destination}})
	}
	return commands
}

func cidrSuffix(cidr string) string {
	parts := strings.Split(cidr, "/")
	if len(parts) != 2 {
		return ""
	}
	return "/" + parts[1]
}

func maskFromCIDR(cidr string) string {
	parts := strings.Split(cidr, "/")
	if len(parts) != 2 {
		return "255.255.255.0"
	}
	switch parts[1] {
	case "8":
		return "255.0.0.0"
	case "16":
		return "255.255.0.0"
	case "24":
		return "255.255.255.0"
	case "32":
		return "255.255.255.255"
	default:
		return "255.255.255.0"
	}
}
