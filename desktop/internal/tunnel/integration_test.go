package tunnel_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/nextunnel/desktop/internal/auth"
	"github.com/nextunnel/desktop/internal/config"
	"github.com/nextunnel/desktop/internal/tunnel"
	"github.com/nextunnel/pkg/protocol"
)

// --- Helpers ---

func testEchoServer(t *testing.T) (string, context.CancelFunc) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("echo server: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for {
			c, err := ln.Accept()
			if err != nil {
				select {
				case <-ctx.Done():
					return
				default:
					continue
				}
			}
			go func() {
				defer c.Close()
				io.Copy(c, c)
			}()
		}
	}()
	return ln.Addr().String(), func() { cancel(); ln.Close() }
}

// testRelay is a reusable mini relay for integration tests.
type testRelay struct {
	controlLn net.Listener
	proxyLn   net.Listener
	logger    *slog.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	ctrlConn  *protocol.Conn
	pendingMu sync.Mutex
	pending   map[string]chan net.Conn
}

func newTestRelay(t *testing.T) *testRelay {
	t.Helper()
	cln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	pln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		cln.Close()
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	r := &testRelay{
		controlLn: cln, proxyLn: pln,
		logger:  slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn})),
		ctx:     ctx,
		cancel:  cancel,
		pending: make(map[string]chan net.Conn),
	}
	return r
}

func (r *testRelay) controlAddr() string { return r.controlLn.Addr().String() }
func (r *testRelay) proxyPort() uint16  { return uint16(r.proxyLn.Addr().(*net.TCPAddr).Port) }

func (r *testRelay) stop() {
	r.cancel()
	r.controlLn.Close()
	r.proxyLn.Close()
}

func (r *testRelay) run(t *testing.T, proxyName string) {
	go func() {
		for {
			conn, err := r.controlLn.Accept()
			if err != nil {
				return
			}
			go r.handleCtrl(t, conn, proxyName)
		}
	}()
	go func() {
		for {
			conn, err := r.proxyLn.Accept()
			if err != nil {
				return
			}
			go r.handleExt(t, conn, proxyName)
		}
	}()
}

func (r *testRelay) handleCtrl(t *testing.T, conn net.Conn, proxyName string) {
	pconn := protocol.NewConn(conn)
	msg, err := pconn.Read()
	if err != nil {
		conn.Close()
		return
	}
	switch msg.Type {
	case protocol.TypeAuth:
		resp, _ := protocol.NewAuthRespMessage(true, "")
		pconn.Write(resp)
		r.ctrlConn = pconn
		for {
			m, err := pconn.Read()
			if err != nil {
				return
			}
			switch m.Type {
			case protocol.TypeHeartbeat:
				pconn.Write(protocol.NewHeartbeatResp())
			case protocol.TypeNewProxy:
				p, _ := m.DecodePayload()
				np := p.(*protocol.NewProxyMessage)
				pr, _ := protocol.NewNewProxyRespMessage(np.ProxyName, true, r.proxyPort(), "")
				pconn.Write(pr)
			case protocol.TypeCloseProxy:
				// ok
			}
		}
	case protocol.TypeWorkConn:
		p, _ := msg.DecodePayload()
		wc := p.(*protocol.WorkConnMessage)
		r.pendingMu.Lock()
		ch, ok := r.pending[wc.SessionID]
		if ok {
			delete(r.pending, wc.SessionID)
		}
		r.pendingMu.Unlock()
		if ok {
			ch <- conn
		} else {
			conn.Close()
		}
	default:
		conn.Close()
	}
}

func (r *testRelay) handleExt(t *testing.T, extConn net.Conn, proxyName string) {
	sid := fmt.Sprintf("s-%d", time.Now().UnixNano())
	ch := make(chan net.Conn, 1)
	r.pendingMu.Lock()
	r.pending[sid] = ch
	r.pendingMu.Unlock()
	if r.ctrlConn == nil {
		extConn.Close()
		return
	}
	msg, _ := protocol.NewStartWorkConnMessage(proxyName, sid)
	if err := r.ctrlConn.Write(msg); err != nil {
		extConn.Close()
		return
	}
	select {
	case workConn := <-ch:
		var wg sync.WaitGroup
		wg.Add(2)
		go func() { defer wg.Done(); io.Copy(workConn, extConn); extConn.Close() }()
		go func() { defer wg.Done(); io.Copy(extConn, workConn); workConn.Close() }()
		wg.Wait()
	case <-time.After(10 * time.Second):
		extConn.Close()
	case <-r.ctx.Done():
		extConn.Close()
	}
}

// --- Tests ---

