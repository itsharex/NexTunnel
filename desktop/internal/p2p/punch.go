package p2p

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"
	"sync/atomic"
	"time"
)

// Punch packet magic bytes and flags.
const (
	punchMagic    uint16 = 0x4E50 // "NP"
	punchFlagSYN  byte   = 0x01
	punchFlagACK  byte   = 0x02
	punchFlagSYNACK byte = 0x03
	punchPacketSize      = 27 // 2 magic + 16 session + 1 flags + 8 nonce
)

// PunchRole defines whether this side initiates or responds.
type PunchRole string

const (
	PunchRoleInitiator PunchRole = "initiator"
	PunchRoleResponder PunchRole = "responder"
)

// PunchState tracks the hole punching progress.
type PunchState string

const (
	PunchStateIdle     PunchState = "idle"
	PunchStatePunching PunchState = "punching"
	PunchStateVerified PunchState = "verified"
	PunchStateFailed   PunchState = "failed"
)

// PunchConfig configures the hole punching engine.
type PunchConfig struct {
	SessionID    [16]byte
	UDPConn      *net.UDPConn
	RemoteAddr   *net.UDPAddr
	Role         PunchRole
	Timeout      time.Duration
	SendInterval time.Duration
	Logger       *slog.Logger
}

// PunchResult holds the outcome of a hole punching attempt.
type PunchResult struct {
	Success    bool
	RemoteAddr net.UDPAddr
	RTT        time.Duration
}

// PunchEngine performs UDP hole punching with simultaneous-open coordination.
type PunchEngine struct {
	config PunchConfig
	state  atomic.Value
	logger *slog.Logger
}

// NewPunchEngine creates a new hole punching engine.
func NewPunchEngine(cfg PunchConfig) *PunchEngine {
	if cfg.Timeout == 0 {
		cfg.Timeout = 10 * time.Second
	}
	if cfg.SendInterval == 0 {
		cfg.SendInterval = 100 * time.Millisecond
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	p := &PunchEngine{config: cfg, logger: cfg.Logger}
	p.state.Store(PunchStateIdle)
	return p
}

// Punch performs the hole punching sequence.
// Both sides send SYN packets; upon receiving a peer SYN, respond with ACK.
// When an ACK is received, the path is verified.
func (p *PunchEngine) Punch(ctx context.Context) (*PunchResult, error) {
	p.state.Store(PunchStatePunching)
	startTime := time.Now()

	deadline := time.After(p.config.Timeout)
	ticker := time.NewTicker(p.config.SendInterval)
	defer ticker.Stop()

	// Generate a nonce for RTT measurement
	nonce := uint64(time.Now().UnixNano())

	// Start a reader goroutine to receive punch packets
	respCh := make(chan punchResponse, 1)
	doneCh := make(chan struct{})
	defer close(doneCh)

	go p.readPunchPackets(ctx, respCh, doneCh, nonce)

	// Send loop: send SYN packets periodically
	for {
		select {
		case <-ctx.Done():
			p.state.Store(PunchStateFailed)
			return nil, ctx.Err()
		case <-deadline:
			p.state.Store(PunchStateFailed)
			return nil, fmt.Errorf("hole punching timed out after %v", p.config.Timeout)
		case <-ticker.C:
			pkt := buildPunchPacket(p.config.SessionID, punchFlagSYN, nonce)
			if _, err := p.config.UDPConn.WriteToUDP(pkt, p.config.RemoteAddr); err != nil {
				p.logger.Debug("failed to send punch SYN", "error", err)
			}
		case resp := <-respCh:
			if resp.success {
				rtt := time.Since(startTime)
				p.state.Store(PunchStateVerified)
				p.logger.Info("hole punch verified",
					"remote", resp.remoteAddr.String(),
					"rtt", rtt)
				return &PunchResult{
					Success:    true,
					RemoteAddr: resp.remoteAddr,
					RTT:        rtt,
				}, nil
			}
		}
	}
}

// GetState returns the current punch state.
func (p *PunchEngine) GetState() PunchState {
	return p.state.Load().(PunchState)
}

type punchResponse struct {
	success    bool
	remoteAddr net.UDPAddr
}

func (p *PunchEngine) readPunchPackets(ctx context.Context, respCh chan<- punchResponse, doneCh <-chan struct{}, localNonce uint64) {
	buf := make([]byte, 1500)
	for {
		select {
		case <-doneCh:
			return
		case <-ctx.Done():
			return
		default:
		}

		p.config.UDPConn.SetReadDeadline(time.Now().Add(1 * time.Second))
		n, from, err := p.config.UDPConn.ReadFromUDP(buf)
		if err != nil {
			continue
		}

		if n < punchPacketSize {
			continue // too small
		}

		magic := binary.BigEndian.Uint16(buf[0:2])
		if magic != punchMagic {
			continue // not a punch packet
		}

		flags := buf[18]
		remoteNonce := binary.BigEndian.Uint64(buf[19:27])
		_ = remoteNonce

		// Check session ID match
		var sessionID [16]byte
		copy(sessionID[:], buf[2:18])
		if sessionID != p.config.SessionID {
			continue
		}

		switch {
		case flags == punchFlagSYN:
			// Received SYN from peer, send SYNACK
			pkt := buildPunchPacket(p.config.SessionID, punchFlagSYNACK, localNonce)
			p.config.UDPConn.WriteToUDP(pkt, from)
			p.logger.Debug("punch: received SYN, sent SYNACK", "from", from)

		case flags == punchFlagACK || flags == punchFlagSYNACK:
			// Received ACK or SYNACK - path verified!
			if flags == punchFlagSYNACK {
				// Send final ACK
				pkt := buildPunchPacket(p.config.SessionID, punchFlagACK, localNonce)
				p.config.UDPConn.WriteToUDP(pkt, from)
			}
			select {
			case respCh <- punchResponse{success: true, remoteAddr: *from}:
			default:
			}
			return
		}
	}
}

func buildPunchPacket(sessionID [16]byte, flags byte, nonce uint64) []byte {
	pkt := make([]byte, punchPacketSize)
	binary.BigEndian.PutUint16(pkt[0:2], punchMagic)
	copy(pkt[2:18], sessionID[:])
	pkt[18] = flags
	binary.BigEndian.PutUint64(pkt[19:27], nonce)
	return pkt
}

// ParsePunchPacket parses a punch packet and returns the session ID, flags, and nonce.
func ParsePunchPacket(pkt []byte) (sessionID [16]byte, flags byte, nonce uint64, ok bool) {
	if len(pkt) < punchPacketSize {
		return sessionID, 0, 0, false
	}
	magic := binary.BigEndian.Uint16(pkt[0:2])
	if magic != punchMagic {
		return sessionID, 0, 0, false
	}
	copy(sessionID[:], pkt[2:18])
	flags = pkt[18]
	nonce = binary.BigEndian.Uint64(pkt[19:27])
	return sessionID, flags, nonce, true
}
