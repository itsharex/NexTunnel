//go:build !linux

package ebpf

import (
	"context"
	"sync"
	"sync/atomic"
)

// Loader manages eBPF program lifecycle.
// On non-Linux platforms, this is a no-op stub that always uses userspace forwarding.
type Loader struct {
	config EBPFConfig
	mu     sync.Mutex
	mode   ForwardingMode

	packetsForwarded atomic.Uint64
	bytesForwarded   atomic.Uint64
	packetsDropped   atomic.Uint64

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewLoader creates a new eBPF loader (stub on non-Linux).
func NewLoader(cfg EBPFConfig) *Loader {
	return &Loader{
		config: normalizeConfig(cfg),
		mode:   ModeUserspace,
	}
}

// Load always returns nil on non-Linux, staying in userspace mode.
func (l *Loader) Load() error {
	l.config.Logger.Info("eBPF not available on this platform, using userspace forwarding")
	l.mode = ModeUserspace
	return nil
}

// Unload is a no-op on non-Linux.
func (l *Loader) Unload() error {
	l.config.Logger.Info("eBPF stub unloaded")
	return nil
}

// GetMode always returns ModeUserspace on non-Linux.
func (l *Loader) GetMode() ForwardingMode {
	return ModeUserspace
}

// StartStats is a no-op on non-Linux.
func (l *Loader) StartStats(_ context.Context) {}

// Stats returns current forwarding statistics.
func (l *Loader) Stats() ForwardingStats {
	return ForwardingStats{
		Mode:             ModeUserspace,
		PacketsForwarded: l.packetsForwarded.Load(),
		BytesForwarded:   l.bytesForwarded.Load(),
		PacketsDropped:   l.packetsDropped.Load(),
	}
}

// RecordForward records a forwarded packet.
func (l *Loader) RecordForward(bytes int) {
	l.packetsForwarded.Add(1)
	l.bytesForwarded.Add(uint64(bytes))
}

// RecordDrop records a dropped packet.
func (l *Loader) RecordDrop() {
	l.packetsDropped.Add(1)
}

// ConfigureRuleMap keeps the API consistent with Linux; non-Linux always uses userspace rules.
func (l *Loader) ConfigureRuleMap(ruleMap *RuleMap) error {
	if ruleMap == nil {
		return nil
	}
	return nil
}