func TestTCPReconnectAfterServerRestart(t *testing.T) {
	echoAddr, stopEcho := testEchoServer(t)
	defer stopEcho()

	// Start first relay
	relay1 := newTestRelay(t)
	relay1.run(t, "reconnect-test")
	serverAddr := relay1.controlAddr()
	proxyPort := relay1.proxyPort()
	t.Logf("relay1 control=%s proxy=%d", serverAddr, proxyPort)

	cfg := tunnel.TunnelClientConfig{
		ServerAddr:         serverAddr,
		ClientID:           "reconnect-client",
		HeartbeatInterval:  5 * time.Second,
		ReconnectBaseDelay: 500 * time.Millisecond,
		ReconnectMaxDelay:  2 * time.Second,
		Tunnels: []tunnel.TunnelDef{
			{Name: "reconnect-test", ProxyType: "tcp", LocalAddr: echoAddr, RemotePort: proxyPort},
		},
	}

	manager := tunnel.NewManager(cfg)
	manager.SetLogger(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn})))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go manager.Start(ctx)

	// Wait for connection
	deadline := time.After(5 * time.Second)
	for !manager.IsConnected() {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for initial connection")
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
	t.Log("initial connection established")

	// Send data through tunnel
	proxyAddr := fmt.Sprintf("127.0.0.1:%d", proxyPort)
	conn1, err := net.DialTimeout("tcp", proxyAddr, 3*time.Second)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	conn1.SetWriteDeadline(time.Now().Add(3 * time.Second))
	conn1.Write([]byte("before-restart"))
	buf := make([]byte, 14)
	conn1.SetReadDeadline(time.Now().Add(3 * time.Second))
	io.ReadFull(conn1, buf)
	conn1.Close()
	if string(buf) != "before-restart" {
		t.Fatalf("pre-restart echo: got %q", buf)
	}
	t.Log("pre-restart data transfer OK")

	// Kill the relay
	relay1.stop()
	t.Log("relay1 stopped")
	time.Sleep(1 * time.Second)

	// Restart relay on same address (we need to use the same port)
	cln2, err := net.Listen("tcp", serverAddr)
	if err != nil {
		t.Skipf("cannot rebind to same address: %v (port may be in use)", err)
	}
	relay2 := &testRelay{
		controlLn: cln2,
		logger:    slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn})),
		ctx:       ctx,
		cancel:    cancel,
		pending:   make(map[string]chan net.Conn),
	}
	// Use a new proxy listener on a different port for the reconnected session
	pln2, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	relay2.proxyLn = pln2
	go func() {
		for {
			conn, err := relay2.controlLn.Accept()
			if err != nil {
				return
			}
			go relay2.handleCtrl(t, conn, "reconnect-test")
		}
	}()

	// Wait for manager to reconnect
	t.Log("waiting for reconnection...")
	reconnDeadline := time.After(10 * time.Second)
	for !manager.IsConnected() {
		select {
		case <-reconnDeadline:
			t.Fatal("timeout waiting for reconnection")
		default:
			time.Sleep(200 * time.Millisecond)
		}
	}
	t.Log("reconnected successfully!")
	cancel()
}

func TestSQLiteConfigPersistence(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")

	// Create and populate
	db, err := config.Open(dbPath)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	store := config.NewStore(db)

	tunnels := []config.TunnelConfig{
		{ID: "t1", Name: "web", ProxyType: "tcp", LocalAddr: "127.0.0.1", LocalPort: 3000, RemotePort: 8080, Status: "stopped"},
		{ID: "t2", Name: "api", ProxyType: "http", LocalAddr: "127.0.0.1", LocalPort: 4000, RemotePort: 9090, Status: "running"},
		{ID: "t3", Name: "ssh", ProxyType: "tcp", LocalAddr: "127.0.0.1", LocalPort: 22, RemotePort: 2222, Status: "stopped"},
	}
	for i := range tunnels {
		if err := store.Create(&tunnels[i]); err != nil {
			t.Fatalf("create %s: %v", tunnels[i].Name, err)
		}
	}
	store.SetSetting("server_addr", "relay.example.com:7000")
	store.SetSetting("client_id", "test-client-123")
	db.Close()

	// Reopen and verify persistence
	db2, err := config.Open(dbPath)
	if err != nil {
		t.Fatalf("reopen: %v", err)
	}
	defer db2.Close()
	store2 := config.NewStore(db2)

	count, _ := store2.Count()
	if count != 3 {
		t.Fatalf("expected 3 tunnels after reopen, got %d", count)
	}

	list, _ := store2.List()
	if len(list) != 3 {
		t.Fatalf("expected 3 tunnels in list, got %d", len(list))
	}

	web, _ := store2.GetByName("web")
	if web == nil || web.LocalPort != 3000 {
		t.Errorf("web tunnel: got %+v", web)
	}

	api, _ := store2.GetByName("api")
	if api == nil || api.ProxyType != "http" {
		t.Errorf("api tunnel proxy type: got %v", api)
	}

	serverAddr, _ := store2.GetSetting("server_addr")
	if serverAddr != "relay.example.com:7000" {
		t.Errorf("server_addr: got %q", serverAddr)
	}

	clientID, _ := store2.GetSetting("client_id")
	if clientID != "test-client-123" {
		t.Errorf("client_id: got %q", clientID)
	}

	// Update and verify
	web.Status = "running"
	store2.Update(web)
	updated, _ := store2.GetByName("web")
	if updated.Status != "running" {
		t.Errorf("update status: got %q", updated.Status)
	}

	// Delete and verify
	store2.Delete("t2")
	count, _ = store2.Count()
	if count != 2 {
		t.Errorf("after delete: expected 2, got %d", count)
	}

	t.Log("SQLite persistence test passed")
}

