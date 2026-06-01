package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"

	"github.com/nextunnel/desktop/internal/config"
	"github.com/nextunnel/desktop/internal/tunnel"
	"github.com/nextunnel/pkg/types"

	"github.com/google/uuid"
)

// App is the main Wails application struct.
type App struct {
	ctx     context.Context
	logger  *slog.Logger
	manager *tunnel.Manager
	db      *config.DB
	store   *config.Store
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

	// Load tunnel configs from database
	configs, err := a.store.List()
	if err != nil {
		a.logger.Error("failed to load tunnel configs", "error", err)
	}

	var defs []tunnel.TunnelDef
	for _, c := range configs {
		defs = append(defs, tunnel.TunnelDef{
			Name:       c.Name,
			ProxyType:  c.ProxyType,
			LocalAddr:  fmt.Sprintf("%s:%d", c.LocalAddr, c.LocalPort),
			RemotePort: uint16(c.RemotePort),
			Domain:     "", // loaded from config if needed
		})
	}

	// Initialize tunnel manager (without server connection for now)
	cfg := tunnel.DefaultClientConfig()
	cfg.ClientID = a.getOrCreateClientID()
	cfg.Tunnels = defs
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

// CreateTunnel creates a new tunnel configuration.
func (a *App) CreateTunnel(input CreateTunnelInput) (*TunnelInfo, error) {
	tc := &config.TunnelConfig{
		ID:         uuid.New().String(),
		Name:       input.Name,
		ProxyType:  input.ProxyType,
		LocalAddr:  input.LocalAddr,
		LocalPort:  input.LocalPort,
		RemotePort: input.RemotePort,
		Status:     "stopped",
	}
	if tc.ProxyType == "" {
		tc.ProxyType = "tcp"
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

// DeleteTunnel removes a tunnel configuration.
func (a *App) DeleteTunnel(id string) error {
	// Also remove from manager if running
	tc, _ := a.store.Get(id)
	if tc != nil && a.manager != nil {
		a.manager.RemoveTunnel(tc.Name)
	}
	return a.store.Delete(id)
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

// Ensure types import is used
var _ = types.ProxyTypeTCP
var _ = json.Marshal
