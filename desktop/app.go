package main

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"os"
	"strings"
	"sync"

	"github.com/nextunnel/desktop/internal/config"
	"github.com/nextunnel/desktop/internal/tunnel"

	"github.com/google/uuid"
)

const (
	defaultProxyType = "tcp"
	statusStopped    = "stopped"
	statusActive     = "active"
	statusRunning    = "running"
)

// App is the main Wails application struct.
type App struct {
	ctx     context.Context
	logger  *slog.Logger
	manager *tunnel.Manager
	db      *config.DB
	store   *config.Store
	runMu   sync.Mutex
	cancel  context.CancelFunc
}

// NewApp creates a new App application struct.
func NewApp() *App {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	return &App{logger: logger}
}

func (a *App) startup(ctx context.Context) {
	a.ctx = ctx

	// Open database
	db, err := config.Open("")
	if err != nil {
		a.logger.Error("failed to open config database", "error", err)
		return
	}
	a.db = db
	a.store = config.NewStore(db)

	// Initialize tunnel manager from local configuration.
	cfg := a.managerConfig()
	a.manager = tunnel.NewManager(cfg)
	a.manager.SetLogger(a.logger)
}

func (a *App) shutdown(ctx context.Context) {
	if a.manager != nil {
		a.manager.Stop()
	}
	if a.db != nil {
		a.db.Close()
	}
}

func (a *App) getOrCreateClientID() string {
	id, err := a.store.GetSetting("client_id")
	if err != nil || id == "" {
		id = uuid.New().String()
		a.store.SetSetting("client_id", id)
	}
	return id
}

// --- Wails-bound methods (callable from frontend) ---

// GetVersion returns the application version.
func (a *App) GetVersion() string {
	return "0.1.0"
}

// Greet returns a greeting for the given name.
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, NexTunnel is ready!", name)
}

// TunnelInfo is the frontend-facing tunnel info.
type TunnelInfo struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	ProxyType  string `json:"proxy_type"`
	LocalAddr  string `json:"local_addr"`
	LocalPort  int    `json:"local_port"`
	RemotePort int    `json:"remote_port"`
	Status     string `json:"status"`
}

// GetTunnels returns all tunnel configurations.
func (a *App) GetTunnels() ([]TunnelInfo, error) {
	configs, err := a.store.List()
	if err != nil {
		return nil, err
	}

	result := make([]TunnelInfo, 0, len(configs))
	for _, c := range configs {
		info := TunnelInfo{
			ID:         c.ID,
			Name:       c.Name,
			ProxyType:  c.ProxyType,
			LocalAddr:  c.LocalAddr,
			LocalPort:  c.LocalPort,
			RemotePort: c.RemotePort,
			Status:     c.Status,
		}
		// Enrich with live status from manager
		if a.manager != nil {
			for _, s := range a.manager.GetStatus() {
				if s.ProxyName == c.Name {
					info.Status = string(s.Status)
				}
			}
		}
		result = append(result, info)
	}
	return result, nil
}

// CreateTunnelInput is the input for creating a tunnel.
type CreateTunnelInput struct {
	Name       string `json:"name"`
	ProxyType  string `json:"proxy_type"`
	LocalAddr  string `json:"local_addr"`
	LocalPort  int    `json:"local_port"`
	RemotePort int    `json:"remote_port"`
}

type ServerConfigInput struct {
	ServerAddr string `json:"server_addr"`
	AuthToken  string `json:"auth_token"`
}

// CreateTunnel creates a new tunnel configuration.
func (a *App) CreateTunnel(input CreateTunnelInput) (*TunnelInfo, error) {
	if err := validateCreateTunnelInput(input); err != nil {
		return nil, err
	}
	proxyType := input.ProxyType
	if proxyType == "" {
		proxyType = defaultProxyType
	}
	tc := &config.TunnelConfig{
		ID:         uuid.New().String(),
		Name:       input.Name,
		ProxyType:  proxyType,
		LocalAddr:  input.LocalAddr,
		LocalPort:  input.LocalPort,
		RemotePort: input.RemotePort,
		Status:     statusStopped,
	}
	if err := a.store.Create(tc); err != nil {
		return nil, err
	}
	return &TunnelInfo{
		ID: tc.ID, Name: tc.Name, ProxyType: tc.ProxyType,
		LocalAddr: tc.LocalAddr, LocalPort: tc.LocalPort,
		RemotePort: tc.RemotePort, Status: tc.Status,
	}, nil
}

func validateCreateTunnelInput(input CreateTunnelInput) error {
	if strings.TrimSpace(input.Name) == "" {
		return fmt.Errorf("tunnel name is required")
	}
	if input.ProxyType != "" && input.ProxyType != "tcp" && input.ProxyType != "http" {
		return fmt.Errorf("unsupported proxy type: %s", input.ProxyType)
	}
	if net.ParseIP(input.LocalAddr) == nil && strings.TrimSpace(input.LocalAddr) == "" {
		return fmt.Errorf("local address is required")
	}
	if input.LocalPort <= 0 || input.LocalPort > 65535 {
		return fmt.Errorf("local port must be between 1 and 65535")
	}
	if input.RemotePort < 0 || input.RemotePort > 65535 {
		return fmt.Errorf("remote port must be between 0 and 65535")
	}
	return nil
}

