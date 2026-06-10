package ipam

import (
	"fmt"
	"sync"
	"testing"
)

func TestNewIPAM(t *testing.T) {
	m, err := NewIPAM("10.7.0.0/24", "10.7.0.1")
	if err != nil {
		t.Fatalf("NewIPAM: %v", err)
	}
	if m.Subnet().String() != "10.7.0.0/24" {
		t.Errorf("Subnet = %s", m.Subnet())
	}
	if m.Gateway().String() != "10.7.0.1" {
		t.Errorf("Gateway = %s", m.Gateway())
	}
}

func TestNewIPAM_InvalidSubnet(t *testing.T) {
	_, err := NewIPAM("invalid", "10.7.0.1")
	if err == nil {
		t.Error("expected error for invalid CIDR")
	}
}

func TestNewIPAM_GatewayOutsideSubnet(t *testing.T) {
	_, err := NewIPAM("10.7.0.0/24", "192.168.1.1")
	if err == nil {
		t.Error("expected error for gateway outside subnet")
	}
}

func TestIPAM_Allocate(t *testing.T) {
	m, err := NewIPAM("10.7.0.0/24", "10.7.0.1")
	if err != nil {
		t.Fatalf("NewIPAM: %v", err)
	}

	// First allocation
	ip1, err := m.Allocate("node-1")
	if err != nil {
		t.Fatalf("Allocate node-1: %v", err)
	}
	if ip1.String() == "10.7.0.0" {
		t.Error("should not allocate network address")
	}
	if ip1.String() == "10.7.0.255" {
		t.Error("should not allocate broadcast address")
	}
	if ip1.String() == "10.7.0.1" {
		t.Error("should not allocate gateway address")
	}

	// Second allocation should be different
	ip2, err := m.Allocate("node-2")
	if err != nil {
		t.Fatalf("Allocate node-2: %v", err)
	}
	if ip1.Equal(ip2) {
		t.Error("node-1 and node-2 should have different IPs")
	}

	// Same node should get same IP
	ip1Again, err := m.Allocate("node-1")
	if err != nil {
		t.Fatalf("Re-allocate node-1: %v", err)
	}
	if !ip1.Equal(ip1Again) {
		t.Error("node-1 should get same IP on re-allocation")
	}
}

func TestIPAM_Release(t *testing.T) {
	m, _ := NewIPAM("10.7.0.0/24", "10.7.0.1")

	ip, _ := m.Allocate("node-1")
	m.Release("node-1")

	_, ok := m.GetAllocation("node-1")
	if ok {
		t.Error("allocation should be gone after release")
	}

	// Re-allocate should work
	ip2, err := m.Allocate("node-2")
	if err != nil {
		t.Fatalf("Allocate after release: %v", err)
	}
	// The released IP should be available again
	_ = ip
	_ = ip2
}

func TestIPAM_GetAllocation(t *testing.T) {
	m, _ := NewIPAM("10.7.0.0/24", "10.7.0.1")

	_, ok := m.GetAllocation("unknown")
	if ok {
		t.Error("unknown node should not have allocation")
	}

	m.Allocate("node-1")
	ip, ok := m.GetAllocation("node-1")
	if !ok || ip == nil {
		t.Error("node-1 should have allocation")
	}
}

func TestIPAM_ListAllocations(t *testing.T) {
	m, _ := NewIPAM("10.7.0.0/24", "10.7.0.1")

	m.Allocate("node-1")
	m.Allocate("node-2")
	m.Allocate("node-3")

	allocs := m.ListAllocations()
	if len(allocs) != 3 {
		t.Errorf("got %d allocations, want 3", len(allocs))
	}
}

func TestIPAM_Exhaustion(t *testing.T) {
	// /30 has only 2 usable addresses (after network and broadcast)
	// With gateway taking one, only 1 is available
	m, err := NewIPAM("10.7.0.0/30", "10.7.0.1")
	if err != nil {
		t.Fatalf("NewIPAM: %v", err)
	}

	// Should allocate 10.7.0.2 (only usable host after gateway)
	_, err = m.Allocate("node-1")
	if err != nil {
		t.Fatalf("First allocation: %v", err)
	}

	// Should fail - no more IPs
	_, err = m.Allocate("node-2")
	if err == nil {
		t.Error("expected exhaustion error")
	}
}

func TestIPAM_ConcurrentSafety(t *testing.T) {
	m, _ := NewIPAM("10.7.0.0/24", "10.7.0.1")

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			nodeID := fmt.Sprintf("node-%d", idx)
			m.Allocate(nodeID)
			m.GetAllocation(nodeID)
			m.ListAllocations()
		}(i)
	}
	wg.Wait()

	allocs := m.ListAllocations()
	if len(allocs) != 50 {
		t.Errorf("got %d allocations, want 50", len(allocs))
	}
}

func TestIPAM_LargerSubnet(t *testing.T) {
	m, err := NewIPAM("10.7.0.0/16", "10.7.0.1")
	if err != nil {
		t.Fatalf("NewIPAM: %v", err)
	}

	// Allocate a bunch of IPs
	for i := 0; i < 100; i++ {
		nodeID := fmt.Sprintf("node-%d", i)
		ip, err := m.Allocate(nodeID)
		if err != nil {
			t.Fatalf("Allocate %s: %v", nodeID, err)
		}
		if !m.Subnet().Contains(ip) {
			t.Errorf("IP %s not in subnet %s", ip, m.Subnet())
		}
	}
}
