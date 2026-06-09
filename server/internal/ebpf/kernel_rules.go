package ebpf

import (
	"fmt"
	"net"
	"strconv"
	"strings"
)

const (
	kernelActionPass     uint32 = 1
	kernelActionDrop     uint32 = 2
	kernelActionRedirect uint32 = 3

	kernelForwardTargetPrefix = "ifindex:"
)

// xdpL4RuleKey 与 xdp_forwarder.c 中的 l4_rule_key 保持二进制布局一致。
type xdpL4RuleKey struct {
	Protocol uint8
	Pad      uint8
	DstPort  uint16
}

// xdpL4RuleValue 与 xdp_forwarder.c 中的 l4_rule_value 保持二进制布局一致。
type xdpL4RuleValue struct {
	Action  uint32
	IfIndex uint32
}

// xdpKernelStats 与 xdp_forwarder.c 中的 xdp_stats 保持二进制布局一致。
type xdpKernelStats struct {
	PacketsForwarded uint64
	BytesForwarded   uint64
	PacketsDropped   uint64
}

type xdpKernelRulePlan struct {
	RuleID uint32
	Key    xdpL4RuleKey
	Value  xdpL4RuleValue
}

// encodeKernelRule 将通用规则收敛成 XDP 可安全执行的 L4 fast-path 规则。
// 复杂 CIDR、源端口和优先级规则仍由用户态 RuleMap 处理，避免扩大内核态匹配范围。
func encodeKernelRule(rule *ForwardingRule) (xdpL4RuleKey, xdpL4RuleValue, bool, string) {
	var key xdpL4RuleKey
	var value xdpL4RuleValue
	if rule == nil {
		return key, value, false, "rule is nil"
	}
	if rule.Protocol != 6 && rule.Protocol != 17 {
		return key, value, false, "only TCP/UDP protocols are supported by XDP fast path"
	}
	if rule.DstPort == 0 {
		return key, value, false, "destination port is required for XDP fast path"
	}
	if rule.SrcPort != 0 || rule.SrcAddr != "" || rule.DstAddr != "" {
		return key, value, false, "address and source-port matching remain in userspace"
	}

	key = xdpL4RuleKey{
		Protocol: rule.Protocol,
		DstPort:  rule.DstPort,
	}

	switch rule.Action {
	case ActionPass:
		value.Action = kernelActionPass
	case ActionDrop:
		value.Action = kernelActionDrop
	case ActionForward:
		ifIndex, ok := resolveForwardTargetIfIndex(rule.Target)
		if !ok {
			return key, value, false, "forward target must be ifindex:<n> or an interface name"
		}
		value.Action = kernelActionRedirect
		value.IfIndex = ifIndex
	default:
		return key, value, false, fmt.Sprintf("unsupported action %q", rule.Action)
	}

	return key, value, true, ""
}

// buildKernelRulePlan 只下沉不会改变 RuleMap 优先级语义的规则。
// 如果更高优先级的复杂规则可能命中同一个 L4 key，后续简单规则必须留在用户态。
func buildKernelRulePlan(rules []*ForwardingRule) []xdpKernelRulePlan {
	plannedRules := make([]xdpKernelRulePlan, 0, len(rules))
	plannedKeys := make(map[xdpL4RuleKey]struct{})
	priorRules := make([]*ForwardingRule, 0, len(rules))

	for _, rule := range rules {
		key, value, ok, _ := encodeKernelRule(rule)
		if !ok {
			priorRules = append(priorRules, rule)
			continue
		}
		if _, exists := plannedKeys[key]; exists {
			priorRules = append(priorRules, rule)
			continue
		}
		if kernelKeyBlockedByPriorRules(key, priorRules) {
			priorRules = append(priorRules, rule)
			continue
		}

		plannedRules = append(plannedRules, xdpKernelRulePlan{
			RuleID: rule.ID,
			Key:    key,
			Value:  value,
		})
		plannedKeys[key] = struct{}{}
		priorRules = append(priorRules, rule)
	}

	return plannedRules
}

func kernelKeyBlockedByPriorRules(key xdpL4RuleKey, priorRules []*ForwardingRule) bool {
	for _, priorRule := range priorRules {
		priorKey, _, ok, _ := encodeKernelRule(priorRule)
		if ok {
			if priorKey == key {
				return true
			}
			continue
		}
		if ruleMayMatchKernelKey(priorRule, key) {
			return true
		}
	}
	return false
}

func ruleMayMatchKernelKey(rule *ForwardingRule, key xdpL4RuleKey) bool {
	if rule == nil {
		return false
	}
	if rule.Protocol != 0 && rule.Protocol != key.Protocol {
		return false
	}
	if rule.DstPort != 0 && rule.DstPort != key.DstPort {
		return false
	}
	return true
}

func resolveForwardTargetIfIndex(target string) (uint32, bool) {
	normalizedTarget := strings.TrimSpace(target)
	if normalizedTarget == "" {
		return 0, false
	}

	if strings.HasPrefix(normalizedTarget, kernelForwardTargetPrefix) {
		rawIndex := strings.TrimPrefix(normalizedTarget, kernelForwardTargetPrefix)
		index, err := strconv.ParseUint(rawIndex, 10, 32)
		if err != nil || index == 0 {
			return 0, false
		}
		return uint32(index), true
	}

	iface, err := net.InterfaceByName(normalizedTarget)
	if err != nil || iface.Index <= 0 {
		return 0, false
	}
	return uint32(iface.Index), true
}
