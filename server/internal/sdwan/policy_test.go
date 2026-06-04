package sdwan

import (
	"testing"
)

func TestClassifier_DefaultPorts(t *testing.T) {
	c := NewClassifier()

	tests := []struct {
		name string
		flow FlowInfo
		want AppType
	}{
		{"HTTP", FlowInfo{DstPort: 80, Protocol: ProtoTCP}, AppHTTP},
		{"HTTPS", FlowInfo{DstPort: 443, Protocol: ProtoTCP}, AppHTTPS},
		{"SSH", FlowInfo{DstPort: 22, Protocol: ProtoTCP}, AppSSH},
		{"RDP", FlowInfo{DstPort: 3389, Protocol: ProtoTCP}, AppRDP},
		{"DNS", FlowInfo{DstPort: 53, Protocol: ProtoUDP}, AppDNS},
		{"WireGuard", FlowInfo{DstPort: 520, Protocol: ProtoUDP}, AppWireGuard},
		{"UDP fallback", FlowInfo{DstPort: 9999, Protocol: ProtoUDP}, AppQUIC},
		{"TCP unknown", FlowInfo{DstPort: 9999, Protocol: ProtoTCP}, AppUnknown},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := c.Classify(tt.flow)
			if got != tt.want {
				t.Errorf("Classify(%v) = %q, want %q", tt.flow.DstPort, got, tt.want)
			}
		})
	}

	t.Logf("Default mappings: %d", c.MappingCount())
}

func TestClassifier_CustomMapping(t *testing.T) {
	c := NewClassifier()
	c.AddPortMapping(8888, AppSSH)

	got := c.Classify(FlowInfo{DstPort: 8888, Protocol: ProtoTCP})
	if got != AppSSH {
		t.Errorf("custom mapping: got %q, want %q", got, AppSSH)
	}

	c.RemovePortMapping(8888)
	got = c.Classify(FlowInfo{DstPort: 8888, Protocol: ProtoTCP})
	if got != AppUnknown {
		t.Errorf("after remove: got %q, want %q", got, AppUnknown)
	}
}

func TestPolicyEngine_AddAndEvaluate(t *testing.T) {
	cfg := DefaultSDWANConfig()
	classifier := NewClassifier()
	engine := NewPolicyEngine(cfg, classifier)

	// Add SSH high-priority rule
	err := engine.AddRule(&PolicyRule{
		ID:           "ssh-priority",
		AppType:      AppSSH,
		Action:       ActionPrioritize,
		Priority:     PriorityCritical,
		RulePriority: 1,
		Enabled:      true,
	})
	if err != nil {
		t.Fatalf("AddRule: %v", err)
	}

	// Add bandwidth limit for HTTP
	err = engine.AddRule(&PolicyRule{
		ID:           "http-limit",
		AppType:      AppHTTP,
		Action:       ActionLimit,
		Priority:     PriorityMedium,
		BandwidthBps: 1024 * 1024, // 1 MB/s
		RulePriority: 2,
		Enabled:      true,
	})
	if err != nil {
		t.Fatalf("AddRule: %v", err)
	}

	if engine.RuleCount() != 2 {
		t.Errorf("RuleCount = %d, want 2", engine.RuleCount())
	}

	// Evaluate SSH traffic
	result := engine.Evaluate(FlowInfo{DstPort: 22, Protocol: ProtoTCP})
	if result.Priority != PriorityCritical {
		t.Errorf("SSH priority = %d, want %d", result.Priority, PriorityCritical)
	}

	// Evaluate HTTP traffic
	result = engine.Evaluate(FlowInfo{DstPort: 80, Protocol: ProtoTCP})
	if result.BandwidthLimit != 1024*1024 {
		t.Errorf("HTTP bandwidth = %d, want %d", result.BandwidthLimit, 1024*1024)
	}

	// Evaluate unmatched traffic → default priority
	result = engine.Evaluate(FlowInfo{DstPort: 9999, Protocol: ProtoTCP})
	if result.Priority != QoSPriority(cfg.DefaultPriority) {
		t.Errorf("default priority = %d, want %d", result.Priority, cfg.DefaultPriority)
	}

	t.Logf("SSH: priority=%d, HTTP: bw=%d, Default: priority=%d",
		PriorityCritical, 1024*1024, cfg.DefaultPriority)
}

func TestPolicyEngine_RulePriority(t *testing.T) {
	cfg := DefaultSDWANConfig()
	engine := NewPolicyEngine(cfg, NewClassifier())

	// Add low-priority rule first
	_ = engine.AddRule(&PolicyRule{
		ID: "low", AppType: AppHTTPS, Priority: PriorityLow, RulePriority: 10, Enabled: true,
	})
	// Add high-priority rule second (should match first due to lower RulePriority)
	_ = engine.AddRule(&PolicyRule{
		ID: "high", AppType: AppHTTPS, Priority: PriorityHigh, RulePriority: 1, Enabled: true,
	})

	result := engine.Evaluate(FlowInfo{DstPort: 443, Protocol: ProtoTCP})
	if result.Priority != PriorityHigh {
		t.Errorf("priority = %d, want %d (high-priority rule should win)", result.Priority, PriorityHigh)
	}

	// ListRules should be sorted
	rules := engine.ListRules()
	if rules[0].ID != "high" {
		t.Errorf("first rule = %q, want %q", rules[0].ID, "high")
	}
}

