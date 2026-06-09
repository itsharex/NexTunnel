package ebpf

import (
	"log/slog"
	"time"
)

// ForwardingMode indicates how packets are forwarded.
type ForwardingMode string

const (
	ModeKernel    ForwardingMode = "kernel"    // eBPF XDP fast path
	ModeUserspace ForwardingMode = "userspace" // fallback userspace forwarding
)

// EBPFConfig configures the eBPF acceleration module.
type EBPFConfig struct {
	// Enabled indicates whether eBPF acceleration is desired.
	Enabled bool

	// RequireKernelMode makes Load return an error instead of falling back to userspace.
	RequireKernelMode bool

	// InterfaceName is the network interface to attach eBPF programs to.
	InterfaceName string

	// XDPMode is the XDP attach mode: "skb" (generic), "drv" (native), "hw" (offload).
	XDPMode string

	// XDPObjectPath is the compiled eBPF ELF object path loaded by cilium/ebpf.
	XDPObjectPath string

	// MaxKernelRules limits entries in the kernel XDP rule map.
	MaxKernelRules uint32

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

// WithRequireKernelMode controls whether Load may gracefully degrade to userspace.
func WithRequireKernelMode(required bool) EBPFOption {
	return func(c *EBPFConfig) { c.RequireKernelMode = required }
}

// WithInterface sets the network interface.
func WithInterface(iface string) EBPFOption {
	return func(c *EBPFConfig) { c.InterfaceName = iface }
}

// WithXDPMode sets the XDP attach mode.
func WithXDPMode(mode string) EBPFOption {
	return func(c *EBPFConfig) { c.XDPMode = mode }
}

// WithXDPObjectPath sets the compiled XDP eBPF ELF object path.
func WithXDPObjectPath(path string) EBPFOption {
	return func(c *EBPFConfig) { c.XDPObjectPath = path }
}

// WithMaxKernelRules sets the maximum number of L4 fast-path rules in BPF maps.
func WithMaxKernelRules(maxRules uint32) EBPFOption {
	return func(c *EBPFConfig) { c.MaxKernelRules = maxRules }
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
		Enabled:        true,
		InterfaceName:  "eth0",
		XDPMode:        "skb",
		MaxKernelRules: 4096,
		StatsInterval:  10 * time.Second,
		Logger:         slog.Default(),
	}
}

func normalizeConfig(cfg EBPFConfig) EBPFConfig {
	defaultConfig := DefaultEBPFConfig()
	if cfg.Logger == nil {
		cfg.Logger = defaultConfig.Logger
	}
	if cfg.InterfaceName == "" {
		cfg.InterfaceName = defaultConfig.InterfaceName
	}
	if cfg.XDPMode == "" {
		cfg.XDPMode = defaultConfig.XDPMode
	}
	if cfg.MaxKernelRules == 0 {
		cfg.MaxKernelRules = defaultConfig.MaxKernelRules
	}
	if cfg.StatsInterval <= 0 {
		cfg.StatsInterval = defaultConfig.StatsInterval
	}
	return cfg
}

// ForwardingStats holds forwarding performance metrics.
type ForwardingStats struct {
	Mode             ForwardingMode `json:"mode"`
	PacketsForwarded uint64         `json:"packets_forwarded"`
	BytesForwarded   uint64         `json:"bytes_forwarded"`
	PacketsDropped   uint64         `json:"packets_dropped"`
	AvgLatencyUs     float64        `json:"avg_latency_us"`
	ThroughputMbps   float64        `json:"throughput_mbps"`
}
