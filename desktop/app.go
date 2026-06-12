package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/nextunnel/desktop/internal/config"
	"github.com/nextunnel/desktop/internal/nat"
	"github.com/nextunnel/desktop/internal/p2p"
	"github.com/nextunnel/desktop/internal/tunnel"
	"github.com/nextunnel/desktop/internal/virtualnet"

	"github.com/google/uuid"
)

const (
	defaultProxyType = "tcp"
	statusStopped    = "stopped"
	statusActive     = "active"
	statusRunning    = "running"
)

// AppVersion 通过发布脚本的 -ldflags 注入；默认值用于本地开发和测试。
var AppVersion = "0.2.1-alpha"

// App is the main Wails application struct.
type App struct {
	ctx             context.Context
	logger          *slog.Logger
	manager         *tunnel.Manager
	db              *config.DB
	store           *config.Store
	vnet            *virtualnet.Manager
	controlServer   *http.Server
	controlFilePath string
	runMu           sync.Mutex
	cancel          context.CancelFunc
	lastErr         string
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
	a.vnet = virtualnet.NewManager(nil, a.logger)

	// Initialize tunnel manager from local configuration.
	cfg := a.managerConfig()
	a.manager = tunnel.NewManager(cfg)
	a.manager.SetLogger(a.logger)
	if err := a.startControlServer(); err != nil {
		a.logger.Error("failed to start desktop control api", "error", err)
		a.recordError(err)
	}
}

func (a *App) shutdown(ctx context.Context) {
	a.stopControlServer(ctx)
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
	return AppVersion
}

// Greet returns a greeting for the given name.
func (a *App) Greet(name string) string {
	return fmt.Sprintf("Hello %s, NexTunnel is ready!", name)
}

// ServerSettings 保存桌面端连接 Relay、Control Plane 和 STUN 的生产配置。
type ServerSettings struct {
	RelayAddr         string `json:"relay_addr"`
	RelayToken        string `json:"relay_token"`
	ControlPlaneURL   string `json:"control_plane_url"`
	ControlPlaneToken string `json:"control_plane_token"`
	STUNServer        string `json:"stun_server"`
	STUNAltServer     string `json:"stun_alt_server"`
}

// RuntimeStatus 汇总桌面端运行态，供总览和网络页展示。
type RuntimeStatus struct {
	ConnectionStatus string                   `json:"connection_status"`
	P2PStatus        string                   `json:"p2p_status"`
	NATType          string                   `json:"nat_type"`
	TUN              p2p.PlatformCapabilities `json:"tun"`
	VirtualNetwork   virtualnet.State         `json:"virtual_network"`
	TrafficStats     map[string]int64         `json:"traffic_stats"`
	LastError        string                   `json:"last_error"`
}

