package tunnel

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nextunnel/pkg/protocol"
)

// ControlClient manages the persistent control connection to the relay server.
type ControlClient struct {
	conn       *protocol.Conn
	clientID   string
	serverAddr string
	msgCh      chan *protocol.Message
	logger     *slog.Logger

	connected atomic.Bool
	mu        sync.Mutex // protects writes

	ctx    context.Context
	cancel context.CancelFunc
}

// NewControlClient creates a new control client.
func NewControlClient(clientID, serverAddr string, logger *slog.Logger) *ControlClient {
	return &ControlClient{
		clientID:   clientID,
		serverAddr: serverAddr,
		msgCh:      make(chan *protocol.Message, 32),
		logger:     logger,
	}
}

// Connect establishes the TCP connection and performs auth handshake.
func (c *ControlClient) Connect(ctx context.Context) error {
	c.ctx, c.cancel = context.WithCancel(ctx)

	conn, err := net.DialTimeout("tcp", c.serverAddr, 10*time.Second)
	if err != nil {
		return fmt.Errorf("dial server: %w", err)
	}

	pconn := protocol.NewConn(conn)

	// Send auth
	authMsg, err := protocol.NewAuthMessage(c.clientID)
	if err != nil {
		pconn.Close()
		return fmt.Errorf("create auth message: %w", err)
	}

	if err := pconn.Write(authMsg); err != nil {
		pconn.Close()
		return fmt.Errorf("send auth: %w", err)
	}

	// Read auth response
	resp, err := pconn.Read()
	if err != nil {
		pconn.Close()
		return fmt.Errorf("read auth response: %w", err)
	}

	if resp.Type != protocol.TypeAuthResp {
		pconn.Close()
		return fmt.Errorf("unexpected response type: %v", resp.Type)
	}

	payload, err := resp.DecodePayload()
	if err != nil {
		pconn.Close()
		return fmt.Errorf("decode auth response: %w", err)
	}

	authResp := payload.(*protocol.AuthRespMessage)
	if !authResp.Success {
		pconn.Close()
		return fmt.Errorf("auth rejected: %s", authResp.Error)
	}

	c.conn = pconn
	c.connected.Store(true)
	c.logger.Info("connected to server", "addr", c.serverAddr)

	// Start read loop
	go c.readLoop()

	return nil
}

// readLoop continuously reads messages from the server.
func (c *ControlClient) readLoop() {
	defer func() {
		c.connected.Store(false)
		close(c.msgCh)
	}()

	for {
		msg, err := c.conn.Read()
		if err != nil {
			select {
			case <-c.ctx.Done():
				return
			default:
				c.logger.Error("control conn read error", "error", err)
				return
			}
		}

		select {
		case c.msgCh <- msg:
		case <-c.ctx.Done():
			return
		}
	}
}

// Send writes a message to the server (thread-safe).
func (c *ControlClient) Send(msg *protocol.Message) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.connected.Load() {
		return fmt.Errorf("not connected")
	}
	return c.conn.Write(msg)
}

// Messages returns the channel for receiving server messages.
func (c *ControlClient) Messages() <-chan *protocol.Message {
	return c.msgCh
}

// IsConnected returns whether the control connection is active.
func (c *ControlClient) IsConnected() bool {
	return c.connected.Load()
}

// Close shuts down the control connection.
func (c *ControlClient) Close() error {
	if c.cancel != nil {
		c.cancel()
	}
	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}
