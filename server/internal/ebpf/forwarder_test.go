package ebpf

import (
	"testing"
)

func TestRuleMap_AddRemove(t *testing.T) {
	rm := NewRuleMap()

	id, err := rm.AddRule(&ForwardingRule{
		DstPort: 80,
		Action:  ActionForward,
		Target:  "10.0.0.1:8080",
	})
	if err != nil {
		t.Fatal(err)
	}
	if rm.RuleCount() != 1 {
		t.Fatalf("expected 1 rule, got %d", rm.RuleCount())
	}

	rule, ok := rm.GetRule(id)
	if !ok {
		t.Fatal("expected rule to exist")
	}
	if rule.Target != "10.0.0.1:8080" {
		t.Fatalf("expected target 10.0.0.1:8080, got %s", rule.Target)
	}

	if err := rm.RemoveRule(id); err != nil {
		t.Fatal(err)
	}
	if rm.RuleCount() != 0 {
		t.Fatal("expected 0 rules after removal")
	}
}

func TestRuleMap_AddRuleRequiresAction(t *testing.T) {
	rm := NewRuleMap()
	_, err := rm.AddRule(&ForwardingRule{DstPort: 80})
	if err == nil {
		t.Fatal("expected error for missing action")
	}
}

func TestRuleMap_MatchByPort(t *testing.T) {
	rm := NewRuleMap()
	rm.AddRule(&ForwardingRule{DstPort: 80, Action: ActionForward, Target: "10.0.0.1:8080", Priority: 10})
	rm.AddRule(&ForwardingRule{DstPort: 443, Action: ActionForward, Target: "10.0.0.1:8443", Priority: 10})

	rule := rm.Match("1.2.3.4", "5.6.7.8", 12345, 80, 6)
	if rule == nil {
		t.Fatal("expected match for port 80")
	}
	if rule.Target != "10.0.0.1:8080" {
		t.Fatalf("expected target 10.0.0.1:8080, got %s", rule.Target)
	}

	rule = rm.Match("1.2.3.4", "5.6.7.8", 12345, 443, 6)
	if rule == nil {
		t.Fatal("expected match for port 443")
	}
}

func TestRuleMap_MatchByCIDR(t *testing.T) {
	rm := NewRuleMap()
	rm.AddRule(&ForwardingRule{
		SrcAddr: "192.168.1.0/24",
		DstPort: 22,
		Action:  ActionDrop,
	})

	// Should match
	rule := rm.Match("192.168.1.100", "10.0.0.1", 54321, 22, 6)
	if rule == nil {
		t.Fatal("expected match for CIDR")
	}
	if rule.Action != ActionDrop {
		t.Fatalf("expected drop, got %s", rule.Action)
	}

	// Should not match
	rule = rm.Match("10.0.0.5", "10.0.0.1", 54321, 22, 6)
	if rule != nil {
		t.Fatal("expected no match for different source")
	}
}

func TestRuleMap_MatchPriority(t *testing.T) {
	rm := NewRuleMap()

	// Low priority catch-all
	rm.AddRule(&ForwardingRule{
		DstPort:  80,
		Action:   ActionForward,
		Target:   "default:80",
		Priority: 100,
	})

	// High priority specific rule
	rm.AddRule(&ForwardingRule{
		SrcAddr:  "10.0.0.0/8",
		DstPort:  80,
		Action:   ActionForward,
		Target:   "internal:80",
		Priority: 1,
	})

	// Internal traffic should hit high-priority rule
	rule := rm.Match("10.0.0.5", "5.6.7.8", 12345, 80, 6)
	if rule == nil {
		t.Fatal("expected match")
	}
	if rule.Target != "internal:80" {
		t.Fatalf("expected internal:80, got %s", rule.Target)
	}

	// External traffic should hit default
	rule = rm.Match("203.0.113.1", "5.6.7.8", 12345, 80, 6)
	if rule == nil {
		t.Fatal("expected match")
	}
	if rule.Target != "default:80" {
		t.Fatalf("expected default:80, got %s", rule.Target)
	}
}

func TestRuleMap_MatchProtocol(t *testing.T) {
	rm := NewRuleMap()
	rm.AddRule(&ForwardingRule{DstPort: 53, Protocol: 17, Action: ActionForward, Target: "dns:53"})

	// UDP DNS should match
	rule := rm.Match("1.1.1.1", "8.8.8.8", 12345, 53, 17)
	if rule == nil {
		t.Fatal("expected match for UDP DNS")
	}

	// TCP DNS should not match
	rule = rm.Match("1.1.1.1", "8.8.8.8", 12345, 53, 6)
	if rule != nil {
		t.Fatal("expected no match for TCP DNS")
	}
}

