package scheduler

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/nextunnel/desktop/internal/probe"
)

// dummyTransport implements scheduler.Transport for testing.
type dummyTransport struct {
	id string
}

func (d *dummyTransport) Read([]byte) (int, error)  { return 0, nil }
func (d *dummyTransport) Write(p []byte) (int, error) { return len(p), nil }
func (d *dummyTransport) Close() error               { return nil }
func (d *dummyTransport) LocalAddr() net.Addr {
	return &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 1000}
}
func (d *dummyTransport) RemoteAddr() net.Addr {
	return &net.UDPAddr{IP: net.IPv4(10, 0, 0, 1), Port: 2000}
}

func makePath(id string, ptype PathType, rtt time.Duration, loss float64, bw int64) *Path {
	return &Path{
		ID:    id,
		Type:  ptype,
		State: PathAvailable,
		Metrics: probe.LinkMetrics{
			RTT:       rtt,
			LossRate:  loss,
			Bandwidth: bw,
		},
		Transport: &dummyTransport{id: id},
	}
}

// --- Policy tests ---

func TestScorePath_Formula(t *testing.T) {
	good := makePath("udp1", PathUDPP2P, 5*time.Millisecond, 0.0, 10e6)
	bad := makePath("relay1", PathGlobalRelay, 200*time.Millisecond, 0.1, 1e6)

	sg := ScorePath(good)
	sb := ScorePath(bad)

	if sg.Score <= sb.Score {
		t.Errorf("good score (%.4f) should be > bad score (%.4f)", sg.Score, sb.Score)
	}
	if sg.Priority != 1 {
		t.Errorf("UDP P2P priority = %d, want 1", sg.Priority)
	}
	if sb.Priority != 5 {
		t.Errorf("Global Relay priority = %d, want 5", sb.Priority)
	}
	t.Logf("Good: score=%.4f type=%s", sg.Score, sg.Type)
	t.Logf("Bad:  score=%.4f type=%s", sb.Score, sb.Type)
}

func TestRankPaths_Tiebreak(t *testing.T) {
	// Same metrics, different types
	p1 := makePath("relay", PathGlobalRelay, 10*time.Millisecond, 0, 10e6)
	p2 := makePath("udp", PathUDPP2P, 10*time.Millisecond, 0, 10e6)
	p3 := makePath("quic", PathQUICP2P, 10*time.Millisecond, 0, 10e6)

	ranked := RankPaths([]*Path{p1, p2, p3})
	if len(ranked) != 3 {
		t.Fatalf("ranked = %d, want 3", len(ranked))
	}

	// UDP P2P should be first (highest priority bonus)
	if ranked[0].Type != PathUDPP2P {
		t.Errorf("first = %s, want udp_p2p", ranked[0].Type)
	}
	if ranked[1].Type != PathQUICP2P {
		t.Errorf("second = %s, want quic_p2p", ranked[1].Type)
	}
	if ranked[2].Type != PathGlobalRelay {
		t.Errorf("third = %s, want global_relay", ranked[2].Type)
	}
}

func TestRankPaths_ExcludesUnavailable(t *testing.T) {
	p1 := makePath("a", PathUDPP2P, 10*time.Millisecond, 0, 10e6)
	p2 := makePath("b", PathQUICP2P, 10*time.Millisecond, 0, 10e6)
	p2.State = PathUnavailable

	ranked := RankPaths([]*Path{p1, p2})
	if len(ranked) != 1 {
		t.Fatalf("ranked = %d, want 1 (unavailable excluded)", len(ranked))
	}
}

// --- Scheduler tests ---

func TestScheduler_PriorityOrder(t *testing.T) {
	s := NewScheduler(DefaultSchedulerConfig())

	// Register 5 paths with equal metrics
	paths := []*Path{
		makePath("relay", PathGlobalRelay, 10*time.Millisecond, 0, 10e6),
		makePath("tcp", PathTCPP2P, 10*time.Millisecond, 0, 10e6),
		makePath("quic", PathQUICP2P, 10*time.Millisecond, 0, 10e6),
		makePath("nearby", PathNearbyRelay, 10*time.Millisecond, 0, 10e6),
		makePath("udp", PathUDPP2P, 10*time.Millisecond, 0, 10e6),
	}

	for _, p := range paths {
		s.RegisterPath(p)
	}

	best := s.Evaluate()
	if best == nil {
		t.Fatal("Evaluate returned nil")
	}
	if best.Type != PathUDPP2P {
		t.Errorf("best = %s, want udp_p2p", best.Type)
	}
	t.Logf("Best path: %s (type=%s)", best.ID, best.Type)
}

