package tunnel_test

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/nextunnel/desktop/internal/tunnel"
	"github.com/nextunnel/pkg/protocol"
)

func startEchoServer(t *testing.T) (string, context.CancelFunc) {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("start echo server: %v", err)
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
			go func(c net.Conn) {
				defer c.Close()
				io.Copy(c, c)
			}(conn)
		}
	}()
	return ln.Addr().String(), func() {
		cancel()
		ln.Close()
	}
}

// miniRelay implements a minimal relay server for integration testing.
type miniRelay struct {
	controlLn net.Listener
	proxyLn   net.Listener
	logger    *slog.Logger
	ctx       context.Context
	cancel    context.CancelFunc

	ctrlConn *protocol.Conn

	pendingMu sync.Mutex
	pending   map[string]chan net.Conn
}

func newMiniRelay(t *testing.T) *miniRelay {
	t.Helper()
	controlLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen control: %v", err)
	}
	proxyLn, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		controlLn.Close()
		t.Fatalf("listen proxy: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &miniRelay{
		controlLn: controlLn,
		proxyLn:   proxyLn,
		logger:    slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
		ctx:       ctx,
		cancel:    cancel,
		pending:   make(map[string]chan net.Conn),
	}
}

func (r *miniRelay) controlAddr() string { return r.controlLn.Addr().String() }
func (r *miniRelay) proxyPort() uint16  { return uint16(r.proxyLn.Addr().(*net.TCPAddr).Port) }

func (r *miniRelay) stop() {
	r.cancel()
	r.controlLn.Close()
	r.proxyLn.Close()
}

func (r *miniRelay) run(t *testing.T) {
	go func() {
		for {
			conn, err := r.controlLn.Accept()
			if err != nil {
				return
			}
			go r.handleControlPort(t, conn)
		}
	}()
	go func() {
		for {
			conn, err := r.proxyLn.Accept()
			if err != nil {
				return
			}
			go r.handleExternalConn(t, conn)
		}
	}()
}

func (r *miniRelay) handleControlPort(t *testing.T, conn net.Conn) {
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
		r.logger.Info("mini relay: client connected")
		for {
			m, err := pconn.Read()
			if err != nil {
				return
			}
			switch m.Type {
			case protocol.TypeHeartbeat:
				pconn.Write(protocol.NewHeartbeatResp())
			case protocol.TypeNewProxy:
				payload, _ := m.DecodePayload()
				np := payload.(*protocol.NewProxyMessage)
				proxyResp, _ := protocol.NewNewProxyRespMessage(np.ProxyName, true, r.proxyPort(), "")
				pconn.Write(proxyResp)
			case protocol.TypeCloseProxy:
				// ok
			}
		}

	case protocol.TypeWorkConn:
		payload, _ := msg.DecodePayload()
		wc := payload.(*protocol.WorkConnMessage)
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

func (r *miniRelay) handleExternalConn(t *testing.T, extConn net.Conn) {
	sessionID := fmt.Sprintf("sess-%d", time.Now().UnixNano())
	ch := make(chan net.Conn, 1)
	r.pendingMu.Lock()
	r.pending[sessionID] = ch
	r.pendingMu.Unlock()

	if r.ctrlConn == nil {
		extConn.Close()
		return
	}
	msg, err := protocol.NewStartWorkConnMessage("echo-test", sessionID)
	if err != nil {
		extConn.Close()
		return
	}
	if err := r.ctrlConn.Write(msg); err != nil {
		extConn.Close()
		return
	}

	select {
	case workConn := <-ch:
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			io.Copy(workConn, extConn)
			extConn.Close()
		}()
		go func() {
			defer wg.Done()
			io.Copy(extConn, workConn)
			workConn.Close()
		}()
		wg.Wait()
	case <-time.After(10 * time.Second):
		extConn.Close()
	case <-r.ctx.Done():
		extConn.Close()
	}
}

func TestTCPTunnelEndToEnd(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	echoAddr, stopEcho := startEchoServer(t)
	defer stopEcho()
	t.Logf("echo server at %s", echoAddr)

	rl := newMiniRelay(t)
	defer rl.stop()
	rl.run(t)
	t.Logf("relay control=%s proxy_port=%d", rl.controlAddr(), rl.proxyPort())

	clientCfg := tunnel.TunnelClientConfig{
		ServerAddr:         rl.controlAddr(),
		ClientID:           "test-client",
		HeartbeatInterval:  10 * time.Second,
		ReconnectBaseDelay: 1 * time.Second,
		ReconnectMaxDelay:  5 * time.Second,
		Tunnels: []tunnel.TunnelDef{
			{
				Name:       "echo-test",
				ProxyType:  "tcp",
				LocalAddr:  echoAddr,
				RemotePort: rl.proxyPort(),
			},
		},
	}

	manager := tunnel.NewManager(clientCfg)
	manager.SetLogger(logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	errCh := make(chan error, 1)
	go func() {
		errCh <- manager.Start(ctx)
	}()

	// Wait for registration
	deadline := time.After(5 * time.Second)
registered := false
	for !registered {
		select {
		case <-deadline:
			t.Fatal("timeout waiting for tunnel registration")
		case err := <-errCh:
			t.Fatalf("manager.Start returned: %v", err)
		default:
		}
		if manager.IsConnected() {
			for _, s := range manager.GetStatus() {
				if s.ProxyName == "echo-test" && s.Status == "active" {
					registered = true
				}
			}
		}
		if !registered {
			time.Sleep(100 * time.Millisecond)
		}
	}
	t.Log("tunnel registered")

	proxyAddr := fmt.Sprintf("127.0.0.1:%d", rl.proxyPort())
	extConn, err := net.DialTimeout("tcp", proxyAddr, 5*time.Second)
	if err != nil {
		t.Fatalf("dial proxy: %v", err)
	}
	defer extConn.Close()

	testData := []byte("Hello NexTunnel!")
	extConn.SetWriteDeadline(time.Now().Add(5 * time.Second))
	if _, err := extConn.Write(testData); err != nil {
		t.Fatalf("write to proxy: %v", err)
	}

	buf := make([]byte, len(testData))
	extConn.SetReadDeadline(time.Now().Add(5 * time.Second))
	n, err := io.ReadFull(extConn, buf)
	if err != nil {
		t.Fatalf("read from proxy: %v", err)
	}

	if string(buf[:n]) != string(testData) {
		t.Fatalf("echo mismatch: got %q, want %q", buf[:n], testData)
	}

	t.Log("end-to-end TCP tunnel test passed!")

	for _, s := range manager.GetStatus() {
		if s.ProxyName == "echo-test" {
			t.Logf("traffic: in=%d out=%d", s.BytesIn, s.BytesOut)
		}
	}
}
