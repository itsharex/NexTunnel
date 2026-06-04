package relay

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/nextunnel/pkg/protocol"
)

// startMiniRelay starts a minimal relay server for testing.
func startMiniRelay(t *testing.T) (string, func()) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}
			go handleTestConn(conn)
		}
	}()

	cleanup := func() {
		cancel()
		ln.Close()
	}

	return ln.Addr().String(), cleanup
}

func handleTestConn(conn net.Conn) {
	defer conn.Close()
	pconn := protocol.NewConn(conn)

	msg, err := pconn.Read()
	if err != nil {
		return
	}
	if msg.Type != protocol.TypeAuth {
		return
	}

	resp, _ := protocol.NewAuthRespMessage(true, "")
	pconn.Write(resp)
}

func TestRelayClient_ConnectDisconnect(t *testing.T) {
	addr, cleanup := startMiniRelay(t)
	defer cleanup()

	client := NewRelayClient(RelayClientConfig{
		ServerAddr: addr,
		ClientID:   "test-client",
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}

	if !client.IsConnected() {
		t.Error("should be connected")
	}

	latency := client.Latency()
	if latency <= 0 {
		t.Error("latency should be > 0")
	}
	t.Logf("Relay connected: addr=%s latency=%v", client.ServerAddr(), latency)

	if err := client.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}

	if client.IsConnected() {
		t.Error("should not be connected after close")
	}
}

func TestRelayManager_ProbeAll(t *testing.T) {
	addr1, cleanup1 := startMiniRelay(t)
	defer cleanup1()
	addr2, cleanup2 := startMiniRelay(t)
	defer cleanup2()

	cfg := DefaultRelayManagerConfig()
	cfg.Relays = []RelayClientConfig{
		{ServerAddr: addr1, ClientID: "test-1", Region: "us-east"},
		{ServerAddr: addr2, ClientID: "test-2", Region: "eu-west"},
	}
	cfg.ProbeInterval = 100 * time.Millisecond

	mgr := NewRelayManager(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := mgr.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Wait for connections and initial selection
	time.Sleep(1 * time.Second)

	relays := mgr.AllRelays()
	t.Logf("Total relays: %d", len(relays))

	connected := 0
	for _, r := range relays {
		if r.IsConnected() {
			connected++
			t.Logf("Relay %s: connected, latency=%v, region=%s", r.ServerAddr(), r.Latency(), r.Region())
		}
	}

	if connected == 0 {
		t.Error("expected at least one connected relay")
	}

	active := mgr.ActiveRelay()
	if active != nil {
		t.Logf("Active relay: %s (region=%s)", active.ServerAddr(), active.Region())
	}

	cancel()
	mgr.Stop()
}

func TestRelayManager_Failover(t *testing.T) {
	addr1, cleanup1 := startMiniRelay(t)
	defer cleanup1()
	addr2, cleanup2 := startMiniRelay(t)
	defer cleanup2()

	cfg := DefaultRelayManagerConfig()
	cfg.Relays = []RelayClientConfig{
		{ServerAddr: addr1, ClientID: "failover-1"},
		{ServerAddr: addr2, ClientID: "failover-2"},
	}
	cfg.ProbeInterval = 100 * time.Millisecond
	cfg.FailoverTime = 500 * time.Millisecond

	mgr := NewRelayManager(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mgr.Start(ctx)
	time.Sleep(1 * time.Second)

	active := mgr.ActiveRelay()
	if active == nil {
		t.Fatal("no active relay")
	}
	t.Logf("Initial active: %s", active.ServerAddr())

	// Switch explicitly
	allRelays := mgr.AllRelays()
	for _, r := range allRelays {
		if r.ServerAddr() != active.ServerAddr() && r.IsConnected() {
			err := mgr.SwitchTo(r.ServerAddr())
			if err != nil {
				t.Logf("SwitchTo %s: %v", r.ServerAddr(), err)
			} else {
				t.Logf("Switched to: %s", r.ServerAddr())
			}
			break
		}
	}

	newActive := mgr.ActiveRelay()
	if newActive == nil {
		t.Error("no active relay after switch")
	}

	cancel()
	mgr.Stop()
}

func TestRelayManager_GeoSelect(t *testing.T) {
	addr, cleanup := startMiniRelay(t)
	defer cleanup()

	cfg := DefaultRelayManagerConfig()
	cfg.Relays = []RelayClientConfig{
		{ServerAddr: addr, ClientID: "geo-1", Region: "us-east"},
	}
	cfg.GeoSelect = true
	cfg.ProbeInterval = 100 * time.Millisecond

	mgr := NewRelayManager(cfg)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mgr.Start(ctx)
	time.Sleep(1 * time.Second)

	relays := mgr.AllRelays()
	if len(relays) != 1 {
		t.Fatalf("expected 1 relay, got %d", len(relays))
	}

	if relays[0].Region() != "us-east" {
		t.Errorf("region = %s, want us-east", relays[0].Region())
	}

	cancel()
	mgr.Stop()
}

// Ensure types are used
var _ = fmt.Sprintf
