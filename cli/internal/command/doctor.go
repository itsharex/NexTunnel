package command

import (
	"os"
	"os/exec"
	"runtime"

	"github.com/nextunnel/cli/internal/desktop"
	"github.com/nextunnel/cli/internal/system"
	"github.com/spf13/cobra"
)

type doctorCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail,omitempty"`
}

func newDoctorCommand(outputFormat *string) *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "检查 NexTunnel 本机环境",
		RunE: func(cmd *cobra.Command, _ []string) error {
			checks := []doctorCheck{
				checkCommand("go"),
				checkCommand("wails"),
				checkDesktopControlFile(),
				checkServerEnvFile(),
			}
			return writeData(commandOutput(cmd), *outputFormat, checks)
		},
	}
}

func checkCommand(name string) doctorCheck {
	path, err := exec.LookPath(name)
	if err != nil {
		return doctorCheck{Name: name, Status: "missing", Detail: err.Error()}
	}
	return doctorCheck{Name: name, Status: "ok", Detail: path}
}

func checkDesktopControlFile() doctorCheck {
	path, err := desktop.DefaultControlFilePath()
	if err != nil {
		return doctorCheck{Name: "desktop_control", Status: "error", Detail: err.Error()}
	}
	if _, err := os.Stat(path); err != nil {
		return doctorCheck{Name: "desktop_control", Status: "missing", Detail: path}
	}
	return doctorCheck{Name: "desktop_control", Status: "ok", Detail: path}
}

func checkServerEnvFile() doctorCheck {
	paths := system.DefaultPaths()
	if _, err := os.Stat(paths.EnvPath); err != nil {
		status := "missing"
		if runtime.GOOS != "windows" && os.Geteuid() != 0 {
			status = "missing_or_permission_denied"
		}
		return doctorCheck{Name: "server_env", Status: status, Detail: paths.EnvPath}
	}
	return doctorCheck{Name: "server_env", Status: "ok", Detail: paths.EnvPath}
}
