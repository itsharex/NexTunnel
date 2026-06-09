package ebpf

import (
	"fmt"
	"net"
	"sync"
	"sync/atomic"
)

// ForwardingRule defines a single packet forwarding rule.
// In kernel mode, these rules are synced to BPF maps.
// In userspace mode, they are used directly for packet processing.
type ForwardingRule struct {
	ID       uint32     `json:"id"`
	SrcAddr  string     `json:"src_addr"` // source IP or CIDR
	DstAddr  string     `json:"dst_addr"` // destination IP or CIDR
	SrcPort  uint16     `json:"src_port"` // 0 = any
	DstPort  uint16     `json:"dst_port"` // 0 = any
	Protocol uint8      `json:"protocol"` // 6=TCP, 17=UDP, 0=any
	Action   RuleAction `json:"action"`
	Target   string     `json:"target"`   // forward target address (ip:port)
	Priority int        `json:"priority"` // lower = higher priority
}

// RuleAction defines what to do with a matched packet.
type RuleAction string

const (
	ActionForward RuleAction = "forward"
	ActionDrop    RuleAction = "drop"
	ActionPass    RuleAction = "pass" // pass to kernel stack
)

// RuleMap manages forwarding rules. In kernel mode, rules are synced to BPF maps.
// In userspace mode, rules are kept in memory for software-based forwarding.
type RuleMap struct {
	mu     sync.RWMutex
	rules  map[uint32]*ForwardingRule
	sorted []*ForwardingRule // sorted by priority
	nextID atomic.Uint32

	// Callback for syncing rules to BPF maps (kernel mode)
	onRuleAdd    func(*ForwardingRule) error
	onRuleRemove func(uint32) error
}

// NewRuleMap creates a new forwarding rule map.
func NewRuleMap() *RuleMap {
	return &RuleMap{
		rules: make(map[uint32]*ForwardingRule),
	}
}

// SetKernelSyncCallbacks registers callbacks for syncing rules to BPF maps.
func (rm *RuleMap) SetKernelSyncCallbacks(onAdd func(*ForwardingRule) error, onRemove func(uint32) error) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	rm.onRuleAdd = onAdd
	rm.onRuleRemove = onRemove
}

// AddRule adds a forwarding rule. Returns the assigned rule ID.
func (rm *RuleMap) AddRule(rule *ForwardingRule) (uint32, error) {
	rm.mu.Lock()
	if rule.Action == "" {
		rm.mu.Unlock()
		return 0, fmt.Errorf("rule action is required")
	}

	id := rm.nextID.Add(1)
	rule.ID = id
	rm.rules[id] = rule
	rm.rebuildSorted()
	onRuleAdd := rm.onRuleAdd
	onRuleRemove := rm.onRuleRemove
	rm.mu.Unlock()

	if onRuleAdd != nil {
		if err := onRuleAdd(rule); err != nil {
			// 内核同步失败时回滚内存规则，避免用户态和内核态规则集长期分裂。
			rm.mu.Lock()
			delete(rm.rules, id)
			rm.rebuildSorted()
			rm.mu.Unlock()
			if onRuleRemove != nil {
				_ = onRuleRemove(id)
			}
			return 0, fmt.Errorf("sync to kernel: %w", err)
		}
	}

	return id, nil
}

// RemoveRule removes a rule by ID.
func (rm *RuleMap) RemoveRule(id uint32) error {
	rm.mu.Lock()
	if _, ok := rm.rules[id]; !ok {
		rm.mu.Unlock()
		return fmt.Errorf("rule %d not found", id)
	}
	delete(rm.rules, id)
	rm.rebuildSorted()
	onRuleRemove := rm.onRuleRemove
	rm.mu.Unlock()

	if onRuleRemove != nil {
		_ = onRuleRemove(id) // best effort
	}
	return nil
}

// GetRule returns a rule by ID.
func (rm *RuleMap) GetRule(id uint32) (*ForwardingRule, bool) {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	r, ok := rm.rules[id]
	return r, ok
}

