package relay

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync/atomic"
	"time"

	"github.com/nextunnel/pkg/protocol"
)

// RelayClient manages a connection to a single relay server.
type RelayClient struct {
	config    RelayClientConfig
	conn      *protocol.Conn
	connected atomic.Bool
	latency   atomic.Value // time.Duration
	region    string

	ctx    context.Context
	cancel context.CancelFunc
	logger *slog.Logger
}

// NewRelayClient creates a new relay client.
func NewRelayClient(cfg RelayClientConfig) *RelayClient {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	ctx, cancel := context.WithCancel(context.Background())
	c := &RelayClient{
		config: cfg,
		region: cfg.Region,
		ctx:    ctx,
		cancel: cancel,
		logger: cfg.Logger,
	}
	c.latency.Store(time.Duration(0))
	return c
}

// Connect establishes a control connection to the relay server.
func (c *RelayClient) Connect(ctx context.Context) error {
	start := time.Now()

	dialer := net.Dialer{Timeout: 5 * time.Second}
	tcpConn, err := dialer.DialContext(ctx, "tcp", c.config.ServerAddr)
	if err != nil {
		return fmt.Errorf("dial relay %s: %w", c.config.ServerAddr, err)
	}

	pconn := protocol.NewConn(tcpConn)

	// Send auth message
	authMsg, err := protocol.NewAuthMessageWithToken(c.config.ClientID, c.config.AuthToken)
	if err != nil {
		tcpConn.Close()
		return fmt.Errorf("create auth: %w", err)
	}
	if err := pconn.Write(authMsg); err != nil {
		tcpConn.Close()
		return fmt.Errorf("send auth: %w", err)
	}

	// Read auth response
	resp, err := pconn.Read()
	if err != nil {
		tcpConn.Close()
		return fmt.Errorf("read auth resp: %w", err)
	}
	if resp.Type != protocol.TypeAuthResp {
		tcpConn.Close()
		return fmt.Errorf("unexpected auth response type: %d", resp.Type)
	}

	payload, err := resp.DecodePayload()
	if err != nil {
		tcpConn.Close()
		return fmt.Errorf("decode auth resp: %w", err)
	}
	authResp := payload.(*protocol.AuthRespMessage)
	if !authResp.Success {
		tcpConn.Close()
		return fmt.Errorf("auth rejected: %s", authResp.Error)
	}

	c.conn = pconn
	c.connected.Store(true)
	c.latency.Store(time.Since(start))

	c.logger.Info("relay connected", "addr", c.config.ServerAddr, "latency", c.latency.Load())
	return nil
}

// Close disconnects from the relay server.
func (c *RelayClient) Close() error {
	c.connected.Store(false)
	c.cancel()
	if c.conn != nil {
		return c.conn.Close()
	}
	c.logger.Info("relay disconnected", "addr", c.config.ServerAddr)
	return nil
}

// IsConnected returns true if the relay is connected.
func (c *RelayClient) IsConnected() bool {
	return c.connected.Load()
}

// Latency returns the connection latency.
func (c *RelayClient) Latency() time.Duration {
	return c.latency.Load().(time.Duration)
}

// Region returns the relay's geographic region.
func (c *RelayClient) Region() string {
	return c.region
}

// ServerAddr returns the relay server address.
func (c *RelayClient) ServerAddr() string {
	return c.config.ServerAddr
}