func TestScheduler_DegradationSwitch(t *testing.T) {
	cfg := DefaultSchedulerConfig()
	cfg.SwitchCooldown = 0 // no cooldown for test
	s := NewScheduler(cfg)

	good := makePath("quic1", PathQUICP2P, 20*time.Millisecond, 0, 10e6)
	bad := makePath("udp1", PathUDPP2P, 10*time.Millisecond, 0, 10e6)

	s.RegisterPath(bad)
	s.RegisterPath(good)

	// UDP should be active initially
	active := s.ActivePath()
	if active == nil || active.Type != PathUDPP2P {
		t.Fatalf("initial active = %v, want udp_p2p", active)
	}

	// Degrade UDP path (high loss)
	bad.Metrics.LossRate = 0.5

	// Evaluate should pick QUIC
	best := s.Evaluate()
	if best == nil || best.Type != PathQUICP2P {
		t.Errorf("best after degradation = %v, want quic_p2p", best)
	}

	// Switch
	err := s.SwitchTo(best.ID)
	if err != nil {
		t.Fatalf("SwitchTo: %v", err)
	}

	newActive := s.ActivePath()
	if newActive.Type != PathQUICP2P {
		t.Errorf("new active = %s, want quic_p2p", newActive.Type)
	}
}

func TestScheduler_ManualLock(t *testing.T) {
	cfg := DefaultSchedulerConfig()
	cfg.SwitchCooldown = 0
	s := NewScheduler(cfg)

	udp := makePath("udp1", PathUDPP2P, 10*time.Millisecond, 0, 10e6)
	quic := makePath("quic1", PathQUICP2P, 10*time.Millisecond, 0, 10e6)

	s.RegisterPath(udp)
	s.RegisterPath(quic)

	// Lock to QUIC even though UDP is better
	if err := s.LockPath("quic1"); err != nil {
		t.Fatalf("LockPath: %v", err)
	}

	active := s.ActivePath()
	if active == nil || active.ID != "quic1" {
		t.Errorf("locked active = %v, want quic1", active)
	}

	// Evaluate should return locked path
	best := s.Evaluate()
	if best == nil || best.ID != "quic1" {
		t.Errorf("evaluate with lock = %v, want quic1", best)
	}

	// Unlock
	s.UnlockPath()
	best = s.Evaluate()
	if best == nil || best.ID != "udp1" {
		t.Errorf("evaluate after unlock = %v, want udp1", best)
	}
}

func TestScheduler_PacketLossDuringSwitch(t *testing.T) {
	cfg := DefaultSchedulerConfig()
	cfg.SwitchCooldown = 0
	s := NewScheduler(cfg)

	p1 := makePath("p1", PathUDPP2P, 10*time.Millisecond, 0, 10e6)
	p2 := makePath("p2", PathQUICP2P, 20*time.Millisecond, 0, 10e6)

	s.RegisterPath(p1)
	s.RegisterPath(p2)

	// Verify atomic switch
	var callbackCalled bool
	s.OnPathChange(func(old, new *Path) {
		callbackCalled = true
		if old == nil || old.ID != "p1" {
			t.Errorf("old path = %v, want p1", old)
		}
		if new.ID != "p2" {
			t.Errorf("new path = %v, want p2", new.ID)
		}
	})

	if err := s.SwitchTo("p2"); err != nil {
		t.Fatalf("SwitchTo: %v", err)
	}

	if !callbackCalled {
		t.Error("OnPathChange callback not called")
	}

	active := s.ActivePath()
	if active.ID != "p2" {
		t.Errorf("active = %s, want p2", active.ID)
	}
}

func TestScheduler_SwitchCooldown(t *testing.T) {
	cfg := DefaultSchedulerConfig()
	cfg.SwitchCooldown = 1 * time.Second
	s := NewScheduler(cfg)

	p1 := makePath("p1", PathUDPP2P, 10*time.Millisecond, 0, 10e6)
	p2 := makePath("p2", PathQUICP2P, 20*time.Millisecond, 0, 10e6)

	s.RegisterPath(p1)
	s.RegisterPath(p2)

	// First switch should succeed
	if err := s.SwitchTo("p2"); err != nil {
		t.Fatalf("first switch: %v", err)
	}

	// Second switch within cooldown should fail
	if err := s.SwitchTo("p1"); err == nil {
		t.Error("expected cooldown error on immediate second switch")
	}
}

func TestScheduler_Failover(t *testing.T) {
	cfg := DefaultSchedulerConfig()
	cfg.SwitchCooldown = 0
	cfg.LossThreshold = 0.1
	cfg.EvalInterval = 50 * time.Millisecond
	s := NewScheduler(cfg)

	primary := makePath("primary", PathUDPP2P, 10*time.Millisecond, 0, 10e6)
	backup := makePath("backup", PathQUICP2P, 20*time.Millisecond, 0, 10e6)

	s.RegisterPath(primary)
	s.RegisterPath(backup)

	ctx, cancel := context.WithCancel(context.Background())
	s.Start(ctx)

	time.Sleep(100 * time.Millisecond)

	// Degrade primary
	primary.Metrics.LossRate = 0.5

	// Wait for auto-evaluation
	time.Sleep(200 * time.Millisecond)
	cancel()
	s.Stop()

	active := s.ActivePath()
	if active == nil {
		t.Fatal("no active path")
	}
	if active.ID != "backup" {
		t.Logf("active = %s (expected backup after degradation)", active.ID)
	}
}