// StartTunnel 启动一个已保存的隧道，并向已连接的 Relay 注册代理。
func (a *App) StartTunnel(id string) error {
	tc, err := a.store.Get(id)
	if err != nil {
		return err
	}
	if tc == nil {
		return fmt.Errorf("tunnel config not found: %s", id)
	}
	a.runMu.Lock()
	defer a.runMu.Unlock()
	if a.manager == nil || !a.manager.IsConnected() {
		return fmt.Errorf("server is not connected")
	}
	if !a.manager.HasTunnel(tc.Name) {
		if err := a.manager.AddTunnel(tunnelDefFromConfig(tc)); err != nil {
			return err
		}
	}
	return a.store.UpdateStatus(id, statusRunning)
}

// StopTunnel 停止一个运行中的隧道，并关闭对应的 Relay 代理。
func (a *App) StopTunnel(id string) error {
	tc, err := a.store.Get(id)
	if err != nil {
		return err
	}
	if tc == nil {
		return fmt.Errorf("tunnel config not found: %s", id)
	}
	a.runMu.Lock()
	defer a.runMu.Unlock()
	if a.manager != nil && a.manager.HasTunnel(tc.Name) {
		if err := a.manager.RemoveTunnel(tc.Name); err != nil {
			return err
		}
	}
	return a.store.UpdateStatus(id, statusStopped)
}

// DeleteTunnel removes a tunnel configuration.
func (a *App) DeleteTunnel(id string) error {
	// Also remove from manager if running
	tc, _ := a.store.Get(id)
	if tc != nil && a.manager != nil {
		a.manager.RemoveTunnel(tc.Name)
	}
	return a.store.Delete(id)
}

// ConnectServer starts the relay control connection with the current tunnel set.
func (a *App) ConnectServer(input ServerConfigInput) error {
	a.runMu.Lock()
	defer a.runMu.Unlock()
	if a.manager != nil && a.manager.IsConnected() {
		return nil
	}
	if strings.TrimSpace(input.ServerAddr) == "" {
		return fmt.Errorf("server address is required")
	}
	cfg := a.managerConfig()
	cfg.ServerAddr = input.ServerAddr
	cfg.AuthToken = input.AuthToken
	manager := tunnel.NewManager(cfg)
	manager.SetLogger(a.logger)
	parentCtx := a.ctx
	if parentCtx == nil {
		parentCtx = context.Background()
	}
	ctx, cancel := context.WithCancel(parentCtx)
	a.manager = manager
	a.cancel = cancel
	go func() {
		if err := manager.Start(ctx); err != nil && ctx.Err() == nil {
			a.logger.Error("relay manager stopped", "error", err)
		}
	}()
	return nil
}

// DisconnectServer stops the active relay control connection.
func (a *App) DisconnectServer() {
	a.runMu.Lock()
	defer a.runMu.Unlock()
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	if a.manager != nil {
		a.manager.Stop()
	}
}

func (a *App) managerConfig() tunnel.TunnelClientConfig {
	cfg := tunnel.DefaultClientConfig()
	cfg.ClientID = a.getOrCreateClientID()
	configs, err := a.store.List()
	if err != nil {
		a.logger.Error("failed to load tunnel configs", "error", err)
		return cfg
	}
	for _, c := range configs {
		if isPersistedTunnelEnabled(c.Status) {
			cfg.Tunnels = append(cfg.Tunnels, tunnelDefFromConfig(c))
		}
	}
	return cfg
}

// tunnelDefFromConfig 将持久化配置转换为运行时隧道定义。
func tunnelDefFromConfig(c *config.TunnelConfig) tunnel.TunnelDef {
	return tunnel.TunnelDef{
		Name:       c.Name,
		ProxyType:  c.ProxyType,
		LocalAddr:  fmt.Sprintf("%s:%d", c.LocalAddr, c.LocalPort),
		RemotePort: uint16(c.RemotePort),
	}
}

// isPersistedTunnelEnabled 判断哪些持久化隧道需要在重连后自动注册。
func isPersistedTunnelEnabled(status string) bool {
	return status == statusActive || status == statusRunning
}

// GetConnectionStatus returns the current connection status.
func (a *App) GetConnectionStatus() string {
	if a.manager != nil && a.manager.IsConnected() {
		return "connected"
	}
	return "disconnected"
}

// GetP2PStatus returns the current P2P engine state.
func (a *App) GetP2PStatus() string {
	if a.manager != nil {
		return a.manager.GetP2PState()
	}
	return ""
}

// GetNATType returns the detected NAT type.
func (a *App) GetNATType() string {
	if a.manager != nil {
		return a.manager.GetNATType()
	}
	return ""
}

// GetTrafficStats returns aggregate traffic statistics.
func (a *App) GetTrafficStats() map[string]int64 {
	stats := map[string]int64{"bytes_in": 0, "bytes_out": 0, "tunnels": 0}
	if a.manager != nil {
		for _, s := range a.manager.GetStatus() {
			stats["bytes_in"] += s.BytesIn
			stats["bytes_out"] += s.BytesOut
			stats["tunnels"]++
		}
	}
	return stats
}
