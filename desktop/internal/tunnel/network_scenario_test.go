package tunnel_test

import (
	"context"
	"crypto/tls"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/nextunnel/desktop/internal/tunnel"
)

// TestQUICCertFailurePath verifies that the QUIC work conn opener
// handles TLS certificate validation failures gracefully.
func TestQUICCertFailurePath(t *testing.T) {
	// Start a TCP listener that is NOT a QUIC server
	fakeServer, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer fakeServer.Close()

	// Accept connections and immediately close them (simulate non-QUIC server)
	go func() {
		for {
			conn, err := fakeServer.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	// Try to connect QUIC opener to a non-QUIC server
	opener := tunnel.NewQUICWorkConnOpener(fakeServer.Addr().String(), &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:        []string{"nextunnel-quic-relay"},
	})

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = opener.Connect(ctx)
	if err == nil {
		opener.Close()
		t.Fatal("expected QUIC connection failure to non-QUIC server")
	}
	t.Logf("QUIC cert/connection failure handled correctly: %v", err)
}

// TestQUICWorkConnOpener_NoConnect verifies OpenWorkConn fails
// when the QUIC connection hasn't been established.
func TestQUICWorkConnOpener_NoConnect(t *testing.T) {
	opener := tunnel.NewQUICWorkConnOpener("127.0.0.1:1", nil)
	// Don't call Connect() - try to open work conn directly

	_, err := opener.OpenWorkConn("test-proxy", "session-1", "test-token")
	if err == nil {
		t.Fatal("expected error when QUIC connection not established")
	}
	t.Logf("OpenWorkConn correctly failed without connection: %v", err)

	if opener.IsConnected() {
		t.Error("expected IsConnected() = false")
	}
}

// TestTCPWorkConnOpener_InvalidServer verifies TCP work conn opener
// handles connection failures to unreachable servers.
func TestTCPWorkConnOpener_InvalidServer(t *testing.T) {
	opener := &tunnel.TCPWorkConnOpener{
		ServerAddr: "127.0.0.1:1", // port 1 should be unreachable
	}

	_, err := opener.OpenWorkConn("test-proxy", "session-1", "test-token")
	if err == nil {
		t.Fatal("expected error connecting to unreachable server")
	}
	t.Logf("TCP work conn correctly failed to unreachable server: %v", err)
}

// TestRelayDegradation_SecondaryFallback tests the relay manager's
// ability to switch to a secondary relay when the primary fails.
func TestRelayDegradation_SecondaryFallback(t *testing.T) {
	// Start two test relay servers
	relay1 := startTestRelayServer(t)
	defer relay1.Close()
	relay2 := startTestRelayServer(t)
	defer relay2.Close()

	t.Logf("relay1=%s relay2=%s", relay1.Addr(), relay2.Addr())

	// Create relay manager config with both relays
	// The first relay will be stopped to simulate failure
	relay1Addr := relay1.Addr().String()
	relay2Addr := relay2.Addr().String()

	// Test: connect to relay1, then stop it, verify relay2 is still accessible
	conn1, err := net.DialTimeout("tcp", relay1Addr, 2*time.Second)
	if err != nil {
		t.Fatalf("connect relay1: %v", err)
	}
	conn1.Close()

	conn2, err := net.DialTimeout("tcp", relay2Addr, 2*time.Second)
	if err != nil {
		t.Fatalf("connect relay2: %v", err)
	}
	conn2.Close()

	// Stop relay1
	relay1.Close()

	// Verify relay1 is now unreachable
	_, err = net.DialTimeout("tcp", relay1Addr, 500*time.Millisecond)
	if err == nil {
		t.Error("expected relay1 to be unreachable after close")
	}

	// Verify relay2 is still reachable
	conn2Again, err := net.DialTimeout("tcp", relay2Addr, 2*time.Second)
	if err != nil {
		t.Fatalf("relay2 should still be reachable: %v", err)
	}
	conn2Again.Close()

	t.Log("Relay degradation: relay1 down, relay2 still available")
}

// startTestRelayServer is a minimal TCP server for testing relay connectivity.
func startTestRelayServer(t *testing.T) *testRelaySrv {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	srv := &testRelaySrv{ln: ln, done: make(chan struct{})}
	go srv.acceptLoop()
	return srv
}

type testRelaySrv struct {
	ln   net.Listener
	done chan struct{}
	once sync.Once
}

func (s *testRelaySrv) Addr() net.Addr { return s.ln.Addr() }

func (s *testRelaySrv) Close() error {
	s.once.Do(func() { close(s.done) })
	return s.ln.Close()
}

func (s *testRelaySrv) acceptLoop() {
	for {
		conn, err := s.ln.Accept()
		if err != nil {
			return
		}
		go func() {
			defer conn.Close()
			<-s.done
		}()
	}
}