func TestPolicyEngine_RemoveRule(t *testing.T) {
	engine := NewPolicyEngine(DefaultSDWANConfig(), NewClassifier())
	_ = engine.AddRule(&PolicyRule{ID: "r1", AppType: AppSSH, Priority: PriorityHigh, RulePriority: 1, Enabled: true})
	_ = engine.AddRule(&PolicyRule{ID: "r2", AppType: AppHTTP, Priority: PriorityLow, RulePriority: 2, Enabled: true})

	if err := engine.RemoveRule("r1"); err != nil {
		t.Fatalf("RemoveRule: %v", err)
	}

	if engine.RuleCount() != 1 {
		t.Errorf("RuleCount = %d, want 1", engine.RuleCount())
	}

	// Remove non-existent
	if err := engine.RemoveRule("nonexistent"); err == nil {
		t.Error("expected error for non-existent rule")
	}
}

func TestPolicyEngine_UpdateRule(t *testing.T) {
	engine := NewPolicyEngine(DefaultSDWANConfig(), NewClassifier())
	_ = engine.AddRule(&PolicyRule{ID: "r1", AppType: AppSSH, Priority: PriorityHigh, RulePriority: 1, Enabled: true})

	// Update to critical
	err := engine.UpdateRule(&PolicyRule{ID: "r1", AppType: AppSSH, Priority: PriorityCritical, RulePriority: 1, Enabled: true})
	if err != nil {
		t.Fatalf("UpdateRule: %v", err)
	}

	result := engine.Evaluate(FlowInfo{DstPort: 22, Protocol: ProtoTCP})
	if result.Priority != PriorityCritical {
		t.Errorf("after update: priority = %d, want %d", result.Priority, PriorityCritical)
	}
}

func TestPolicyEngine_DisabledRule(t *testing.T) {
	engine := NewPolicyEngine(DefaultSDWANConfig(), NewClassifier())
	_ = engine.AddRule(&PolicyRule{
		ID: "disabled", AppType: AppSSH, Priority: PriorityCritical, RulePriority: 1, Enabled: false,
	})

	// Disabled rule should not match
	result := engine.Evaluate(FlowInfo{DstPort: 22, Protocol: ProtoTCP})
	if result.Priority != QoSPriority(DefaultSDWANConfig().DefaultPriority) {
		t.Errorf("disabled rule should not match, got priority %d", result.Priority)
	}
}

func TestPolicyEngine_NodeFilter(t *testing.T) {
	engine := NewPolicyEngine(DefaultSDWANConfig(), NewClassifier())
	_ = engine.AddRule(&PolicyRule{
		ID: "node-specific", AppType: AppSSH, SrcNodeID: "node-A", Priority: PriorityCritical, RulePriority: 1, Enabled: true,
	})

	// Matching node
	result := engine.Evaluate(FlowInfo{DstPort: 22, Protocol: ProtoTCP, NodeID: "node-A"})
	if result.Priority != PriorityCritical {
		t.Errorf("node-A: priority = %d, want %d", result.Priority, PriorityCritical)
	}

	// Non-matching node
	result = engine.Evaluate(FlowInfo{DstPort: 22, Protocol: ProtoTCP, NodeID: "node-B"})
	if result.Priority == PriorityCritical {
		t.Error("node-B should not match node-specific rule")
	}
}

func TestPolicyEngine_MaxRules(t *testing.T) {
	cfg := DefaultSDWANConfig()
	cfg.MaxRules = 2
	engine := NewPolicyEngine(cfg, NewClassifier())

	_ = engine.AddRule(&PolicyRule{ID: "r1", RulePriority: 1, Enabled: true})
	_ = engine.AddRule(&PolicyRule{ID: "r2", RulePriority: 2, Enabled: true})
	err := engine.AddRule(&PolicyRule{ID: "r3", RulePriority: 3, Enabled: true})
	if err == nil {
		t.Error("expected error when exceeding max rules")
	}
}

func TestPolicyEngine_RouteHint(t *testing.T) {
	engine := NewPolicyEngine(DefaultSDWANConfig(), NewClassifier())
	_ = engine.AddRule(&PolicyRule{
		ID: "ssh-direct", AppType: AppSSH, Action: ActionRoute,
		Priority: PriorityHigh, RouteHint: "udp_p2p", RulePriority: 1, Enabled: true,
	})

	result := engine.Evaluate(FlowInfo{DstPort: 22, Protocol: ProtoTCP})
	if result.RouteHint != "udp_p2p" {
		t.Errorf("RouteHint = %q, want %q", result.RouteHint, "udp_p2p")
	}
}
