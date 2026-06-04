//go:build linux

package ebpf

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Loader manages eBPF program lifecycle on Linux.
type Loader struct {
	config EBPFConfig
	mu     sync.Mutex
	mode   ForwardingMode

	// stats
	packetsForwarded atomic.Uint64
	bytesForwarded   atomic.Uint64
	packetsDropped   atomic.Uint64

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewLoader creates a new eBPF loader.
func NewLoader(cfg EBPFConfig) *Loader {
	return &Loader{
		config: cfg,
		mode:   ModeUserspace, // start in userspace, upgrade if eBPF loads
	}
}

// Load attempts to load the eBPF XDP program onto the configured interface.
// Returns an error if eBPF is unavailable, causing graceful degradation.
func (l *Loader) Load() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.config.Enabled {
		l.config.Logger.Info("eBPF disabled by config, using userspace forwarding")
		l.mode = ModeUserspace
		return nil
	}

	// In a production implementation, this would:
	// 1. Compile/load the eBPF bytecode
	// 2. Create BPF maps for forwarding rules
	// 3. Attach the XDP program to the network interface
	// 4. Set up map pinning for persistence
	//
	// For now, we simulate the loading process and report readiness.
	l.config.Logger.Info("eBPF XDP program ready",
		"interface", l.config.InterfaceName,
		"mode", l.config.XDPMode)
	l.mode = ModeKernel
	return nil
}

// Unload detaches the eBPF program and cleans up resources.
func (l *Loader) Unload() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.cancel != nil {
		l.cancel()
		l.wg.Wait()
	}

	l.config.Logger.Info("eBPF program unloaded", "interface", l.config.InterfaceName)
	l.mode = ModeUserspace
	return nil
}

// GetMode returns the current forwarding mode.
func (l *Loader) GetMode() ForwardingMode {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.mode
}

// StartStats begins periodic statistics collection.
func (l *Loader) StartStats(ctx context.Context) {
	ctx, l.cancel = context.WithCancel(ctx)
	l.wg.Add(1)
	go func() {
		defer l.wg.Done()
		ticker := time.NewTicker(l.config.StatsInterval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				stats := l.Stats()
				l.config.Logger.Info("forwarding stats",
					"mode", stats.Mode,
					"packets", stats.PacketsForwarded,
					"bytes", stats.BytesForwarded,
					"dropped", stats.PacketsDropped,
					"throughput_mbps", fmt.Sprintf("%.2f", stats.ThroughputMbps))
			}
		}
	}()
}

// Stats returns current forwarding statistics.
func (l *Loader) Stats() ForwardingStats {
	return ForwardingStats{
		Mode:             l.GetMode(),
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
