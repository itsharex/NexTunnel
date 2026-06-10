package controlplane

import (
	"log/slog"
	"time"

	"github.com/nextunnel/pkg/tlsutil"
)

// ControlPlaneConfig configures the control plane server.
type ControlPlaneConfig struct {
	ListenAddr         string
	APIToken           string
	NodeTimeout        time.Duration
	KeyRotationPeriod  time.Duration
	ACLEvalTimeout     time.Duration
	StorePath          string // SQLite database path; empty = MemoryStore
	AuditLogPath       string // JSON Lines audit log path; empty = no audit logging
	IPAMEnabled        bool
	VirtualSubnet      string
	VirtualGateway     string
	VirtualInterface   string
	VirtualMTU         int
	VirtualRouteMetric int
	TLSEnabled         bool
	TLS                tlsutil.TLSConfig
	Logger             *slog.Logger
}

// ControlPlaneOption configures a ControlPlaneConfig.
type ControlPlaneOption func(*ControlPlaneConfig)

// WithListenAddr sets the listen address.
func WithListenAddr(addr string) ControlPlaneOption {
	return func(c *ControlPlaneConfig) { c.ListenAddr = addr }
}

// WithAPIToken 设置非健康检查 HTTP API 需要的可选 Bearer Token。
func WithAPIToken(token string) ControlPlaneOption {
	return func(c *ControlPlaneConfig) { c.APIToken = token }
}

// WithNodeTimeout sets the node heartbeat timeout.
func WithNodeTimeout(d time.Duration) ControlPlaneOption {
	return func(c *ControlPlaneConfig) { c.NodeTimeout = d }
}

// WithKeyRotation sets the key rotation period.
func WithKeyRotation(d time.Duration) ControlPlaneOption {
	return func(c *ControlPlaneConfig) { c.KeyRotationPeriod = d }
}

// WithCPLogger sets the logger.
func WithCPLogger(l *slog.Logger) ControlPlaneOption {
	return func(c *ControlPlaneConfig) { c.Logger = l }
}

// WithStorePath sets the SQLite database path for persistent storage.
// When empty, the server uses an in-memory MemoryStore (suitable for testing).
func WithStorePath(path string) ControlPlaneOption {
	return func(c *ControlPlaneConfig) { c.StorePath = path }
}

// WithTLS enables mTLS authentication on the control plane HTTP server.
func WithTLS(cfg tlsutil.TLSConfig) ControlPlaneOption {
	return func(c *ControlPlaneConfig) {
		c.TLSEnabled = true
		c.TLS = cfg
	}
}

// WithAuditLogPath sets the path for JSON Lines audit log file.
func WithAuditLogPath(path string) ControlPlaneOption {
	return func(c *ControlPlaneConfig) { c.AuditLogPath = path }
}

// WithVirtualNetwork 设置控制面下发给客户端的虚拟网络地址池与默认网关。
func WithVirtualNetwork(subnet, gateway string) ControlPlaneOption {
	return func(c *ControlPlaneConfig) {
		c.IPAMEnabled = true
		c.VirtualSubnet = subnet
		c.VirtualGateway = gateway
	}
}

// DefaultControlPlaneConfig returns sensible defaults.
func DefaultControlPlaneConfig() ControlPlaneConfig {
	return ControlPlaneConfig{
		ListenAddr:         "0.0.0.0:9090",
		NodeTimeout:        60 * time.Second,
		KeyRotationPeriod:  24 * time.Hour,
		ACLEvalTimeout:     100 * time.Millisecond,
		IPAMEnabled:        true,
		VirtualSubnet:      "10.7.0.0/24",
		VirtualGateway:     "10.7.0.1",
		VirtualInterface:   "nextunnel0",
		VirtualMTU:         1420,
		VirtualRouteMetric: 100,
		Logger:             slog.Default(),
	}
}
