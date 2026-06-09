//go:build linux

package ebpf

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cilium/ebpf"
	"github.com/cilium/ebpf/link"
	"github.com/cilium/ebpf/rlimit"
)

const (
	xdpProgramName = "xdp_nextunnel_forward"
	xdpRulesMap    = "l4_rules"
	xdpDevMap      = "tx_ports"
	xdpStatsMap    = "xdp_stats_map"
	xdpObjectName  = "xdp_forwarder_bpfel.o"
	xdpStatsKey    = uint32(0)
)

type xdpObjects struct {
	Program *ebpf.Program `ebpf:"xdp_nextunnel_forward"`
	Rules   *ebpf.Map     `ebpf:"l4_rules"`
	DevMap  *ebpf.Map     `ebpf:"tx_ports"`
	Stats   *ebpf.Map     `ebpf:"xdp_stats_map"`
}

// Loader manages eBPF program lifecycle on Linux.
type Loader struct {
	config         EBPFConfig
	mu             sync.Mutex
	mode           ForwardingMode
	xdpLink        link.Link
	xdpObjects     *xdpObjects
	ruleMap        *RuleMap
	kernelRuleKeys map[uint32]xdpL4RuleKey

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
		config:         normalizeConfig(cfg),
		mode:           ModeUserspace, // start in userspace, upgrade if eBPF loads
		kernelRuleKeys: make(map[uint32]xdpL4RuleKey),
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

	l.closeKernelLocked()
	if err := l.loadKernelLocked(); err != nil {
		if l.config.RequireKernelMode {
			return err
		}
		// 内核态能力依赖 Linux 内核、权限、网卡驱动和 BPF 对象文件；失败时必须明确降级。
		l.config.Logger.Warn("eBPF XDP unavailable, falling back to userspace forwarding",
			"interface", l.config.InterfaceName,
			"mode", l.config.XDPMode,
			"error", err)
		l.mode = ModeUserspace
		return nil
	}

	l.mode = ModeKernel
	if err := l.syncConfiguredRulesLocked(); err != nil {
		l.closeKernelLocked()
		l.mode = ModeUserspace
		if l.config.RequireKernelMode {
			return err
		}
		l.config.Logger.Warn("eBPF rule sync failed, falling back to userspace forwarding", "error", err)
		return nil
	}

	l.config.Logger.Info("eBPF XDP program attached",
		"interface", l.config.InterfaceName,
		"mode", l.config.XDPMode)
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

	l.closeKernelLocked()
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
	kernelStats := l.readKernelStats()
	return ForwardingStats{
		Mode:             l.GetMode(),
		PacketsForwarded: l.packetsForwarded.Load() + kernelStats.PacketsForwarded,
		BytesForwarded:   l.bytesForwarded.Load() + kernelStats.BytesForwarded,
		PacketsDropped:   l.packetsDropped.Load() + kernelStats.PacketsDropped,
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

// ConfigureRuleMap connects userspace RuleMap changes to the XDP BPF map.
func (l *Loader) ConfigureRuleMap(ruleMap *RuleMap) error {
	if ruleMap == nil {
		return fmt.Errorf("rule map is required")
	}

	l.mu.Lock()
	l.ruleMap = ruleMap
	l.mu.Unlock()

	ruleMap.SetKernelSyncCallbacks(l.syncKernelRulesAfterAdd, l.syncKernelRulesAfterRemove)
	return l.syncKernelRules(ruleMap.ListRules())
}

func (l *Loader) loadKernelLocked() error {
	iface, err := net.InterfaceByName(l.config.InterfaceName)
	if err != nil {
		return fmt.Errorf("lookup interface %q: %w", l.config.InterfaceName, err)
	}

	objectPath, err := l.resolveObjectPath()
	if err != nil {
		return err
	}

	if err := rlimit.RemoveMemlock(); err != nil {
		return fmt.Errorf("remove memlock limit: %w", err)
	}

	spec, err := ebpf.LoadCollectionSpec(objectPath)
	if err != nil {
		return fmt.Errorf("load eBPF object %q: %w", objectPath, err)
	}
	if rulesSpec := spec.Maps[xdpRulesMap]; rulesSpec != nil {
		rulesSpec.MaxEntries = l.config.MaxKernelRules
	}

	objects := &xdpObjects{}
	if err := spec.LoadAndAssign(objects, nil); err != nil {
		closeXDPObjects(objects)
		return fmt.Errorf("load eBPF collection: %w", err)
	}

	xdpLink, err := link.AttachXDP(link.XDPOptions{
		Program:   objects.Program,
		Interface: iface.Index,
		Flags:     xdpAttachFlags(l.config.XDPMode),
	})
	if err != nil {
		closeXDPObjects(objects)
		return fmt.Errorf("attach XDP to %q: %w", iface.Name, err)
	}

	l.xdpObjects = objects
	l.xdpLink = xdpLink
	return nil
}

func (l *Loader) closeKernelLocked() {
	if l.xdpLink != nil {
		_ = l.xdpLink.Close()
		l.xdpLink = nil
	}
	closeXDPObjects(l.xdpObjects)
	l.xdpObjects = nil
	l.kernelRuleKeys = make(map[uint32]xdpL4RuleKey)
}

func (l *Loader) resolveObjectPath() (string, error) {
	if l.config.XDPObjectPath != "" {
		cleanPath := filepath.Clean(l.config.XDPObjectPath)
		if _, err := os.Stat(cleanPath); err != nil {
			return "", fmt.Errorf("configured XDP object %q is not available: %w", cleanPath, err)
		}
		return cleanPath, nil
	}

	candidates := []string{
		xdpObjectName,
		filepath.Join("internal", "ebpf", xdpObjectName),
		filepath.Join("server", "internal", "ebpf", xdpObjectName),
		filepath.Join(filepath.Dir(os.Args[0]), xdpObjectName),
	}

	checked := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		cleanCandidate := filepath.Clean(candidate)
		checked = append(checked, cleanCandidate)
		if _, err := os.Stat(cleanCandidate); err == nil {
			return cleanCandidate, nil
		}
	}
	return "", fmt.Errorf("compiled XDP object not found; checked %s", strings.Join(checked, ", "))
}

func xdpAttachFlags(mode string) link.XDPAttachFlags {
	switch strings.ToLower(strings.TrimSpace(mode)) {
	case "", "auto":
		return 0
	case "skb", "generic":
		return link.XDPGenericMode
	case "drv", "native":
		return link.XDPDriverMode
	case "hw", "offload":
		return link.XDPOffloadMode
	default:
		return link.XDPGenericMode
	}
}

func closeXDPObjects(objects *xdpObjects) {
	if objects == nil {
		return
	}
	if objects.Program != nil {
		_ = objects.Program.Close()
	}
	if objects.Rules != nil {
		_ = objects.Rules.Close()
	}
	if objects.DevMap != nil {
		_ = objects.DevMap.Close()
	}
	if objects.Stats != nil {
		_ = objects.Stats.Close()
	}
}

func (l *Loader) syncConfiguredRulesLocked() error {
	if l.ruleMap == nil {
		return nil
	}
	return l.rebuildKernelRulesLocked(l.ruleMap.ListRules())
}

func (l *Loader) syncKernelRules(rules []*ForwardingRule) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.rebuildKernelRulesLocked(rules)
}