func TestRuleMap_NoMatch(t *testing.T) {
	rm := NewRuleMap()
	rm.AddRule(&ForwardingRule{DstPort: 80, Action: ActionForward, Target: "web:80"})

	rule := rm.Match("1.1.1.1", "2.2.2.2", 12345, 443, 6)
	if rule != nil {
		t.Fatal("expected no match for port 443")
	}
}

func TestRuleMap_KernelSyncCallback(t *testing.T) {
	rm := NewRuleMap()

	var syncedAdds int
	var syncedRemoves int

	rm.SetKernelSyncCallbacks(
		func(r *ForwardingRule) error { syncedAdds++; return nil },
		func(id uint32) error { syncedRemoves++; return nil },
	)

	id, _ := rm.AddRule(&ForwardingRule{DstPort: 80, Action: ActionForward, Target: "t:80"})
	rm.RemoveRule(id)

	if syncedAdds != 1 {
		t.Fatalf("expected 1 synced add, got %d", syncedAdds)
	}
	if syncedRemoves != 1 {
		t.Fatalf("expected 1 synced remove, got %d", syncedRemoves)
	}
}

func TestRuleMap_ListRules(t *testing.T) {
	rm := NewRuleMap()
	rm.AddRule(&ForwardingRule{DstPort: 80, Action: ActionForward, Target: "a", Priority: 2})
	rm.AddRule(&ForwardingRule{DstPort: 443, Action: ActionForward, Target: "b", Priority: 1})

	rules := rm.ListRules()
	if len(rules) != 2 {
		t.Fatalf("expected 2 rules, got %d", len(rules))
	}
	// Should be sorted by priority
	if rules[0].Target != "b" {
		t.Fatalf("expected b first (priority 1), got %s", rules[0].Target)
	}
}

func TestUserspaceForwarder_ProcessForward(t *testing.T) {
	loader := NewLoader(DefaultEBPFConfig())
	fwd := NewUserspaceForwarder(loader)

	fwd.RuleMap().AddRule(&ForwardingRule{
		DstPort: 80,
		Action:  ActionForward,
		Target:  "10.0.0.1:8080",
	})

	target, err := fwd.ProcessPacket("1.1.1.1", "2.2.2.2", 12345, 80, 6, 1500)
	if err != nil {
		t.Fatal(err)
	}
	if target != "10.0.0.1:8080" {
		t.Fatalf("expected target 10.0.0.1:8080, got %q", target)
	}

	stats := loader.Stats()
	if stats.PacketsForwarded != 1 {
		t.Fatalf("expected 1 forwarded, got %d", stats.PacketsForwarded)
	}
}

func TestUserspaceForwarder_ProcessDrop(t *testing.T) {
	loader := NewLoader(DefaultEBPFConfig())
	fwd := NewUserspaceForwarder(loader)

	fwd.RuleMap().AddRule(&ForwardingRule{
		DstPort: 22,
		Action:  ActionDrop,
	})

	target, err := fwd.ProcessPacket("1.1.1.1", "2.2.2.2", 12345, 22, 6, 100)
	if err != nil {
		t.Fatal(err)
	}
	if target != "" {
		t.Fatalf("expected empty target for drop, got %q", target)
	}

	stats := loader.Stats()
	if stats.PacketsDropped != 1 {
		t.Fatalf("expected 1 dropped, got %d", stats.PacketsDropped)
	}
}

func TestUserspaceForwarder_ProcessNoRule(t *testing.T) {
	loader := NewLoader(DefaultEBPFConfig())
	fwd := NewUserspaceForwarder(loader)

	target, err := fwd.ProcessPacket("1.1.1.1", "2.2.2.2", 12345, 9999, 6, 200)
	if err != nil {
		t.Fatal(err)
	}
	if target != "" {
		t.Fatalf("expected empty target for no-rule, got %q", target)
	}

	stats := loader.Stats()
	if stats.PacketsForwarded != 1 {
		t.Fatalf("expected 1 forwarded (default), got %d", stats.PacketsForwarded)
	}
}

func TestIPMatches(t *testing.T) {
	tests := []struct {
		ruleAddr string
		packetIP string
		expected bool
	}{
		{"10.0.0.1", "10.0.0.1", true},
		{"10.0.0.1", "10.0.0.2", false},
		{"192.168.1.0/24", "192.168.1.50", true},
		{"192.168.1.0/24", "192.168.2.1", false},
		{"10.0.0.0/8", "10.255.255.255", true},
	}

	for _, tt := range tests {
		result := ipMatches(tt.ruleAddr, tt.packetIP)
		if result != tt.expected {
			t.Errorf("ipMatches(%q, %q) = %v, want %v", tt.ruleAddr, tt.packetIP, result, tt.expected)
		}
	}
}