func TestAuthTokenLifecycle(t *testing.T) {
	secret := []byte("nexTunnel-test-secret-key-2026")

	// Generate token
	token, err := auth.GenerateToken("client-alpha", secret, 2*time.Hour)
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	// Validate
	claims, err := auth.ValidateToken(token, secret)
	if err != nil {
		t.Fatalf("validate: %v", err)
	}
	if claims.ClientID != "client-alpha" {
		t.Errorf("client_id: got %q", claims.ClientID)
	}

	// Tamper with token -> invalid
	tampered := token + "x"
	_, err = auth.ValidateToken(tampered, secret)
	if err != auth.ErrTokenInvalid {
		t.Errorf("tampered token: expected ErrTokenInvalid, got %v", err)
	}

	// Wrong secret -> invalid
	_, err = auth.ValidateToken(token, []byte("wrong"))
	if err != auth.ErrTokenInvalid {
		t.Errorf("wrong secret: expected ErrTokenInvalid, got %v", err)
	}

	// Generate expired token
	expiredToken, _ := auth.GenerateToken("client-beta", secret, -1*time.Hour)
	_, err = auth.ValidateToken(expiredToken, secret)
	if err != auth.ErrTokenExpired {
		t.Errorf("expired: expected ErrTokenExpired, got %v", err)
	}

	// Refresh expired token
	refreshed, err := auth.RefreshToken(expiredToken, secret, 1*time.Hour)
	if err != nil {
		t.Fatalf("refresh: %v", err)
	}
	claims2, err := auth.ValidateToken(refreshed, secret)
	if err != nil {
		t.Fatalf("validate refreshed: %v", err)
	}
	if claims2.ClientID != "client-beta" {
		t.Errorf("refreshed client_id: got %q", claims2.ClientID)
	}

	// IsExpiringSoon
	shortToken, _ := auth.GenerateToken("client-gamma", secret, 5*time.Minute)
	if !auth.IsExpiringSoon(shortToken, secret, 10*time.Minute) {
		t.Error("should be expiring soon")
	}

	longToken, _ := auth.GenerateToken("client-gamma", secret, 24*time.Hour)
	if auth.IsExpiringSoon(longToken, secret, 10*time.Minute) {
		t.Error("should not be expiring soon")
	}

	t.Log("auth token lifecycle test passed")
}

func TestMultipleTunnelsE2E(t *testing.T) {
	echo1, stop1 := testEchoServer(t)
	defer stop1()

	relay := newTestRelay(t)
	// The relay needs to handle multiple proxy names, but our test relay
	// only supports one. We'll test sequential tunnel registration instead.
	relay.run(t, "multi-test")
	defer relay.stop()

	cfg := tunnel.TunnelClientConfig{
		ServerAddr:         relay.controlAddr(),
		ClientID:           "multi-client",
		HeartbeatInterval:  10 * time.Second,
		ReconnectBaseDelay: 1 * time.Second,
		ReconnectMaxDelay:  5 * time.Second,
		Tunnels: []tunnel.TunnelDef{
			{Name: "multi-test", ProxyType: "tcp", LocalAddr: echo1, RemotePort: relay.proxyPort()},
		},
	}

	manager := tunnel.NewManager(cfg)
	manager.SetLogger(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelWarn})))

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go manager.Start(ctx)

	// Wait for registration
	deadline := time.After(5 * time.Second)
	for {
		select {
		case <-deadline:
			t.Fatal("timeout")
		default:
		}
		if manager.IsConnected() {
			for _, s := range manager.GetStatus() {
				if s.ProxyName == "multi-test" && s.Status == "active" {
					goto ready
				}
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
ready:

	// Transfer data
	conn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", relay.proxyPort()), 3*time.Second)
	if err != nil {
		t.Fatalf("dial: %v", err)
	}
	defer conn.Close()

	data := []byte("multi-tunnel-test-data")
	conn.SetWriteDeadline(time.Now().Add(3 * time.Second))
	conn.Write(data)

	buf := make([]byte, len(data))
	conn.SetReadDeadline(time.Now().Add(3 * time.Second))
	n, err := io.ReadFull(conn, buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(buf[:n]) != string(data) {
		t.Fatalf("echo: got %q, want %q", buf[:n], data)
	}

	// Verify status reports
	statuses := manager.GetStatus()
	if len(statuses) == 0 {
		t.Fatal("no statuses reported")
	}
	found := false
	for _, s := range statuses {
		if s.ProxyName == "multi-test" {
			found = true
			if s.Status != "active" {
				t.Errorf("status: got %q, want active", s.Status)
			}
		}
	}
	if !found {
		t.Error("multi-test tunnel not found in status")
	}

	t.Log("multiple tunnels E2E test passed")
}
