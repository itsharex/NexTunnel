package quic

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net"

	q "github.com/quic-go/quic-go"
)

// Listener wraps a QUIC listener and accepts incoming connections.
type Listener struct {
	config   QUICConfig
	listener *q.Listener
	logger   *slog.Logger
}

// NewListener creates a new QUIC listener on the given address.
func NewListener(cfg QUICConfig) (*Listener, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	tlsCfg := cfg.TLSConfig
	if tlsCfg == nil {
		// Generate a self-signed certificate for testing
		cert, err := generateSelfSignedCert()
		if err != nil {
			return nil, fmt.Errorf("generate self-signed cert: %w", err)
		}
		tlsCfg = &tls.Config{
			Certificates: []tls.Certificate{cert},
			NextProtos:   []string{cfg.ALPN},
		}
	} else if len(tlsCfg.NextProtos) == 0 {
		tlsCfg.NextProtos = []string{cfg.ALPN}
	}

	quicCfg := &q.Config{
		MaxIncomingStreams:    cfg.MaxStreams,
		MaxIncomingUniStreams: cfg.MaxStreams,
		EnableDatagrams:       false,
		Allow0RTT:             cfg.Enable0RTT,
	}

	ln, err := q.ListenAddr(cfg.ListenAddr, tlsCfg, quicCfg)
	if err != nil {
		return nil, fmt.Errorf("quic listen on %s: %w", cfg.ListenAddr, err)
	}

	cfg.Logger.Info("QUIC listener started", "addr", ln.Addr())
	return &Listener{
		config:   cfg,
		listener: ln,
		logger:   cfg.Logger,
	}, nil
}

// Addr returns the listener's network address.
func (l *Listener) Addr() net.Addr {
	return l.listener.Addr()
}

// Accept waits for and returns the next incoming QUIC connection.
func (l *Listener) Accept(ctx context.Context) (*q.Conn, error) {
	return l.listener.Accept(ctx)
}

// AcceptStream accepts the next incoming connection and its first
// bidirectional stream, returning a StreamAdapter ready for Transport use.
func (l *Listener) AcceptStream(ctx context.Context) (*StreamAdapter, error) {
	conn, err := l.listener.Accept(ctx)
	if err != nil {
		return nil, fmt.Errorf("accept quic connection: %w", err)
	}

	stream, err := conn.AcceptStream(ctx)
	if err != nil {
		conn.CloseWithError(1, "failed to accept stream")
		return nil, fmt.Errorf("accept quic stream: %w", err)
	}

	l.logger.Debug("accepted QUIC stream",
		"remote", conn.RemoteAddr())

	return NewStreamAdapter(stream, conn.LocalAddr(), conn.RemoteAddr()), nil
}

// Close shuts down the QUIC listener.
func (l *Listener) Close() error {
	l.logger.Info("QUIC listener stopped")
	return l.listener.Close()
}
