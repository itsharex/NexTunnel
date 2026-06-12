package virtualnet

import (
	"fmt"
	"os/exec"
	"strings"
)

// ExecRunner 使用系统命令应用网卡和路由配置。
type ExecRunner struct{}

// Run 执行指定系统命令，返回带上下文的错误信息。
func (ExecRunner) Run(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("%s %s failed: %w: %s", name, strings.Join(args, " "), err, strings.TrimSpace(string(output)))
	}
	return nil
}
