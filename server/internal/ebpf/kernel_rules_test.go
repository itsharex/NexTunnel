package ebpf

import "testing"

func TestEncodeKernelRule_DropFastPath(t *testing.T) {
	key, value, ok, reason := encodeKernelRule(&ForwardingRule{
		ID:       7,
		DstPort:  443,
		Protocol: 6,
		Action:   ActionDrop,
	})
	if !ok {
		t.Fatalf("encodeKernelRule failed: %s", reason)
	}
	if key.Protocol != 6 || key.DstPort != 443 {
		t.Fatalf("key = %+v, want protocol=6 dst_port=443", key)
	}
	if value.Action != kernelActionDrop || value.IfIndex != 0 {
		t.Fatalf("value = %+v, want drop action", value)
	}
}

func TestEncodeKernelRule_RedirectFastPath(t *testing.T) {
	key, value, ok, reason := encodeKernelRule(&ForwardingRule{
		DstPort:  8080,
		Protocol: 17,
		Action:   ActionForward,
		Target:   "ifindex:12",
	})
	if !ok {
		t.Fatalf("encodeKernelRule failed: %s", reason)
	}
	if key.Protocol != 17 || key.DstPort != 8080 {
		t.Fatalf("key = %+v, want protocol=17 dst_port=8080", key)
	}
	if value.Action != kernelActionRedirect || value.IfIndex != 12 {
		t.Fatalf("value = %+v, want redirect to ifindex 12", value)
	}
}

func TestEncodeKernelRule_ComplexRuleStaysUserspace(t *testing.T) {
	_, _, ok, reason := encodeKernelRule(&ForwardingRule{
		SrcAddr:  "10.0.0.0/8",
		DstPort:  80,
		Protocol: 6,
		Action:   ActionDrop,
	})
	if ok {
		t.Fatal("expected CIDR rule to stay in userspace")
	}
	if reason == "" {
		t.Fatal("expected reason for userspace fallback")
	}
}

func TestBuildKernelRulePlan_SkipsRuleShadowedByComplexPriority(t *testing.T) {
	rules := []*ForwardingRule{
		{
			ID:       1,
			SrcAddr:  "10.0.0.0/8",
			DstPort:  443,
			Protocol: 6,
			Action:   ActionDrop,
			Priority: 1,
		},
		{
			ID:       2,
			DstPort:  443,
			Protocol: 6,
			Action:   ActionPass,
			Priority: 10,
		},
	}

	plan := buildKernelRulePlan(rules)
	if len(plan) != 0 {
		t.Fatalf("expected shadowed L4 rule to stay in userspace, got %d planned rules", len(plan))
	}
}

func TestBuildKernelRulePlan_AllowsIndependentFastPathRule(t *testing.T) {
	rules := []*ForwardingRule{
		{
			ID:       1,
			SrcAddr:  "10.0.0.0/8",
			DstPort:  443,
			Protocol: 6,
			Action:   ActionDrop,
			Priority: 1,
		},
		{
			ID:       2,
			DstPort:  8443,
			Protocol: 6,
			Action:   ActionPass,
			Priority: 10,
		},
	}

	plan := buildKernelRulePlan(rules)
	if len(plan) != 1 {
		t.Fatalf("expected one independent L4 rule, got %d", len(plan))
	}
	if plan[0].RuleID != 2 || plan[0].Key.DstPort != 8443 || plan[0].Value.Action != kernelActionPass {
		t.Fatalf("unexpected plan entry: %+v", plan[0])
	}
}

func TestResolveForwardTargetIfIndex(t *testing.T) {
	ifIndex, ok := resolveForwardTargetIfIndex("ifindex:3")
	if !ok || ifIndex != 3 {
		t.Fatalf("ifIndex = %d, ok = %v, want 3 true", ifIndex, ok)
	}

	if _, ok := resolveForwardTargetIfIndex("ifindex:0"); ok {
		t.Fatal("expected zero ifindex to be rejected")
	}
}
