package virtualnet

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// ExecRunner 使用系统命令应用网卡和路由配置。
type ExecRunner struct{}

// Run 执行指定系统命令，返回带上下文的错误信息。
func (ExecRunner) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s failed: %w: %s", name, strings.Join(args, " "), err, trimCommandOutput(output))
	}
	return nil
}

// InterfaceExists 使用只读系统命令确认 Windows 网卡别名是否已经可被 netsh 解析。
func (ExecRunner) InterfaceExists(name string) (bool, error) {
	if runtime.GOOS != "windows" {
		return true, nil
	}
	normalizedName := strings.TrimSpace(name)
	if normalizedName == "" {
		return false, nil
	}
	output, err := exec.Command("netsh", "interface", "ipv4", "show", "interfaces", fmt.Sprintf("interface=%s", normalizedName)).CombinedOutput()
	if err != nil {
		trimmedOutput := trimCommandOutput(output)
		if trimmedOutput == "" {
			return false, fmt.Errorf("check Windows interface %q failed: %w", normalizedName, err)
		}
		return false, nil
	}
	return true, nil
}

func trimCommandOutput(output []byte) string {
	return strings.TrimSpace(string(output))
}
