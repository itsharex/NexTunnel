package sdwan

import (
	"fmt"
	"sort"
	"sync"
	"time"
)

// PolicyAction defines what to do with matched traffic.
type PolicyAction string

const (
	ActionAllow   PolicyAction = "allow"
	ActionDeny    PolicyAction = "deny"
	ActionLimit   PolicyAction = "limit"    // bandwidth limit
	ActionRoute   PolicyAction = "route"    // route to specific path
	ActionPrioritize PolicyAction = "prioritize" // set QoS priority
)

// PolicyRule defines a single SD-WAN policy rule.
type PolicyRule struct {
	ID           string       `json:"id"`
	Description  string       `json:"description"`
	AppType      AppType      `json:"app_type"`
	Protocol     ProtocolType `json:"protocol"`
	SrcNodeID    string       `json:"src_node_id"` // empty = any
	DstAddr      string       `json:"dst_addr"`    // empty = any
	DstPort      int          `json:"dst_port"`    // 0 = any
	Action       PolicyAction `json:"action"`
	Priority     QoSPriority  `json:"priority"`
	BandwidthBps int64        `json:"bandwidth_bps"` // bytes/sec for ActionLimit
	RouteHint    string       `json:"route_hint"`    // path type for ActionRoute
	RulePriority int          `json:"rule_priority"` // rule evaluation order (lower = first)
	Enabled      bool         `json:"enabled"`
	CreatedAt    time.Time    `json:"created_at"`
}

// PolicyEngine evaluates traffic against rules and returns routing decisions.
type PolicyEngine struct {
	config     SDWANConfig
	classifier *Classifier
	mu         sync.RWMutex
	rules      map[string]*PolicyRule
	sorted     []*PolicyRule // sorted by RulePriority
}

// NewPolicyEngine creates a new SD-WAN policy engine.
func NewPolicyEngine(cfg SDWANConfig, classifier *Classifier) *PolicyEngine {
	return &PolicyEngine{
		config:     cfg,
		classifier: classifier,
		rules:      make(map[string]*PolicyRule),
	}
}

// AddRule adds a policy rule.
func (e *PolicyEngine) AddRule(rule *PolicyRule) error {
	if rule.ID == "" {
		return fmt.Errorf("rule ID is required")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	if len(e.rules) >= e.config.MaxRules {
		return fmt.Errorf("max rules reached (%d)", e.config.MaxRules)
	}

	rule.CreatedAt = time.Now()
	e.rules[rule.ID] = rule
	e.rebuildSorted()
	return nil
}

// RemoveRule removes a policy rule.
func (e *PolicyEngine) RemoveRule(ruleID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.rules[ruleID]; !ok {
		return fmt.Errorf("rule %q not found", ruleID)
	}
	delete(e.rules, ruleID)
	e.rebuildSorted()
	return nil
}

// UpdateRule replaces an existing rule atomically.
func (e *PolicyEngine) UpdateRule(rule *PolicyRule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, ok := e.rules[rule.ID]; !ok {
		return fmt.Errorf("rule %q not found", rule.ID)
	}
	e.rules[rule.ID] = rule
	e.rebuildSorted()
	return nil
}

// GetRule returns a rule by ID.
func (e *PolicyEngine) GetRule(ruleID string) (*PolicyRule, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	r, ok := e.rules[ruleID]
	return r, ok
}

// ListRules returns all rules sorted by priority.
func (e *PolicyEngine) ListRules() []*PolicyRule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]*PolicyRule, len(e.sorted))
	copy(result, e.sorted)
	return result
}

// RuleCount returns the number of rules.
func (e *PolicyEngine) RuleCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.rules)
}

// Evaluate classifies a flow and finds the matching policy.
func (e *PolicyEngine) Evaluate(flow FlowInfo) ClassifyResult {
	app := e.classifier.Classify(flow)

	e.mu.RLock()
	defer e.mu.RUnlock()

	// Check rules in priority order
	for _, rule := range e.sorted {
		if !rule.Enabled {
			continue
		}
		if e.matches(rule, flow, app) {
			return ClassifyResult{
				App:            app,
				Priority:       rule.Priority,
				BandwidthLimit: rule.BandwidthBps,
				RouteHint:      rule.RouteHint,
			}
		}
	}

	// Default result
	return ClassifyResult{
		App:            app,
		Priority:       QoSPriority(e.config.DefaultPriority),
		BandwidthLimit: 0,
		RouteHint:      "",
	}
}

func (e *PolicyEngine) matches(rule *PolicyRule, flow FlowInfo, app AppType) bool {
	// App type match
	if rule.AppType != "" && rule.AppType != app {
		return false
	}
	// Protocol match
	if rule.Protocol != "" && rule.Protocol != flow.Protocol {
		return false
	}
	// Source node match
	if rule.SrcNodeID != "" && rule.SrcNodeID != flow.NodeID {
		return false
	}
	// Destination port match
	if rule.DstPort != 0 && rule.DstPort != flow.DstPort {
		return false
	}
	return true
}

func (e *PolicyEngine) rebuildSorted() {
	e.sorted = make([]*PolicyRule, 0, len(e.rules))
	for _, r := range e.rules {
		e.sorted = append(e.sorted, r)
	}
	sort.Slice(e.sorted, func(i, j int) bool {
		return e.sorted[i].RulePriority < e.sorted[j].RulePriority
	})
}
