package tunnel_test

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"log/slog"
	"math/big"
	"net"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/nextunnel/desktop/internal/tunnel"
	"github.com/nextunnel/pkg/protocol"
	"github.com/quic-go/quic-go"
)

// TestQUICRelayE2E tests end-to-end TCP tunnel forwarding through a QUIC relay.
// Flow: ExternalUser --TCP--> MiniRelay --QUIC stream--> Client --TCP--> EchoServer
func TestQUICRelayE2E(t *testing.T) {
	// 1. Start echo server (local service)
	echoAddr, stopEcho := testEchoServer(t)
	defer stopEcho()

	// 2. Start QUIC-aware mini relay
	relay := newQUICRelay(t, "quic-e2e-test")
	relay.run(t)
	defer relay.stop()

	t.Logf("QUIC relay: control=%s quic=%s proxy=%d",
		relay.controlAddr(), relay.quicAddr(), relay.proxyPort())

	// 3. Create QUIC work conn opener on client side
	quicOpener := tunnel.NewQUICWorkConnOpener(relay.quicAddr(), &tls.Config{
		InsecureSkipVerify: true,
		NextProtos:        []string{"nextunnel-quic-relay"},
	})
	if err := quicOpener.Connect(context.Background()); err != nil {
		t.Fatalf("QUIC connect: %v", err)
	}
	defer quicOpener.Close()

	// 4. Create tunnel manager with QUIC opener
	cfg := tunnel.TunnelClientConfig{
		ServerAddr:         relay.controlAddr(),
		ClientID:           "quic-e2e-client",
		HeartbeatInterval:  10 * time.Second,
		ReconnectBaseDelay: 1 * time.Second,
		ReconnectMaxDelay:  5 * time.Second,
		Tunnels: []tunnel.TunnelDef{
			{Name: "quic-e2e-test", ProxyType: "tcp", LocalAddr: echoAddr, RemotePort: relay.proxyPort()},
		},
	}

	mgr := tunnel.NewManager(cfg)
	mgr.SetWorkConnOpener(quicOpener)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Start manager in background
	go func() {
		if err := mgr.Start(ctx); err != nil {
			t.Logf("manager stopped: %v", err)
		}
	}()

	// Wait for tunnel to register
	time.Sleep(2 * time.Second)

	// 5. External user connects to the proxy port
	extConn, err := net.DialTimeout("tcp", fmt.Sprintf("127.0.0.1:%d", relay.proxyPort()), 5*time.Second)
	if err != nil {
		t.Fatalf("external dial: %v", err)
	}
	defer extConn.Close()

	// 6. Send data and verify echo
	testData := "Hello QUIC Relay E2E!"
	extConn.SetDeadline(time.Now().Add(5 * time.Second))

	if _, err := extConn.Write([]byte(testData)); err != nil {
		t.Fatalf("external write: %v", err)
	}

	buf := make([]byte, len(testData))
	if _, err := io.ReadFull(extConn, buf); err != nil {
		t.Fatalf("external read: %v", err)
	}

	if string(buf) != testData {
		t.Errorf("echo mismatch: got %q, want %q", string(buf), testData)
	}

	t.Logf("QUIC Relay E2E PASSED: sent %q, received %q via QUIC work stream", testData, string(buf))
	cancel()
}

// --- QUIC-aware mini relay for testing ---

type quicRelay struct {
	controlLn net.Listener
	proxyLn   net.Listener
	quicLn    *quic.Listener
	logger    *slog.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	ctrlConn  *protocol.Conn
	pendingMu sync.Mutex
	pending   map[string]chan net.Conn
}

func newQUICRelay(t *testing.T, proxyName string) *quicRelay {
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
	return &quicRelay{
		controlLn: cln, proxyLn: pln,
		logger:  slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug})),
		ctx:     ctx,
		cancel:  cancel,
		pending: make(map[string]chan net.Conn),
	}
}

func (r *quicRelay) controlAddr() string { return r.controlLn.Addr().String() }
func (r *quicRelay) proxyPort() uint16  { return uint16(r.proxyLn.Addr().(*net.TCPAddr).Port) }

func (r *quicRelay) quicAddr() string {
	if r.quicLn != nil {
		return r.quicLn.Addr().String()
	}
	return ""
}

func (r *quicRelay) stop() {
	r.cancel()
	r.controlLn.Close()
	r.proxyLn.Close()
	if r.quicLn != nil {
		r.quicLn.Close()
	}
}

