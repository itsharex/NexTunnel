package ebpf

import (
	"log/slog"
	"time"
)

// ForwardingMode indicates how packets are forwarded.
type ForwardingMode string

const (
	ModeKernel   ForwardingMode = "kernel"   // eBPF XDP fast path
	ModeUserspace ForwardingMode = "userspace" // fallback userspace forwarding
)

// EBPFConfig configures the eBPF acceleration module.
type EBPFConfig struct {
	// Enabled indicates whether eBPF acceleration is desired.
	Enabled bool

	// InterfaceName is the network interface to attach eBPF programs to.
	InterfaceName string

	// XDPMode is the XDP attach mode: "skb" (generic), "drv" (native), "hw" (offload).
	XDPMode string

	// StatsInterval is the interval for collecting forwarding statistics.
	StatsInterval time.Duration

	Logger *slog.Logger
}

// EBPFOption configures an EBPFConfig.
type EBPFOption func(*EBPFConfig)

// WithEnabled sets whether eBPF is enabled.
func WithEnabled(e bool) EBPFOption {
	return func(c *EBPFConfig) { c.Enabled = e }
}

// WithInterface sets the network interface.
func WithInterface(iface string) EBPFOption {
	return func(c *EBPFConfig) { c.InterfaceName = iface }
}

// WithXDPMode sets the XDP attach mode.
func WithXDPMode(mode string) EBPFOption {
	return func(c *EBPFConfig) { c.XDPMode = mode }
}

// WithStatsInterval sets the stats collection interval.
func WithStatsInterval(d time.Duration) EBPFOption {
	return func(c *EBPFConfig) { c.StatsInterval = d }
}

// WithEBPFLogger sets the logger.
func WithEBPFLogger(l *slog.Logger) EBPFOption {
	return func(c *EBPFConfig) { c.Logger = l }
}

// DefaultEBPFConfig returns sensible defaults.
func DefaultEBPFConfig() EBPFConfig {
	return EBPFConfig{
		Enabled:       true,
		InterfaceName: "eth0",
		XDPMode:       "skb",
		StatsInterval: 10 * time.Second,
		Logger:        slog.Default(),
	}
}

// ForwardingStats holds forwarding performance metrics.
type ForwardingStats struct {
	Mode           ForwardingMode `json:"mode"`
	PacketsForwarded uint64       `json:"packets_forwarded"`
	BytesForwarded   uint64       `json:"bytes_forwarded"`
	PacketsDropped   uint64       `json:"packets_dropped"`
	AvgLatencyUs     float64      `json:"avg_latency_us"`
	ThroughputMbps   float64      `json:"throughput_mbps"`
}
