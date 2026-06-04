package quic

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log/slog"
	"math/big"
	"net"
	"sync/atomic"
	"time"

	q "github.com/quic-go/quic-go"
)

// QUICTransport implements the Transport interface over QUIC.
// It provides bidirectional streaming, 0-RTT recovery, and connection migration.
type QUICTransport struct {
	config QUICConfig
	conn   *q.Conn
	stream *q.Stream
	state  atomic.Value // QUICTransportState

	ctx    context.Context
	cancel context.CancelFunc
	logger *slog.Logger
}

// NewQUICTransport creates a new QUICTransport with the given options.
func NewQUICTransport(opts ...Option) *QUICTransport {
	cfg := DefaultQUICConfig()
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	ctx, cancel := context.WithCancel(context.Background())
	t := &QUICTransport{
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
		logger: cfg.Logger,
	}
	t.state.Store(QUICStateIdle)
	return t
}

// Dial connects to a remote QUIC endpoint and opens the first bidirectional stream.
// A ready signal is sent on the stream so the peer's AcceptStream returns immediately.
func (t *QUICTransport) Dial(ctx context.Context, addr string) error {
	t.state.Store(QUICStateConnecting)

	tlsCfg := t.config.TLSConfig
	if tlsCfg == nil {
		t.state.Store(QUICStateError)
		return fmt.Errorf("quic dial %s: TLSConfig is required for certificate verification", addr)
	} else if len(tlsCfg.NextProtos) == 0 {
		tlsCfg.NextProtos = []string{t.config.ALPN}
	}

	quicCfg := &q.Config{
		MaxIncomingStreams:    t.config.MaxStreams,
		MaxIncomingUniStreams: t.config.MaxStreams,
		Allow0RTT:             t.config.Enable0RTT,
	}

	if t.config.HandshakeTimeout > 0 {
		var cancelCtx context.CancelFunc
		ctx, cancelCtx = context.WithTimeout(ctx, t.config.HandshakeTimeout)
		defer cancelCtx()
	}

	conn, err := q.DialAddr(ctx, addr, tlsCfg, quicCfg)
	if err != nil {
		t.state.Store(QUICStateError)
		return fmt.Errorf("quic dial %s: %w", addr, err)
	}
	t.conn = conn

	stream, err := conn.OpenStreamSync(ctx)
	if err != nil {
		conn.CloseWithError(1, "failed to open stream")
		t.state.Store(QUICStateError)
		return fmt.Errorf("open quic stream: %w", err)
	}

	// Write a ready signal so the peer's AcceptStream returns immediately
	if _, err := stream.Write([]byte{0x01}); err != nil {
		conn.CloseWithError(1, "failed to write ready signal")
		t.state.Store(QUICStateError)
		return fmt.Errorf("write ready signal: %w", err)
	}

	t.stream = stream
	t.state.Store(QUICStateConnected)
	t.logger.Info("QUIC transport connected", "remote", conn.RemoteAddr())
	return nil
}

// AcceptFromListener accepts a connection from a QUIC listener.
// It waits for the peer's ready signal before returning.
func (t *QUICTransport) AcceptFromListener(ctx context.Context, ln *Listener) error {
	t.state.Store(QUICStateConnecting)

	conn, err := ln.Accept(ctx)
	if err != nil {
		t.state.Store(QUICStateError)
		return fmt.Errorf("accept quic conn: %w", err)
	}
	t.conn = conn

	stream, err := conn.AcceptStream(ctx)
	if err != nil {
		conn.CloseWithError(1, "failed to accept stream")
		t.state.Store(QUICStateError)
		return fmt.Errorf("accept quic stream: %w", err)
	}

	// Read the ready signal from the peer
	readyBuf := make([]byte, 1)
	if _, err := stream.Read(readyBuf); err != nil {
		conn.CloseWithError(1, "failed to read ready signal")
		t.state.Store(QUICStateError)
		return fmt.Errorf("read ready signal: %w", err)
	}

	t.stream = stream
	t.state.Store(QUICStateConnected)
	t.logger.Info("QUIC transport accepted", "remote", conn.RemoteAddr())
	return nil
}

