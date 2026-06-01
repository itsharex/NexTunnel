package nat

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/pion/stun/v2"
)

// STUNBinding holds the result of a single STUN binding request.
type STUNBinding struct {
	MappedAddr   net.UDPAddr   `json:"mapped_addr"`
	ResponseOrigin net.UDPAddr `json:"response_origin"`
	RTT          time.Duration `json:"rtt"`
}

// STUNClient defines the interface for STUN binding operations.
// This abstraction allows mocking in tests.
type STUNClient interface {
	BindingRequest(ctx context.Context, serverAddr string, localConn *net.UDPConn) (*STUNBinding, error)
	BindingRequestFromAlt(ctx context.Context, serverAddr string, localConn *net.UDPConn) (*STUNBinding, error)
}

// ClientOption configures the STUN client.
type ClientOption func(*Client)

// WithTimeout sets the per-request timeout.
func WithTimeout(d time.Duration) ClientOption {
	return func(c *Client) { c.timeout = d }
}

// WithRetries sets the number of retry attempts.
func WithRetries(n int) ClientOption {
	return func(c *Client) { c.retries = n }
}

// WithLogger sets the logger.
func WithLogger(l *slog.Logger) ClientOption {
	return func(c *Client) { c.logger = l }
}

// Client is a STUN binding client built on pion/stun.
type Client struct {
	timeout time.Duration
	retries int
	logger  *slog.Logger
}

// NewClient creates a new STUN client with the given options.
func NewClient(opts ...ClientOption) *Client {
	c := &Client{
		timeout: 3 * time.Second,
		retries: 3,
		logger:  slog.Default(),
	}
	for _, opt := range opts {
		opt(c)
	}
	return c
}

// BindingRequest sends a STUN Binding Request to serverAddr using the given localConn
// and returns the mapped (server-reflexive) address.
func (c *Client) BindingRequest(ctx context.Context, serverAddr string, localConn *net.UDPConn) (*STUNBinding, error) {
	return c.doBindingRequest(ctx, serverAddr, localConn, false)
}

// BindingRequestFromAlt sends a STUN Binding Request asking the server to respond
// from its alternate IP:port (RFC 3489 CHANGE-REQUEST attribute).
func (c *Client) BindingRequestFromAlt(ctx context.Context, serverAddr string, localConn *net.UDPConn) (*STUNBinding, error) {
	return c.doBindingRequest(ctx, serverAddr, localConn, true)
}

func (c *Client) doBindingRequest(ctx context.Context, serverAddr string, localConn *net.UDPConn, changeAddr bool) (*STUNBinding, error) {
	addr, err := net.ResolveUDPAddr("udp", serverAddr)
	if err != nil {
		return nil, fmt.Errorf("resolve STUN server %s: %w", serverAddr, err)
	}

	// Build STUN Binding Request
	msg := stun.MustBuild(stun.TransactionID, stun.BindingRequest)

	if changeAddr {
		// CHANGE-REQUEST attribute: request response from alternate IP and port
		// Value: 0x06 = change IP (0x04) + change port (0x02)
		msg.Add(stun.AttrChangeRequest, []byte{0x00, 0x00, 0x00, 0x06})
	}

	var lastErr error
	for attempt := 0; attempt <= c.retries; attempt++ {
		if attempt > 0 {
			// Exponential backoff between retries
			backoff := time.Duration(attempt) * 500 * time.Millisecond
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(backoff):
			}
		}

		binding, err := c.sendAndReceive(ctx, localConn, addr, msg)
		if err != nil {
			lastErr = err
			c.logger.Debug("STUN binding attempt failed", "attempt", attempt+1, "error", err)
			continue
		}
		return binding, nil
	}

	return nil, fmt.Errorf("STUN binding failed after %d attempts: %w", c.retries+1, lastErr)
}

func (c *Client) sendAndReceive(ctx context.Context, conn *net.UDPConn, serverAddr *net.UDPAddr, msg *stun.Message) (*STUNBinding, error) {
	start := time.Now()

	// Send the request
	if _, err := conn.WriteToUDP(msg.Raw, serverAddr); err != nil {
		return nil, fmt.Errorf("send STUN request: %w", err)
	}

	// Set read deadline
	deadline := time.Now().Add(c.timeout)
	if err := conn.SetReadDeadline(deadline); err != nil {
		return nil, fmt.Errorf("set read deadline: %w", err)
	}
	defer conn.SetReadDeadline(time.Time{}) // clear deadline

	// Read response
	buf := make([]byte, 1024)
	n, fromAddr, err := conn.ReadFromUDP(buf)
	if err != nil {
		return nil, fmt.Errorf("read STUN response: %w", err)
	}

	rtt := time.Since(start)

	// Parse STUN response
	resp := new(stun.Message)
	resp.Raw = append(resp.Raw[:0], buf[:n]...)
	if err := resp.Decode(); err != nil {
		return nil, fmt.Errorf("decode STUN response: %w", err)
	}

	// Check for STUN error response
	if resp.Type.Class == stun.ClassErrorResponse {
		var errCode stun.ErrorCodeAttribute
		if err := errCode.GetFrom(resp); err == nil {
			return nil, fmt.Errorf("STUN error: %d %s", errCode.Code, errCode.Reason)
		}
		return nil, fmt.Errorf("STUN error response")
	}

	// Extract mapped address (try XORMappedAddress first, then MappedAddress)
	var mappedAddr net.UDPAddr
	var xorAddr stun.XORMappedAddress
	if err := xorAddr.GetFrom(resp); err == nil {
		mappedAddr = net.UDPAddr{IP: xorAddr.IP, Port: xorAddr.Port}
	} else {
		var maddr stun.MappedAddress
		if err := maddr.GetFrom(resp); err == nil {
			mappedAddr = net.UDPAddr{IP: maddr.IP, Port: maddr.Port}
		} else {
			return nil, fmt.Errorf("no mapped address in STUN response")
		}
	}

	// Extract response origin (RESPONSE-ORIGIN attribute, type 0x802B)
	var responseOrigin net.UDPAddr
	const attrResponseOrigin stun.AttrType = 0x802B
	if raw, err := resp.Get(attrResponseOrigin); err == nil && len(raw) >= 8 {
		// Parse: 2 bytes reserved + 2 bytes port + 4 bytes IPv4
		port := int(raw[2])<<8 | int(raw[3])
		ip := net.IPv4(raw[4], raw[5], raw[6], raw[7])
		responseOrigin = net.UDPAddr{IP: ip, Port: port}
	} else {
		responseOrigin = *fromAddr
	}

	return &STUNBinding{
		MappedAddr:     mappedAddr,
		ResponseOrigin: responseOrigin,
		RTT:            rtt,
	}, nil
}