func (l *Loader) syncKernelRulesAfterAdd(_ *ForwardingRule) error {
	return l.syncKernelRulesFromConfiguredMap()
}

func (l *Loader) syncKernelRulesAfterRemove(_ uint32) error {
	return l.syncKernelRulesFromConfiguredMap()
}

func (l *Loader) syncKernelRulesFromConfiguredMap() error {
	l.mu.Lock()
	ruleMap := l.ruleMap
	l.mu.Unlock()
	if ruleMap == nil {
		return nil
	}
	return l.syncKernelRules(ruleMap.ListRules())
}

func (l *Loader) rebuildKernelRulesLocked(rules []*ForwardingRule) error {
	if l.mode != ModeKernel || l.xdpObjects == nil || l.xdpObjects.Rules == nil {
		return nil
	}

	plan := buildKernelRulePlan(rules)
	if len(plan) > int(l.config.MaxKernelRules) {
		return fmt.Errorf("XDP rule plan exceeds max kernel rules: %d > %d", len(plan), l.config.MaxKernelRules)
	}

	desiredRuleKeys := make(map[uint32]xdpL4RuleKey, len(plan))
	desiredKeys := make(map[xdpL4RuleKey]struct{}, len(plan))
	for _, plannedRule := range plan {
		if plannedRule.Value.Action == kernelActionRedirect {
			if err := l.xdpObjects.DevMap.Update(plannedRule.Value.IfIndex, plannedRule.Value.IfIndex, ebpf.UpdateAny); err != nil {
				return fmt.Errorf("sync XDP devmap for rule %d: %w", plannedRule.RuleID, err)
			}
		}
		if err := l.xdpObjects.Rules.Update(plannedRule.Key, plannedRule.Value, ebpf.UpdateAny); err != nil {
			return fmt.Errorf("sync XDP rule %d: %w", plannedRule.RuleID, err)
		}
		desiredRuleKeys[plannedRule.RuleID] = plannedRule.Key
		desiredKeys[plannedRule.Key] = struct{}{}
	}

	for ruleID, key := range l.kernelRuleKeys {
		if desiredKey, keepRule := desiredRuleKeys[ruleID]; keepRule && desiredKey == key {
			continue
		}
		if _, keepKey := desiredKeys[key]; keepKey {
			continue
		}
		if err := l.xdpObjects.Rules.Delete(key); err != nil && !errors.Is(err, ebpf.ErrKeyNotExist) {
			return fmt.Errorf("remove stale XDP rule %d: %w", ruleID, err)
		}
	}
	l.kernelRuleKeys = desiredRuleKeys
	return nil
}

func (l *Loader) readKernelStats() xdpKernelStats {
	l.mu.Lock()
	objects := l.xdpObjects
	l.mu.Unlock()

	if objects == nil || objects.Stats == nil {
		return xdpKernelStats{}
	}
	var stats xdpKernelStats
	if err := objects.Stats.Lookup(xdpStatsKey, &stats); err != nil {
		l.config.Logger.Debug("read XDP stats failed", "error", err)
		return xdpKernelStats{}
	}
	return stats
}

func ruleID(rule *ForwardingRule) uint32 {
	if rule == nil {
		return 0
	}
	return rule.ID
}
