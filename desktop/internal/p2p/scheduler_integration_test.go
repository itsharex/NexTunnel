package p2p_test

import (
	"context"
	"net"
	"testing"
	"time"

	"github.com/nextunnel/desktop/internal/migration"
	"github.com/nextunnel/desktop/internal/p2p"
	"github.com/nextunnel/desktop/internal/relay"
	"github.com/nextunnel/desktop/internal/scheduler"
	"github.com/nextunnel/pkg/protocol"
)

// TestSchedulerEngineIntegration verifies that the P2P Engine can use
// PathManager/RelaySelector/MigrationController interfaces to drive
// real path switching via the scheduler, relay manager, and migrator.
func TestSchedulerEngineIntegration(t *testing.T) {
	// Create engine
	engine, err := p2p.NewEngine(p2p.EngineConfig{
		ClientID: "test-client",
	})
	if err != nil {
		t.Fatalf("NewEngine: %v", err)
	}
	defer engine.Close()

	// 1. Create scheduler with multiple paths (short cooldown for testing)
	sched := scheduler.NewScheduler(scheduler.DefaultSchedulerConfig(),
		scheduler.WithSwitchCooldown(0))

	// Register paths representing different transport options
	sched.RegisterPath(&scheduler.Path{
		ID:    "udp-p2p",
		Type:  scheduler.PathUDPP2P,
		State: scheduler.PathAvailable,
	})
	sched.RegisterPath(&scheduler.Path{
		ID:    "quic-p2p",
		Type:  scheduler.PathQUICP2P,
		State: scheduler.PathAvailable,
	})
	sched.RegisterPath(&scheduler.Path{
		ID:    "tcp-relay",
		Type:  scheduler.PathTCPP2P,
		State: scheduler.PathAvailable,
	})
	sched.RegisterPath(&scheduler.Path{
		ID:    "global-relay",
		Type:  scheduler.PathGlobalRelay,
		State: scheduler.PathAvailable,
	})

	// Verify scheduler has the paths
	allPaths := sched.AllPaths()
	if len(allPaths) != 4 {
		t.Fatalf("expected 4 paths, got %d", len(allPaths))
	}

	// Set scheduler on engine
	engine.SetScheduler(sched)
	if engine.GetScheduler() == nil {
		t.Fatal("scheduler not set on engine")
	}

	// 2. Trigger path switching via the PathManager interface
	pm := engine.GetScheduler()

	// Switch to QUIC P2P
	if err := pm.SwitchTo("quic-p2p"); err != nil {
		t.Fatalf("SwitchTo quic-p2p: %v", err)
	}
	active := sched.ActivePath()
	if active == nil || active.ID != "quic-p2p" {
		t.Errorf("active path = %v, want quic-p2p", active)
	}

	// Switch to TCP relay (simulating degradation)
	if err := pm.SwitchTo("tcp-relay"); err != nil {
		t.Fatalf("SwitchTo tcp-relay: %v", err)
	}
	active = sched.ActivePath()
	if active == nil || active.ID != "tcp-relay" {
		t.Errorf("active path = %v, want tcp-relay", active)
	}

	// Switch to global relay (worst case fallback)
	if err := pm.SwitchTo("global-relay"); err != nil {
		t.Fatalf("SwitchTo global-relay: %v", err)
	}
	active = sched.ActivePath()
	if active == nil || active.ID != "global-relay" {
		t.Errorf("active path = %v, want global-relay", active)
	}

	t.Logf("PathManager: switched through quic-p2p → tcp-relay → global-relay successfully")

	// 3. Test RelaySelector interface
	relay1 := startTestRelayServer(t)
	defer relay1.Close()
	relay2 := startTestRelayServer(t)
	defer relay2.Close()

	rMgr := relay.NewRelayManager(relay.RelayManagerConfig{
		Relays: []relay.RelayClientConfig{
			{ServerAddr: relay1.Addr().String(), ClientID: "test-client"},
			{ServerAddr: relay2.Addr().String(), ClientID: "test-client"},
		},
		ProbeInterval: 100 * time.Millisecond,
	})

	ctx := context.Background()
	if err := rMgr.Start(ctx); err != nil {
		t.Fatalf("relay manager start: %v", err)
	}
	defer rMgr.Stop()

	// Wait for relay connections to establish
	time.Sleep(500 * time.Millisecond)

	engine.SetRelayManager(rMgr)
	rs := engine.GetRelayManager()
	if rs == nil {
		t.Fatal("relay manager not set on engine")
	}

	// Switch relay via interface
	if err := rs.SwitchTo(relay1.Addr().String()); err != nil {
		t.Fatalf("SwitchTo relay1: %v", err)
	}
	t.Logf("RelaySelector: switched to relay1 at %s", relay1.Addr().String())

	if err := rs.SwitchTo(relay2.Addr().String()); err != nil {
		t.Fatalf("SwitchTo relay2: %v", err)
	}
	t.Logf("RelaySelector: switched to relay2 at %s", relay2.Addr().String())

	// 4. Test MigrationController interface
	det := migration.NewPollingDetector(200 * time.Millisecond)
	migrator := migration.NewMigrator(migration.DefaultMigrationConfig(), det)

	engine.SetMigrator(migrator)
	mc := engine.GetMigrator()
	if mc == nil {
		t.Fatal("migrator not set on engine")
	}

	// Start/Stop via interface
	migratorCtx, migratorCancel := context.WithCancel(ctx)
	defer migratorCancel()

	if err := mc.Start(migratorCtx); err != nil {
		t.Fatalf("Start migrator: %v", err)
	}

	time.Sleep(300 * time.Millisecond) // Let it run briefly
	mc.Stop()

	t.Logf("MigrationController: start/stop lifecycle completed")

	// 5. Verify all three interfaces are accessible from the engine
	if engine.GetScheduler() == nil {
		t.Error("GetScheduler returned nil")
	}
	if engine.GetRelayManager() == nil {
		t.Error("GetRelayManager returned nil")
	}
	if engine.GetMigrator() == nil {
		t.Error("GetMigrator returned nil")
	}

	t.Log("Scheduler data plane loop integration PASSED: " +
		"PathManager drives path switching, " +
		"RelaySelector drives relay failover, " +
		"MigrationController manages network handoff lifecycle")
}

// --- Test helpers ---

func startTestRelayServer(t *testing.T) *testTCPServer {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := &testTCPServer{ln: ln, done: make(chan struct{})}
	go srv.acceptLoop()
	return srv
}

type testTCPServer struct {
	ln   net.Listener
	done chan struct{}
}

func (s *testTCPServer) Addr() net.Addr { return s.ln.Addr() }

func (s *testTCPServer) Close() error {
	close(s.done)
	return s.ln.Close()
}

func (s *testTCPServer) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go s.handleConn(conn)
	}
}

func (s *testTCPServer) handleConn(conn net.Conn) {
	defer conn.Close()
	pconn := protocol.NewConn(conn)
	msg, err := pconn.Read()
	if err != nil {
		return
	}
	if msg.Type == protocol.TypeAuth {
		resp, _ := protocol.NewAuthRespMessage(true, "")
		pconn.Write(resp)
	}
	// Keep connection alive until server closes
	<-s.done
}