func (r *quicRelay) run(t *testing.T) {
	// Start QUIC listener
	tlsCfg := generateTestTLSConfig()
	quicCfg := &quic.Config{
		MaxIncomingStreams:    100,
		MaxIncomingUniStreams: 10,
	}
	qln, err := quic.ListenAddr("127.0.0.1:0", tlsCfg, quicCfg)
	if err != nil {
		t.Fatalf("quic listen: %v", err)
	}
	r.quicLn = qln

	// Accept QUIC streams as work connections
	go func() {
		for {
			conn, err := qln.Accept(r.ctx)
			if err != nil {
				return
			}
			go r.handleQUICConn(t, conn)
		}
	}()

	// Accept TCP control connections
	go func() {
		for {
			conn, err := r.controlLn.Accept()
			if err != nil {
				return
			}
			go r.handleCtrl(t, conn)
		}
	}()

	// Accept external TCP connections on proxy port
	go func() {
		for {
			conn, err := r.proxyLn.Accept()
			if err != nil {
				return
			}
			go r.handleExt(t, conn)
		}
	}()
}

func (r *quicRelay) handleQUICConn(t *testing.T, conn *quic.Conn) {
	for {
		stream, err := conn.AcceptStream(r.ctx)
		if err != nil {
			return
		}
		go r.handleQUICStream(t, stream, conn.LocalAddr(), conn.RemoteAddr())
	}
}

func (r *quicRelay) handleQUICStream(t *testing.T, stream *quic.Stream, localAddr, remoteAddr net.Addr) {
	// Wrap stream as net.Conn for protocol framing
	streamConn := &testQUICStreamConn{stream: stream, localAddr: localAddr, remoteAddr: remoteAddr}
	pconn := protocol.NewConn(streamConn)

	msg, err := pconn.Read()
	if err != nil {
		t.Logf("QUIC stream read error: %v", err)
		stream.Close()
		return
	}

	if msg.Type != protocol.TypeWorkConn {
		t.Logf("unexpected QUIC first message type: %d", msg.Type)
		stream.Close()
		return
	}

	p, _ := msg.DecodePayload()
	wc := p.(*protocol.WorkConnMessage)

	r.pendingMu.Lock()
	ch, ok := r.pending[wc.SessionID]
	if ok {
		delete(r.pending, wc.SessionID)
	}
	r.pendingMu.Unlock()

	if ok {
		// Pass the raw stream (not protocol.Conn) for data forwarding
		ch <- streamConn
	} else {
		stream.Close()
	}
}

func (r *quicRelay) handleCtrl(t *testing.T, conn net.Conn) {
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
	default:
		conn.Close()
	}
}

func (r *quicRelay) handleExt(t *testing.T, extConn net.Conn) {
	sid := fmt.Sprintf("s-%d", time.Now().UnixNano())
	ch := make(chan net.Conn, 1)
	r.pendingMu.Lock()
	r.pending[sid] = ch
	r.pendingMu.Unlock()

	if r.ctrlConn == nil {
		extConn.Close()
		return
	}
	msg, _ := protocol.NewStartWorkConnMessage("quic-e2e-test", sid)
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

func generateTestTLSConfig() *tls.Config {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		panic(err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
		IPAddresses:  []net.IP{net.ParseIP("127.0.0.1")},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		panic(err)
	}
	return &tls.Config{
		Certificates: []tls.Certificate{
			{Certificate: [][]byte{certDER}, PrivateKey: key},
		},
		MinVersion: tls.VersionTLS13,
		NextProtos: []string{"nextunnel-quic-relay"},
	}
}

// testQUICStreamConn wraps a QUIC stream as net.Conn for testing.
type testQUICStreamConn struct {
	stream     *quic.Stream
	localAddr  net.Addr
	remoteAddr net.Addr
}

func (c *testQUICStreamConn) Read(p []byte) (int, error)  { return c.stream.Read(p) }
func (c *testQUICStreamConn) Write(p []byte) (int, error) { return c.stream.Write(p) }
func (c *testQUICStreamConn) Close() error {
	c.stream.CancelRead(0)
	return c.stream.Close()
}
func (c *testQUICStreamConn) LocalAddr() net.Addr                { return c.localAddr }
func (c *testQUICStreamConn) RemoteAddr() net.Addr               { return c.remoteAddr }
func (c *testQUICStreamConn) SetDeadline(t time.Time) error {
	if err := c.stream.SetReadDeadline(t); err != nil {
		return err
	}
	return c.stream.SetWriteDeadline(t)
}
func (c *testQUICStreamConn) SetReadDeadline(t time.Time) error {
	return c.stream.SetReadDeadline(t)
}
func (c *testQUICStreamConn) SetWriteDeadline(t time.Time) error {
	return c.stream.SetWriteDeadline(t)
}