// NATDetectionInfo 是手动 NAT 探测返回给前端的诊断结果。
type NATDetectionInfo struct {
	Type       string `json:"type"`
	PublicAddr string `json:"public_addr"`
	MappedPort uint16 `json:"mapped_port"`
	LocalAddr  string `json:"local_addr"`
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

const (
	settingRelayAddr         = "relay_addr"
	settingRelayToken        = "relay_token"
	settingControlPlaneURL   = "control_plane_url"
	settingControlPlaneToken = "control_plane_token"
	settingSTUNServer        = "stun_server"
	settingSTUNAltServer     = "stun_alt_server"
	defaultRelayAddr         = "127.0.0.1:7000"
	defaultSTUNServer        = "stun.l.google.com:19302"
)

// CreateTunnel creates a new tunnel configuration.
func (a *App) CreateTunnel(input CreateTunnelInput) (*TunnelInfo, error) {
	if err := validateCreateTunnelInput(input); err != nil {
		a.recordError(err)
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
		a.recordError(err)
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
		a.recordError(err)
		return err
	}
	if tc == nil {
		err := fmt.Errorf("tunnel config not found: %s", id)
		a.recordError(err)
		return err
	}
	a.runMu.Lock()
	defer a.runMu.Unlock()
	if a.manager == nil || !a.manager.IsConnected() {
		err := fmt.Errorf("server is not connected")
		a.recordError(err)
		return err
	}
	if !a.manager.HasTunnel(tc.Name) {
		if err := a.manager.AddTunnel(tunnelDefFromConfig(tc)); err != nil {
			a.recordError(err)
			return err
		}
	}
	if err := a.store.UpdateStatus(id, statusRunning); err != nil {
		a.recordError(err)
		return err
	}
	a.clearError()
	return nil
}

// StopTunnel 停止一个运行中的隧道，并关闭对应的 Relay 代理。
func (a *App) StopTunnel(id string) error {
	tc, err := a.store.Get(id)
	if err != nil {
		a.recordError(err)
		return err
	}
	if tc == nil {
		err := fmt.Errorf("tunnel config not found: %s", id)
		a.recordError(err)
		return err
	}
	a.runMu.Lock()
	defer a.runMu.Unlock()
	if a.manager != nil && a.manager.HasTunnel(tc.Name) {
		if err := a.manager.RemoveTunnel(tc.Name); err != nil {
			a.recordError(err)
			return err
		}
	}
	if err := a.store.UpdateStatus(id, statusStopped); err != nil {
		a.recordError(err)
		return err
	}
	a.clearError()
	return nil
}

// DeleteTunnel removes a tunnel configuration.
func (a *App) DeleteTunnel(id string) error {
	// Also remove from manager if running
	tc, _ := a.store.Get(id)
	if tc != nil && a.manager != nil {
		_ = a.manager.RemoveTunnel(tc.Name)
	}
	if err := a.store.Delete(id); err != nil {
		a.recordError(err)
		return err
	}
	a.clearError()
	return nil
}

// ConnectServer starts the relay control connection with the current tunnel set.
func (a *App) ConnectServer(input ServerConfigInput) error {
	a.runMu.Lock()
	defer a.runMu.Unlock()
	if a.manager != nil && a.manager.IsConnected() {
		return nil
	}
	if strings.TrimSpace(input.ServerAddr) == "" {
		err := fmt.Errorf("server address is required")
		a.recordError(err)
		return err
	}
	settings := a.GetServerSettings()
	settings.RelayAddr = input.ServerAddr
	settings.RelayToken = input.AuthToken
	if err := a.SaveServerSettings(settings); err != nil {
		a.recordError(err)
		return err
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
			a.recordError(err)
		}
	}()
	a.clearError()
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

// GetServerSettings 返回持久化连接设置，缺省值用于首次启动。
func (a *App) GetServerSettings() ServerSettings {
	return ServerSettings{
		RelayAddr:         a.settingOrDefault(settingRelayAddr, defaultRelayAddr),
		RelayToken:        a.settingOrDefault(settingRelayToken, ""),
		ControlPlaneURL:   a.settingOrDefault(settingControlPlaneURL, ""),
		ControlPlaneToken: a.settingOrDefault(settingControlPlaneToken, ""),
		STUNServer:        a.settingOrDefault(settingSTUNServer, defaultSTUNServer),
		STUNAltServer:     a.settingOrDefault(settingSTUNAltServer, defaultSTUNServer),
	}
}

// SaveServerSettings 持久化桌面端连接和网络探测配置。
func (a *App) SaveServerSettings(settings ServerSettings) error {
	if a.store == nil {
		err := fmt.Errorf("config store is not ready")
		a.recordError(err)
		return err
	}
	if strings.TrimSpace(settings.RelayAddr) == "" {
		settings.RelayAddr = defaultRelayAddr
	}
	if strings.TrimSpace(settings.STUNServer) == "" {
		settings.STUNServer = defaultSTUNServer
	}
	if strings.TrimSpace(settings.STUNAltServer) == "" {
		settings.STUNAltServer = settings.STUNServer
	}
	values := map[string]string{
		settingRelayAddr:         settings.RelayAddr,
		settingRelayToken:        settings.RelayToken,
		settingControlPlaneURL:   strings.TrimRight(settings.ControlPlaneURL, "/"),
		settingControlPlaneToken: settings.ControlPlaneToken,
		settingSTUNServer:        settings.STUNServer,
		settingSTUNAltServer:     settings.STUNAltServer,
	}
	for key, value := range values {
		if err := a.store.SetSetting(key, value); err != nil {
			a.recordError(err)
			return fmt.Errorf("save setting %s: %w", key, err)
		}
	}
	a.clearError()
	return nil
}

// GetRuntimeStatus 聚合连接、P2P、NAT、TUN 和虚拟网络运行状态。
func (a *App) GetRuntimeStatus() RuntimeStatus {
	return RuntimeStatus{
		ConnectionStatus: a.GetConnectionStatus(),
		P2PStatus:        a.GetP2PStatus(),
		NATType:          a.GetNATType(),
		TUN:              p2p.CurrentPlatform(),
		VirtualNetwork:   a.virtualNetworkState(),
		TrafficStats:     a.GetTrafficStats(),
		LastError:        a.lastErr,
	}
}

// ApplyVirtualNetwork 从 Control Plane 拉取节点路由配置并应用到本机。
func (a *App) ApplyVirtualNetwork() (virtualnet.State, error) {
	settings := a.GetServerSettings()
	if strings.TrimSpace(settings.ControlPlaneURL) == "" {
		err := fmt.Errorf("control plane url is required")
		a.recordError(err)
		return a.virtualNetworkState(), err
	}
	clientID := a.getOrCreateClientID()
	cfg, err := fetchVirtualNetworkConfig(settings.ControlPlaneURL, settings.ControlPlaneToken, clientID)
	if err != nil {
		a.recordError(err)
		return a.virtualNetworkState(), err
	}
	state, err := a.vnet.Apply(*cfg)
	if err != nil {
		a.recordError(err)
		return state, err
	}
	a.clearError()
	return state, nil
}

// ResetVirtualNetwork 回滚已应用的虚拟网络路由。
func (a *App) ResetVirtualNetwork() (virtualnet.State, error) {
	if a.vnet == nil {
		return virtualnet.State{}, nil
	}
	state, err := a.vnet.Reset()
	if err != nil {
		a.recordError(err)
		return state, err
	}
	a.clearError()
	return state, nil
}

// DetectNAT 手动执行 STUN 探测，用于网络页诊断。
func (a *App) DetectNAT() (*NATDetectionInfo, error) {
	settings := a.GetServerSettings()
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()
	client := nat.NewClient(nat.WithLogger(a.logger))
	detector := nat.NewDetector(settings.STUNServer, settings.STUNAltServer, client, a.logger)
	result, err := detector.Detect(ctx)
	if err != nil {
		a.recordError(err)
		return nil, err
	}
	info := &NATDetectionInfo{
		Type:       string(result.Type),
		PublicAddr: result.PublicAddr,
		MappedPort: result.MappedPort,
		LocalAddr:  result.LocalAddr,
	}
	a.clearError()
	return info, nil
}

func (a *App) managerConfig() tunnel.TunnelClientConfig {
	cfg := tunnel.DefaultClientConfig()
	cfg.ClientID = a.getOrCreateClientID()
	settings := a.GetServerSettings()
	cfg.ServerAddr = settings.RelayAddr
	cfg.AuthToken = settings.RelayToken
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

func (a *App) settingOrDefault(key, fallback string) string {
	if a.store == nil {
		return fallback
	}
	value, err := a.store.GetSetting(key)
	if err != nil || strings.TrimSpace(value) == "" {
		return fallback
	}
	return value
}

func (a *App) virtualNetworkState() virtualnet.State {
	if a.vnet == nil {
		return virtualnet.State{}
	}
	return a.vnet.State()
}

func (a *App) recordError(err error) {
	if err == nil {
		return
	}
	a.lastErr = err.Error()
	if a.logger != nil {
		a.logger.Error("desktop runtime error", "error", err)
	}
}

func (a *App) clearError() {
	a.lastErr = ""
}

func fetchVirtualNetworkConfig(baseURL, token, nodeID string) (*virtualnet.Config, error) {
	requestURL := fmt.Sprintf("%s/api/v1/nodes/%s/routes", strings.TrimRight(baseURL, "/"), nodeID)
	req, err := http.NewRequest(http.MethodGet, requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("create route config request: %w", err)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch route config: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("read route config response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("fetch route config failed: HTTP %d %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var cfg virtualnet.Config
	if err := json.Unmarshal(body, &cfg); err != nil {
		return nil, fmt.Errorf("decode route config: %w", err)
	}
	return &cfg, nil
}
