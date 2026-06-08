package edge

import (
	"context"
	"fmt"
	"net"
	"strings"
	"testing"
	"time"
)

func TestEdgeNode_CreateAndStatus(t *testing.T) {
	node, err := NewEdgeNode("edge-1", "127.0.0.1:4433", "us-east", RoleFull, 1000)
	if err != nil {
		t.Fatalf("NewEdgeNode: %v", err)
	}
	if node.ID != "edge-1" {
		t.Errorf("ID = %q, want %q", node.ID, "edge-1")
	}
	if node.GetStatus() != StatusHealthy {
		t.Errorf("initial status = %q, want %q", node.GetStatus(), StatusHealthy)
	}
	if node.Region != "us-east" {
		t.Errorf("Region = %q, want %q", node.Region, "us-east")
	}

	node.SetStatus(StatusUnhealthy)
	if node.GetStatus() != StatusUnhealthy {
		t.Errorf("status after set = %q, want %q", node.GetStatus(), StatusUnhealthy)
	}

	t.Logf("EdgeNode created: id=%s addr=%s region=%s role=%s status=%s",
		node.ID, node.Addr, node.Region, node.Role, node.GetStatus())
}

func TestEdgeNode_Validation(t *testing.T) {
	tests := []struct {
		name    string
		id      string
		addr    string
		wantErr bool
	}{
		{"valid", "n1", "127.0.0.1:4433", false},
		{"empty id", "", "127.0.0.1:4433", true},
		{"empty addr", "n1", "", true},
		{"bad addr", "n1", "not-an-addr", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewEdgeNode(tt.id, tt.addr, "test", RoleRelay, 100)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewEdgeNode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEdgeNode_HealthTracking(t *testing.T) {
	node, _ := NewEdgeNode("edge-h", "127.0.0.1:4433", "eu-west", RoleRelay, 500)

	// Record successes
	node.RecordSuccess(10 * time.Millisecond)
	node.RecordSuccess(15 * time.Millisecond)
	if node.ConsecutiveOKs() != 2 {
		t.Errorf("ConsecutiveOKs = %d, want 2", node.ConsecutiveOKs())
	}
	if node.ConsecutiveFails() != 0 {
		t.Errorf("ConsecutiveFails = %d, want 0", node.ConsecutiveFails())
	}

	// Record failure resets OK counter
	node.RecordFailure()
	if node.ConsecutiveOKs() != 0 {
		t.Errorf("ConsecutiveOKs after fail = %d, want 0", node.ConsecutiveOKs())
	}
	if node.ConsecutiveFails() != 1 {
		t.Errorf("ConsecutiveFails = %d, want 1", node.ConsecutiveFails())
	}

	// Record success resets fail counter
	node.RecordSuccess(5 * time.Millisecond)
	if node.ConsecutiveFails() != 0 {
		t.Errorf("ConsecutiveFails after success = %d, want 0", node.ConsecutiveFails())
	}

	t.Logf("Health tracking: latency=%v", node.Latency)
}

func TestRegistry_RegisterAndGet(t *testing.T) {
	reg := NewRegistry()

	var registered []*EdgeNode
	reg.OnRegister(func(n *EdgeNode) {
		registered = append(registered, n)
	})

	n1, _ := NewEdgeNode("e1", "127.0.0.1:4433", "us-east", RoleFull, 1000)
	n2, _ := NewEdgeNode("e2", "127.0.0.1:4434", "eu-west", RoleRelay, 500)

	if err := reg.Register(n1); err != nil {
		t.Fatalf("Register n1: %v", err)
	}
	if err := reg.Register(n2); err != nil {
		t.Fatalf("Register n2: %v", err)
	}

	// Duplicate
	if err := reg.Register(n1); err == nil {
		t.Error("expected error for duplicate registration")
	}

	if reg.Count() != 2 {
		t.Errorf("Count = %d, want 2", reg.Count())
	}

	got, ok := reg.Get("e1")
	if !ok || got.ID != "e1" {
		t.Errorf("Get e1: ok=%v, got=%v", ok, got)
	}

	_, ok = reg.Get("nonexistent")
	if ok {
		t.Error("expected not found for nonexistent node")
	}

	if len(registered) != 2 {
		t.Errorf("registered callbacks = %d, want 2", len(registered))
	}
}

func TestRegistry_ListAndFilter(t *testing.T) {
	reg := NewRegistry()

	nodes := []struct {
		id, addr, region string
		role             NodeRole
	}{
		{"e1", "127.0.0.1:4433", "us-east", RoleFull},
		{"e2", "127.0.0.1:4434", "us-east", RoleRelay},
		{"e3", "127.0.0.1:4435", "eu-west", RoleFull},
		{"e4", "127.0.0.1:4436", "ap-south", RoleAccelerator},
	}

	for _, n := range nodes {
		node, _ := NewEdgeNode(n.id, n.addr, n.region, n.role, 100)
		_ = reg.Register(node)
	}

	// ListByRegion
	useast := reg.ListByRegion("us-east")
	if len(useast) != 2 {
		t.Errorf("ListByRegion us-east = %d, want 2", len(useast))
	}

	// Regions
	regions := reg.Regions()
	if len(regions) != 3 {
		t.Errorf("Regions = %d, want 3: %v", len(regions), regions)
	}

	// Mark one unhealthy
	e1, _ := reg.Get("e1")
	e1.SetStatus(StatusUnhealthy)

	healthy := reg.ListHealthy()
	if len(healthy) != 3 {
		t.Errorf("ListHealthy = %d, want 3", len(healthy))
	}

	if reg.CountHealthy() != 3 {
		t.Errorf("CountHealthy = %d, want 3", reg.CountHealthy())
	}

	// Deregister
	var deregistered []*EdgeNode
	reg.OnDeregister(func(n *EdgeNode) {
		deregistered = append(deregistered, n)
	})

	if err := reg.Deregister("e1"); err != nil {
		t.Fatalf("Deregister: %v", err)
	}
	if reg.Count() != 3 {
		t.Errorf("Count after deregister = %d, want 3", reg.Count())
	}
	if len(deregistered) != 1 {
		t.Errorf("deregistered callbacks = %d, want 1", len(deregistered))
	}
}

func TestRegistry_ListHealthySortedByLatency(t *testing.T) {
	reg := NewRegistry()

	latencies := []time.Duration{50 * time.Millisecond, 10 * time.Millisecond, 30 * time.Millisecond}
	for i, lat := range latencies {
		node, _ := NewEdgeNode(fmt.Sprintf("n%d", i), fmt.Sprintf("127.0.0.1:%d", 5000+i), "us-east", RoleFull, 100)
		node.Latency = lat
		_ = reg.Register(node)
	}

	healthy := reg.ListHealthy()
	if len(healthy) != 3 {
		t.Fatalf("ListHealthy = %d, want 3", len(healthy))
	}

	// Should be sorted by latency ascending
	for i := 1; i < len(healthy); i++ {
		if healthy[i].Latency < healthy[i-1].Latency {
			t.Errorf("not sorted: healthy[%d].Latency=%v < healthy[%d].Latency=%v",
				i, healthy[i].Latency, i-1, healthy[i-1].Latency)
		}
	}

	t.Logf("Sorted by latency: %v, %v, %v", healthy[0].Latency, healthy[1].Latency, healthy[2].Latency)
}

func TestHealthChecker_ProbeReachable(t *testing.T) {
	// Start a TCP listener to simulate a reachable node
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	cfg := DefaultEdgeConfig()
	cfg.HeartbeatInterval = 100 * time.Millisecond
	cfg.UnhealthyThreshold = 2
	cfg.HealthyThreshold = 1

	reg := NewRegistry()
	node, _ := NewEdgeNode("probe-1", ln.Addr().String(), "test", RoleFull, 100)
	_ = reg.Register(node)

	checker := NewHealthChecker(cfg, reg)
	ctx, cancel := context.WithCancel(context.Background())
	checker.Start(ctx)

	// Wait for at least 3 full probe cycles (100ms interval → 300ms+)
	// Add buffer for goroutine scheduling
	time.Sleep(800 * time.Millisecond)
	// Let any in-flight probe complete before stopping
	time.Sleep(50 * time.Millisecond)
	checker.Stop()
	cancel()

	if node.GetStatus() != StatusHealthy {
		t.Errorf("status = %q, want %q", node.GetStatus(), StatusHealthy)
	}
	// ConsecutiveOKs uses atomic int - safe to read across goroutines
	oks := node.ConsecutiveOKs()
	if oks < 1 {
		t.Errorf("ConsecutiveOKs = %d, want >= 1 (timing-sensitive, check DirectProbe for deterministic result)", oks)
	}

	t.Logf("Health check: status=%s oks=%d", node.GetStatus(), oks)
}

func TestHealthChecker_ProbeUnreachable(t *testing.T) {
	cfg := DefaultEdgeConfig()
	cfg.HeartbeatInterval = 50 * time.Millisecond
	cfg.HealthCheckTimeout = 100 * time.Millisecond
	cfg.UnhealthyThreshold = 2
	cfg.DeregisterTimeout = 200 * time.Millisecond

	reg := NewRegistry()
	// Use a port that nobody is listening on
	node, _ := NewEdgeNode("dead-1", "127.0.0.1:1", "test", RoleFull, 100)
	_ = reg.Register(node)

	checker := NewHealthChecker(cfg, reg)
	ctx, cancel := context.WithCancel(context.Background())
	checker.Start(ctx)

	// Wait for unhealthy + deregister
	time.Sleep(500 * time.Millisecond)
	checker.Stop()
	cancel()

	// Node should have been deregistered
	if reg.Count() != 0 {
		t.Errorf("Count = %d, want 0 (auto-deregistered)", reg.Count())
	}

	t.Logf("Unreachable node: status=%s fails=%d", node.GetStatus(), node.ConsecutiveFails())
}

func TestHealthChecker_DirectProbe(t *testing.T) {
	// Deterministic test: directly invoke probeNode without relying on timer
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer ln.Close()

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	cfg := DefaultEdgeConfig()
	reg := NewRegistry()
	node, _ := NewEdgeNode("direct-1", ln.Addr().String(), "test", RoleFull, 100)
	_ = reg.Register(node)

	checker := NewHealthChecker(cfg, reg)
	ctx := context.Background()

	// Directly call probeNode 3 times
	for i := 0; i < 3; i++ {
		checker.probeNode(ctx, node)
	}

	if node.GetStatus() != StatusHealthy {
		t.Errorf("status = %q, want %q", node.GetStatus(), StatusHealthy)
	}
	if node.ConsecutiveOKs() != 3 {
		t.Errorf("ConsecutiveOKs = %d, want 3", node.ConsecutiveOKs())
	}
	t.Logf("Direct probe: status=%s oks=%d", node.GetStatus(), node.ConsecutiveOKs())
}

func TestDeployCompose_Generate(t *testing.T) {
	cfg := DeployConfig{
		NodeID:      "edge-tokyo-1",
		Addr:        "203.0.113.10:4433",
		Region:      "ap-northeast-1",
		Role:        "full",
		RelayPort:   4433,
		ControlAddr: "control.nextunnel.io:9090",
	}

	compose, err := GenerateDeployCompose(cfg)
	if err != nil {
		t.Fatalf("GenerateDeployCompose: %v", err)
	}

	// Verify key fields are present
	checks := []string{
		"nexedge-edge-tokyo-1",
		"ap-northeast-1",
		"4433:4433",
		"control.nextunnel.io:9090",
	}
	for _, check := range checks {
		if !strings.Contains(compose, check) {
			t.Errorf("compose missing %q", check)
		}
	}

	t.Logf("Generated compose:\n%s", compose)
}

func TestDeployCompose_Validation(t *testing.T) {
	_, err := GenerateDeployCompose(DeployConfig{})
	if err == nil {
		t.Error("expected error for empty node_id")
	}

	_, err = GenerateDeployCompose(DeployConfig{NodeID: "n1"})
	if err == nil {
		t.Error("expected error for empty region")
	}
}

func TestDeployManifest_Create(t *testing.T) {
	cfg := DeployConfig{
		NodeID:      "edge-eu-1",
		Region:      "eu-west",
		ControlAddr: "control.nextunnel.io:9090",
	}

	manifest, err := CreateDeployManifest(cfg)
	if err != nil {
		t.Fatalf("CreateDeployManifest: %v", err)
	}

	if manifest.NodeID != "edge-eu-1" {
		t.Errorf("NodeID = %q, want %q", manifest.NodeID, "edge-eu-1")
	}
	if manifest.Region != "eu-west" {
		t.Errorf("Region = %q, want %q", manifest.Region, "eu-west")
	}
	if !strings.Contains(manifest.Instructions, "edge-eu-1") {
		t.Error("instructions missing node ID")
	}
	if !strings.Contains(manifest.ComposeFile, "nexedge-edge-eu-1") {
		t.Error("compose file missing service name")
	}

	t.Logf("Manifest instructions:\n%s", manifest.Instructions)
}
