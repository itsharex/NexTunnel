package sdwan

import (
	"log/slog"
	"time"
)

// SDWANConfig configures the SD-WAN policy engine.
type SDWANConfig struct {
	// MaxRules is the maximum number of policy rules.
	MaxRules int

	// DefaultPriority is the default QoS priority for unmatched traffic.
	DefaultPriority int

	// PolicyEvalTimeout is the max time for evaluating a single packet.
	PolicyEvalTimeout time.Duration

	Logger *slog.Logger
}

// SDWANOption configures an SDWANConfig.
type SDWANOption func(*SDWANConfig)

// WithMaxRules sets the maximum rule count.
func WithMaxRules(n int) SDWANOption {
	return func(c *SDWANConfig) { c.MaxRules = n }
}

// WithDefaultPriority sets the default QoS priority.
func WithDefaultPriority(p int) SDWANOption {
	return func(c *SDWANConfig) { c.DefaultPriority = p }
}

// WithPolicyEvalTimeout sets the evaluation timeout.
func WithPolicyEvalTimeout(d time.Duration) SDWANOption {
	return func(c *SDWANConfig) { c.PolicyEvalTimeout = d }
}

// WithSDWANLogger sets the logger.
func WithSDWANLogger(l *slog.Logger) SDWANOption {
	return func(c *SDWANConfig) { c.Logger = l }
}

// DefaultSDWANConfig returns sensible defaults.
func DefaultSDWANConfig() SDWANConfig {
	return SDWANConfig{
		MaxRules:          1000,
		DefaultPriority:   4, // mid-priority
		PolicyEvalTimeout: 100 * time.Millisecond,
		Logger:            slog.Default(),
	}
}

// QoSPriority defines traffic priority levels (0=highest, 7=lowest).
type QoSPriority int

const (
	PriorityCritical  QoSPriority = 0
	PriorityRealtime  QoSPriority = 1
	PriorityHigh      QoSPriority = 2
	PriorityMedium    QoSPriority = 4
	PriorityLow       QoSPriority = 5
	PriorityBestEffort QoSPriority = 6
	PriorityBackground QoSPriority = 7
)

// AppType represents a classified application type.
type AppType string

const (
	AppHTTP     AppType = "http"
	AppHTTPS    AppType = "https"
	AppSSH      AppType = "ssh"
	AppRDP      AppType = "rdp"
	AppDNS      AppType = "dns"
	AppQUIC     AppType = "quic"
	AppWireGuard AppType = "wireguard"
	AppUnknown  AppType = "unknown"
)

// ProtocolType represents a network protocol.
type ProtocolType string

const (
	ProtoTCP ProtocolType = "tcp"
	ProtoUDP ProtocolType = "udp"
)

// FlowInfo describes a network flow for classification.
type FlowInfo struct {
	SrcAddr  string       `json:"src_addr"`
	DstAddr  string       `json:"dst_addr"`
	SrcPort  int          `json:"src_port"`
	DstPort  int          `json:"dst_port"`
	Protocol ProtocolType `json:"protocol"`
	NodeID   string       `json:"node_id"`
}

// ClassifyResult is the output of traffic classification.
type ClassifyResult struct {
	App      AppType     `json:"app"`
	Priority QoSPriority `json:"priority"`
	BandwidthLimit int64 `json:"bandwidth_limit"` // bytes/sec, 0 = unlimited
	RouteHint string    `json:"route_hint"`       // preferred path type
}