// Read reads data from the primary QUIC stream.
func (t *QUICTransport) Read(p []byte) (int, error) {
	if t.stream == nil {
		return 0, fmt.Errorf("quic transport not connected")
	}
	return t.stream.Read(p)
}

// Write writes data to the primary QUIC stream.
func (t *QUICTransport) Write(p []byte) (int, error) {
	if t.stream == nil {
		return 0, fmt.Errorf("quic transport not connected")
	}
	return t.stream.Write(p)
}

// Close shuts down the QUIC transport.
func (t *QUICTransport) Close() error {
	t.state.Store(QUICStateClosed)
	t.cancel()
	if t.stream != nil {
		t.stream.CancelRead(0)
		t.stream.Close()
	}
	if t.conn != nil {
		return t.conn.CloseWithError(0, "closing")
	}
	t.logger.Info("QUIC transport closed")
	return nil
}

// LocalAddr returns the local network address.
func (t *QUICTransport) LocalAddr() net.Addr {
	if t.conn != nil {
		return t.conn.LocalAddr()
	}
	return nil
}

// RemoteAddr returns the remote network address.
func (t *QUICTransport) RemoteAddr() net.Addr {
	if t.conn != nil {
		return t.conn.RemoteAddr()
	}
	return nil
}

// State returns the current transport state.
func (t *QUICTransport) State() QUICTransportState {
	return t.state.Load().(QUICTransportState)
}

// OpenStream opens a new bidirectional QUIC stream and sends a ready signal.
func (t *QUICTransport) OpenStream() (*StreamAdapter, error) {
	if t.conn == nil {
		return nil, fmt.Errorf("quic transport not connected")
	}
	stream, err := t.conn.OpenStreamSync(t.ctx)
	if err != nil {
		return nil, fmt.Errorf("open stream: %w", err)
	}
	// Write ready signal so peer's AcceptStream returns
	if _, err := stream.Write([]byte{0x01}); err != nil {
		stream.Close()
		return nil, fmt.Errorf("write ready signal: %w", err)
	}
	return NewStreamAdapter(stream, t.conn.LocalAddr(), t.conn.RemoteAddr()), nil
}

// AcceptStream accepts the next incoming bidirectional QUIC stream.
// It waits for the peer's ready signal before returning.
func (t *QUICTransport) AcceptStream(ctx context.Context) (*StreamAdapter, error) {
	if t.conn == nil {
		return nil, fmt.Errorf("quic transport not connected")
	}
	stream, err := t.conn.AcceptStream(ctx)
	if err != nil {
		return nil, fmt.Errorf("accept stream: %w", err)
	}
	// Read ready signal
	readyBuf := make([]byte, 1)
	if _, err := stream.Read(readyBuf); err != nil {
		stream.Close()
		return nil, fmt.Errorf("read ready signal: %w", err)
	}
	return NewStreamAdapter(stream, t.conn.LocalAddr(), t.conn.RemoteAddr()), nil
}

// ConnectionState returns the underlying QUIC connection state.
func (t *QUICTransport) ConnectionState() q.ConnectionState {
	if t.conn == nil {
		return q.ConnectionState{}
	}
	return t.conn.ConnectionState()
}

// SetKeepAlive starts a background goroutine that sends periodic keep-alive pings.
func (t *QUICTransport) SetKeepAlive() {
	if t.config.KeepAlive <= 0 || t.conn == nil {
		return
	}
	go func() {
		ticker := time.NewTicker(t.config.KeepAlive)
		defer ticker.Stop()
		for {
			select {
			case <-t.ctx.Done():
				return
			case <-ticker.C:
				_ = t.conn.SendDatagram([]byte("ping"))
			}
		}
	}()
}

// generateSelfSignedCert creates a self-signed TLS certificate for QUIC.
func generateSelfSignedCert() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate key: %w", err)
	}

	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     time.Now().Add(365 * 24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}

	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create certificate: %w", err)
	}

	return tls.Certificate{
		Certificate: [][]byte{certDER},
		PrivateKey:  key,
	}, nil
}
