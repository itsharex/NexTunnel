package system

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/nextunnel/cli/internal/envfile"
)

const (
	defaultLinuxInstallDir = "/opt/nextunnel"
	defaultLinuxConfigDir  = "/etc/nextunnel"
	defaultLinuxDataDir    = "/var/lib/nextunnel"
)

type Paths struct {
	InstallDir string `json:"install_dir"`
	ConfigDir  string `json:"config_dir"`
	DataDir    string `json:"data_dir"`
	BinDir     string `json:"bin_dir"`
	LogDir     string `json:"log_dir"`
	RunDir     string `json:"run_dir"`
	EnvPath    string `json:"env_path"`
}

func DefaultPaths() Paths {
	if runtime.GOOS == "windows" {
		return defaultWindowsPaths()
	}
	return defaultLinuxPaths(os.Getenv, executableInstallRoots)
}

func defaultWindowsPaths() Paths {
	root := os.Getenv("ProgramData")
	if root == "" {
		root = filepath.Join(os.TempDir(), "NexTunnel")
	} else {
		root = filepath.Join(root, "NexTunnel")
	}
	installDir := filepath.Join(root, "server")
	configDir := filepath.Join(root, "config")
	dataDir := filepath.Join(root, "data")
	return Paths{
		InstallDir: installDir,
		ConfigDir:  configDir,
		DataDir:    dataDir,
		BinDir:     filepath.Join(installDir, "bin"),
		LogDir:     filepath.Join(installDir, "logs"),
		RunDir:     filepath.Join(installDir, "run"),
		EnvPath:    filepath.Join(configDir, "server.env"),
	}
}

func defaultLinuxPaths(getenv func(string) string, installRoots func() []string) Paths {
	localEnv := readFirstLocalInstallEnv(installRoots())
	installDir := envOrLocalOrDefault(getenv, localEnv, "NEXTUNNEL_INSTALL_DIR", defaultLinuxInstallDir)
	configDir := envOrLocalOrDefault(getenv, localEnv, "NEXTUNNEL_CONFIG_DIR", defaultLinuxConfigDir)
	dataDir := envOrLocalOrDefault(getenv, localEnv, "NEXTUNNEL_DATA_DIR", defaultLinuxDataDir)
	return Paths{
		InstallDir: installDir,
		ConfigDir:  configDir,
		DataDir:    dataDir,
		BinDir:     filepath.Join(installDir, "bin"),
		LogDir:     "/var/log/nextunnel",
		RunDir:     filepath.Join(installDir, "run"),
		EnvPath:    filepath.Join(configDir, "server.env"),
	}
}

func envOrDefault(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envOrLocalOrDefault(getenv func(string) string, localEnv map[string]string, key, fallback string) string {
	if value := getenv(key); value != "" {
		return value
	}
	if value := localEnv[key]; value != "" {
		return value
	}
	return fallback
}

func readFirstLocalInstallEnv(roots []string) map[string]string {
	for _, root := range roots {
		values, err := envfile.Read(filepath.Join(root, "deploy", "server", ".env"))
		if err == nil {
			return values
		}
	}
	return map[string]string{}
}
