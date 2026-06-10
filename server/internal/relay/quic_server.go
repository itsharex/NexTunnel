package relay

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
	"sync"
	"time"

	"github.com/nextunnel/pkg/protocol"
	"github.com/nextunnel/pkg/tlsutil"
	"github.com/quic-go/quic-go"
)

// QUICTransport extends the relay server to support QUIC connections
// alongside the existing TCP transport.
type QUICTransport struct {
	config   *Config
	listener *quic.Listener
	logger   *slog.Logger
	server   *Server

	connsMu sync.RWMutex
	conns   map[string]*quic.Conn
}

// NewQUICTransport creates a QUIC transport layer for the relay server.
func NewQUICTransport(cfg *Config, server *Server, logger *slog.Logger) *QUICTransport {
	return &QUICTransport{
		config: cfg,
		logger: logger,
		server: server,
		conns:  make(map[string]*quic.Conn),
	}
}

// Start begins listening for QUIC connections on the configured port.
func (qt *QUICTransport) Start(ctx context.Context) error {
	addr := fmt.Sprintf("%s:%d", qt.config.BindAddr, qt.config.QUICPort)

	tlsCfg := qt.generateTLSConfig()
	quicCfg := &quic.Config{
		MaxIncomingStreams:    1000,
		MaxIncomingUniStreams: 100,
		Allow0RTT:             true,
	}

	ln, err := quic.ListenAddr(addr, tlsCfg, quicCfg)
	if err != nil {
		return fmt.Errorf("quic listen on %s: %w", addr, err)
	}
	qt.listener = ln
	qt.logger.Info("QUIC relay transport started", "addr", addr)

	go qt.acceptLoop(ctx)
	return nil
}

// Stop shuts down the QUIC transport.
func (qt *QUICTransport) Stop() {
	if qt.listener != nil {
		qt.listener.Close()
	}
	qt.logger.Info("QUIC relay transport stopped")
}

func (qt *QUICTransport) acceptLoop(ctx context.Context) {
	for {
		conn, err := qt.listener.Accept(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				qt.logger.Error("QUIC accept error", "error", err)
				continue
			}
		}
		go qt.handleConn(ctx, conn)
	}
}

func (qt *QUICTransport) handleConn(ctx context.Context, conn *quic.Conn) {
	clientAddr := conn.RemoteAddr().String()
	qt.connsMu.Lock()
	qt.conns[clientAddr] = conn
	qt.connsMu.Unlock()

	defer func() {
		qt.connsMu.Lock()
		delete(qt.conns, clientAddr)
		qt.connsMu.Unlock()
	}()

	qt.logger.Info("QUIC client connected", "remote", clientAddr)

	// Accept streams and relay data
	for {
		stream, err := conn.AcceptStream(ctx)
		if err != nil {
			select {
			case <-ctx.Done():
				return
			default:
				qt.logger.Debug("QUIC stream accept error", "error", err)
				return
			}
		}
		go qt.handleStream(stream, conn.LocalAddr(), conn.RemoteAddr())
	}
}

func (qt *QUICTransport) handleStream(stream *quic.Stream, localAddr, remoteAddr net.Addr) {
	clientAddr := remoteAddr.String()
	qt.logger.Debug("QUIC stream opened", "client", clientAddr, "stream_id", stream.StreamID())
	pconn := protocol.NewConn(newQUICStreamConn(stream, localAddr, remoteAddr))
	msg, err := pconn.Read()
	if err != nil {
		qt.logger.Warn("failed to read QUIC work handshake", "client", clientAddr, "error", err)
		pconn.Close()
		return
	}
	if msg.Type != protocol.TypeWorkConn {
		qt.logger.Warn("unexpected QUIC first message", "client", clientAddr, "type", msg.Type)
		pconn.Close()
		return
	}
	if err := qt.server.handleWorkConnStream(pconn, msg); err != nil {
		qt.logger.Warn("failed to attach QUIC work stream", "client", clientAddr, "error", err)
		pconn.Close()
	}
}

func (qt *QUICTransport) generateTLSConfig() *tls.Config {
	// Use CA-signed certificates when configured
	if qt.config.TLSEnabled && qt.config.TLS.Enabled() {
		tlsCfg, err := tlsutil.LoadServerTLS(qt.config.TLS.CACert, qt.config.TLS.Cert, qt.config.TLS.Key)
		if err == nil {
			tlsCfg.NextProtos = []string{"nextunnel-quic-relay"}
			return tlsCfg
		}
		qt.logger.Warn("failed to load configured TLS certs for QUIC, falling back to self-signed", "error", err)
	}

	cert, err := generateSelfSignedRelayCert()
	if err != nil {
		qt.logger.Error("failed to generate QUIC relay certificate", "error", err)
		return &tls.Config{MinVersion: tls.VersionTLS13, NextProtos: []string{"nextunnel-quic-relay"}}
	}
	return &tls.Config{
		Certificates: []tls.Certificate{cert},
		MinVersion:   tls.VersionTLS13,
		NextProtos:   []string{"nextunnel-quic-relay"},
	}
}

func generateSelfSignedRelayCert() (tls.Certificate, error) {
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("generate key: %w", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(time.Now().UnixNano()),
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(24 * time.Hour),
		KeyUsage:     x509.KeyUsageDigitalSignature,
		ExtKeyUsage:  []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth},
	}
	certDER, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		return tls.Certificate{}, fmt.Errorf("create certificate: %w", err)
	}
	return tls.Certificate{Certificate: [][]byte{certDER}, PrivateKey: key}, nil
}

// quicStreamConn 将 QUIC stream 适配为 net.Conn，便于复用现有 Relay 会话桥接逻辑。
type quicStreamConn struct {
	stream     *quic.Stream
	localAddr  net.Addr
	remoteAddr net.Addr
}

func newQUICStreamConn(stream *quic.Stream, localAddr, remoteAddr net.Addr) *quicStreamConn {
	return &quicStreamConn{
		stream:     stream,
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
	}
}

func (c *quicStreamConn) Read(p []byte) (int, error) {
	return c.stream.Read(p)
}

func (c *quicStreamConn) Write(p []byte) (int, error) {
	return c.stream.Write(p)
}

func (c *quicStreamConn) Close() error {
	c.stream.CancelRead(0)
	return c.stream.Close()
}

func (c *quicStreamConn) LocalAddr() net.Addr {
	return c.localAddr
}

func (c *quicStreamConn) RemoteAddr() net.Addr {
	return c.remoteAddr
}

func (c *quicStreamConn) SetDeadline(t time.Time) error {
	if err := c.stream.SetReadDeadline(t); err != nil {
		return err
	}
	return c.stream.SetWriteDeadline(t)
}

func (c *quicStreamConn) SetReadDeadline(t time.Time) error {
	return c.stream.SetReadDeadline(t)
}

func (c *quicStreamConn) SetWriteDeadline(t time.Time) error {
	return c.stream.SetWriteDeadline(t)
}

// ClientCount returns the number of connected QUIC clients.
func (qt *QUICTransport) ClientCount() int {
	qt.connsMu.RLock()
	defer qt.connsMu.RUnlock()
	return len(qt.conns)
}
