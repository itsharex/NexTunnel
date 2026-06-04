package quic

import (
	"crypto/tls"
	"log/slog"
	"time"
)

// QUICTransportState represents the state of a QUIC transport.
type QUICTransportState string

const (
	QUICStateIdle       QUICTransportState = "idle"
	QUICStateConnecting QUICTransportState = "connecting"
	QUICStateConnected  QUICTransportState = "connected"
	QUICStateMigrating  QUICTransportState = "migrating"
	QUICStateClosed     QUICTransportState = "closed"
	QUICStateError      QUICTransportState = "error"
)

// QUICConfig holds the configuration for a QUIC transport.
type QUICConfig struct {
	ListenAddr       string
	TLSConfig        *tls.Config
	Enable0RTT       bool
	EnableMigration  bool
	MaxStreams       int64
	KeepAlive        time.Duration
	HandshakeTimeout time.Duration
	ALPN             string
	Logger           *slog.Logger
}

// Option configures a QUICConfig.
type Option func(*QUICConfig)

// WithTLSConfig sets the TLS configuration.
func WithTLSConfig(cfg *tls.Config) Option {
	return func(c *QUICConfig) { c.TLSConfig = cfg }
}

// With0RTT enables or disables 0-RTT connection resumption.
func With0RTT(enabled bool) Option {
	return func(c *QUICConfig) { c.Enable0RTT = enabled }
}

// WithConnectionMigration enables or disables QUIC connection migration.
func WithConnectionMigration(enabled bool) Option {
	return func(c *QUICConfig) { c.EnableMigration = enabled }
}

// WithMaxStreams sets the maximum number of concurrent bidirectional streams.
func WithMaxStreams(n int64) Option {
	return func(c *QUICConfig) { c.MaxStreams = n }
}

// WithKeepAlive sets the keep-alive interval.
func WithKeepAlive(d time.Duration) Option {
	return func(c *QUICConfig) { c.KeepAlive = d }
}

// WithListenAddr sets the local listen address.
func WithListenAddr(addr string) Option {
	return func(c *QUICConfig) { c.ListenAddr = addr }
}

// WithALPN sets the ALPN identifier.
func WithALPN(alpn string) Option {
	return func(c *QUICConfig) { c.ALPN = alpn }
}

// WithQUICLogger sets the logger.
func WithQUICLogger(l *slog.Logger) Option {
	return func(c *QUICConfig) { c.Logger = l }
}

// DefaultQUICConfig returns a QUICConfig with sensible defaults.
func DefaultQUICConfig() QUICConfig {
	return QUICConfig{
		ListenAddr:       "0.0.0.0:0",
		Enable0RTT:       false,
		EnableMigration:  true,
		MaxStreams:       100,
		KeepAlive:        15 * time.Second,
		HandshakeTimeout: 10 * time.Second,
		ALPN:             "nextunnel",
		Logger:           slog.Default(),
	}
}