// ListRules returns all rules sorted by priority.
func (rm *RuleMap) ListRules() []*ForwardingRule {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	result := make([]*ForwardingRule, len(rm.sorted))
	copy(result, rm.sorted)
	return result
}

// RuleCount returns the number of rules.
func (rm *RuleMap) RuleCount() int {
	rm.mu.RLock()
	defer rm.mu.RUnlock()
	return len(rm.rules)
}

// Match finds the first matching rule for a packet.
// Returns nil if no rule matches.
func (rm *RuleMap) Match(srcIP, dstIP string, srcPort, dstPort uint16, protocol uint8) *ForwardingRule {
	rm.mu.RLock()
	defer rm.mu.RUnlock()

	for _, rule := range rm.sorted {
		if rm.matchesRule(rule, srcIP, dstIP, srcPort, dstPort, protocol) {
			return rule
		}
	}
	return nil
}

func (rm *RuleMap) matchesRule(rule *ForwardingRule, srcIP, dstIP string, srcPort, dstPort uint16, protocol uint8) bool {
	// Protocol match
	if rule.Protocol != 0 && rule.Protocol != protocol {
		return false
	}
	// Source port match
	if rule.SrcPort != 0 && rule.SrcPort != srcPort {
		return false
	}
	// Destination port match
	if rule.DstPort != 0 && rule.DstPort != dstPort {
		return false
	}
	// Source IP match
	if rule.SrcAddr != "" && !ipMatches(rule.SrcAddr, srcIP) {
		return false
	}
	// Destination IP match
	if rule.DstAddr != "" && !ipMatches(rule.DstAddr, dstIP) {
		return false
	}
	return true
}

// ipMatches checks if an IP address matches a rule address (exact or CIDR).
func ipMatches(ruleAddr, packetIP string) bool {
	if ruleAddr == packetIP {
		return true
	}
	// Try CIDR match
	_, cidr, err := net.ParseCIDR(ruleAddr)
	if err != nil {
		return false
	}
	ip := net.ParseIP(packetIP)
	if ip == nil {
		return false
	}
	return cidr.Contains(ip)
}

func (rm *RuleMap) rebuildSorted() {
	rm.sorted = make([]*ForwardingRule, 0, len(rm.rules))
	for _, r := range rm.rules {
		rm.sorted = append(rm.sorted, r)
	}
	// Sort by priority (lower = higher priority)
	for i := 0; i < len(rm.sorted); i++ {
		for j := i + 1; j < len(rm.sorted); j++ {
			if rm.sorted[j].Priority < rm.sorted[i].Priority {
				rm.sorted[i], rm.sorted[j] = rm.sorted[j], rm.sorted[i]
			}
		}
	}
}

// UserspaceForwarder provides software-based packet forwarding
// when eBPF/XDP is not available.
type UserspaceForwarder struct {
	ruleMap *RuleMap
	loader  *Loader
}

// NewUserspaceForwarder creates a userspace forwarding engine.
func NewUserspaceForwarder(loader *Loader) *UserspaceForwarder {
	return &UserspaceForwarder{
		ruleMap: NewRuleMap(),
		loader:  loader,
	}
}

// RuleMap returns the underlying rule map for configuration.
func (f *UserspaceForwarder) RuleMap() *RuleMap {
	return f.ruleMap
}

// ProcessPacket evaluates a packet against rules and performs the action.
// Returns the target address for forwarding, or empty string for drop/pass.
func (f *UserspaceForwarder) ProcessPacket(srcIP, dstIP string, srcPort, dstPort uint16, protocol uint8, packetSize int) (string, error) {
	rule := f.ruleMap.Match(srcIP, dstIP, srcPort, dstPort, protocol)
	if rule == nil {
		// No matching rule: default forward with userspace stats
		f.loader.RecordForward(packetSize)
		return "", nil
	}

	switch rule.Action {
	case ActionDrop:
		f.loader.RecordDrop()
		return "", nil
	case ActionPass:
		// Pass to kernel stack (no userspace action)
		return "", nil
	case ActionForward:
		f.loader.RecordForward(packetSize)
		return rule.Target, nil
	default:
		return "", fmt.Errorf("unknown action: %s", rule.Action)
	}
}
