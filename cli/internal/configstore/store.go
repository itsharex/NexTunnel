package configstore

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

const configFileName = "config.json"

// Context 描述一个远端控制面或 Dashboard 上下文。
type Context struct {
	Name           string `json:"name"`
	ControlPlane   string `json:"control_plane,omitempty"`
	ControlToken   string `json:"control_token,omitempty"`
	Dashboard      string `json:"dashboard,omitempty"`
	DashboardToken string `json:"dashboard_token,omitempty"`
}

// Store 是 CLI 的用户级配置文件结构。
type Store struct {
	CurrentContext string             `json:"current_context"`
	Contexts       map[string]Context `json:"contexts"`
}

func DefaultPath() (string, error) {
	base, err := defaultConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, configFileName), nil
}

func LoadDefault() (*Store, error) {
	path, err := DefaultPath()
	if err != nil {
		return nil, err
	}
	return Load(path)
}

func Load(path string) (*Store, error) {
	store := &Store{Contexts: map[string]Context{}}
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return store, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read cli config: %w", err)
	}
	if err := json.Unmarshal(data, store); err != nil {
		return nil, fmt.Errorf("decode cli config: %w", err)
	}
	if store.Contexts == nil {
		store.Contexts = map[string]Context{}
	}
	return store, nil
}

func SaveDefault(store *Store) error {
	path, err := DefaultPath()
	if err != nil {
		return err
	}
	return Save(path, store)
}

func Save(path string, store *Store) error {
	if store.Contexts == nil {
		store.Contexts = map[string]Context{}
	}
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create cli config dir: %w", err)
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return fmt.Errorf("encode cli config: %w", err)
	}
	// 配置中可能包含 token，因此使用 0600 权限避免同机用户读取。
	if err := os.WriteFile(path, data, 0600); err != nil {
		return fmt.Errorf("write cli config: %w", err)
	}
	return nil
}

func CurrentContext() (Context, error) {
	store, err := LoadDefault()
	if err != nil {
		return Context{}, err
	}
	if store.CurrentContext == "" {
		return Context{}, fmt.Errorf("current context is not configured")
	}
	ctx, ok := store.Contexts[store.CurrentContext]
	if !ok {
		return Context{}, fmt.Errorf("current context %q not found", store.CurrentContext)
	}
	return ctx, nil
}

func defaultConfigDir() (string, error) {
	switch runtime.GOOS {
	case "windows":
		if appData := os.Getenv("APPDATA"); appData != "" {
			return filepath.Join(appData, "NexTunnel", "cli"), nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("get home dir: %w", err)
	}
	return filepath.Join(home, ".nextunnel", "cli"), nil
}
