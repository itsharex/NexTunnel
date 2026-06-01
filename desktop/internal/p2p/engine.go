package p2p

import (
	"context"
	"encoding/binary"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"

	"github.com/google/uuid"
	"github.com/nextunnel/desktop/internal/nat"
	"github.com/nextunnel/pkg/crypto"
	"github.com/nextunnel/pkg/protocol"
)

// SessionState tracks the lifecycle of a P2P session.
type SessionState string

const (
	SessionIdle             SessionState = "idle"
	SessionDetectingNAT     SessionState = "detecting_nat"
	SessionGatheringCands   SessionState = "gathering_candidates"
	SessionExchangingCands  SessionState = "exchanging_candidates"
	SessionChecking         SessionState = "checking_connectivity"
	SessionPunching         SessionState = "punching"
	SessionEstablishing     SessionState = "establishing_tunnel"
	SessionConnected        SessionState = "connected"
	SessionFailed           SessionState = "failed"
	SessionClosed           SessionState = "closed"
)

// Transport is the abstraction for relay or P2P data transport.
type Transport interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

// ControlChannel is the interface for signaling message exchange.
type ControlChannel interface {
	Send(*protocol.Message) error
}

// EngineConfig configures the P2P engine.
type EngineConfig struct {
	ClientID   string
	Control    ControlChannel
	STUNServer string
	Logger     *slog.Logger
}

// Engine orchestrates the full P2P connection flow: NAT detection -> ICE -> punch -> WireGuard.
type Engine struct {
	config    EngineConfig
	natResult *nat.NATResult
	wgPrivKey string
	wgPubKey  string

	sessions sync.Map // sessionID -> *Session
	state    atomic.Value

	onStateChange func(SessionState)

	ctx    context.Context
	cancel context.CancelFunc
	logger *slog.Logger
}

// Session holds the state for a single P2P connection attempt.
type Session struct {
	ID         string
	PeerID     string
	State      atomic.Value
	iceAgent   *Agent
	punchEng   *PunchEngine
	wgTunnel   *WGTunnel
	transport  *p2pTransport
}

// NewEngine creates a new P2P engine.
func NewEngine(cfg EngineConfig) (*Engine, error) {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	// Generate WireGuard keypair
	privKey, pubKey, err := crypto.GenerateWGKeyPair()
	if err != nil {
		return nil, fmt.Errorf("generate WG keypair: %w", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	e := &Engine{
		config:    cfg,
		wgPrivKey: privKey,
		wgPubKey:  pubKey,
		ctx:       ctx,
		cancel:    cancel,
		logger:    cfg.Logger,
	}
	e.state.Store(SessionIdle)
	return e, nil
}

// GetState returns the current engine state.
func (e *Engine) GetState() SessionState {
	return e.state.Load().(SessionState)
}

// GetNATType returns the detected NAT type, or empty string if not detected.
func (e *Engine) GetNATType() string {
	if e.natResult != nil {
		return string(e.natResult.Type)
	}
	return ""
}

// WGPUbKey returns this engine's WireGuard public key.
func (e *Engine) WGPUbKey() string {
	return e.wgPubKey
}

// DetectNAT runs NAT type detection and caches the result.
func (e *Engine) DetectNAT(ctx context.Context) (*nat.NATResult, error) {
	e.setState(SessionDetectingNAT)

	stunClient := nat.NewClient(nat.WithLogger(e.logger))
	detector := nat.NewDetector(e.config.STUNServer, e.config.STUNServer, stunClient, e.logger)

	result, err := detector.Detect(ctx)
	if err != nil {
		e.setState(SessionFailed)
		return nil, err
	}

	e.natResult = result
	e.logger.Info("NAT detection complete", "type", result.Type, "public", result.PublicAddr)
	return result, nil
}

// InitiateP2P starts a full P2P connection flow as the initiator.
func (e *Engine) InitiateP2P(ctx context.Context, peerClientID string) (Transport, error) {
	sessionID := uuid.New().String()
	session := &Session{ID: sessionID, PeerID: peerClientID}
	session.State.Store(SessionGatheringCands)
	e.sessions.Store(sessionID, session)

	// 1. Detect NAT if not cached
	if e.natResult == nil {
		if _, err := e.DetectNAT(ctx); err != nil {
			return nil, fmt.Errorf("NAT detection: %w", err)
		}
	}

	// 2. Gather ICE candidates
	iceCfg := DefaultAgentConfig()
	iceCfg.STUNServer = e.config.STUNServer
	iceCfg.Logger = e.logger

	agent := NewAgent(iceCfg)
	session.iceAgent = agent

	candidates, err := agent.GatherCandidates(ctx)
	if err != nil {
		agent.Close()
		return nil, fmt.Errorf("gather candidates: %w", err)
	}

	// 3. Convert candidates to JSON and send offer
	candJSONs := candidatesToJSON(candidates)
	offer, err := protocol.NewP2POfferMessage(sessionID, e.config.ClientID, peerClientID,
		string(e.natResult.Type), e.wgPubKey, candJSONs)
	if err != nil {
		agent.Close()
		return nil, fmt.Errorf("create offer: %w", err)
	}

	if err := e.config.Control.Send(offer); err != nil {
		agent.Close()
		return nil, fmt.Errorf("send offer: %w", err)
	}

	e.setState(SessionExchangingCands)

	// 4. Wait for answer (this would be handled by the message handler in practice)
	// For now, we assume the answer is processed via HandleP2PAnswer
	return nil, fmt.Errorf("initiate P2P: signaling requires async message handling")
}

// HandleP2POffer processes an incoming P2P offer and returns a transport.
func (e *Engine) HandleP2POffer(ctx context.Context, offer *protocol.P2POfferMessage) (Transport, error) {
	session := &Session{ID: offer.SessionID, PeerID: offer.FromClientID}
	session.State.Store(SessionGatheringCands)
	e.sessions.Store(offer.SessionID, session)

	// Check NAT compatibility
	if e.natResult != nil && !e.natResult.IsP2PPossible(nat.NATType(offer.NATType)) {
		e.sendAnswer(offer, false, "incompatible NAT types")
		return nil, fmt.Errorf("incompatible NAT: local=%s, remote=%s", e.natResult.Type, offer.NATType)
	}

	// Detect NAT if not cached
	if e.natResult == nil {
		if _, err := e.DetectNAT(ctx); err != nil {
			e.sendAnswer(offer, false, "NAT detection failed")
			return nil, err
		}
	}

	// Gather ICE candidates
	iceCfg := DefaultAgentConfig()
	iceCfg.STUNServer = e.config.STUNServer
	iceCfg.Logger = e.logger

	agent := NewAgent(iceCfg)
	session.iceAgent = agent

	candidates, err := agent.GatherCandidates(ctx)
	if err != nil {
		agent.Close()
		e.sendAnswer(offer, false, "candidate gathering failed")
		return nil, err
	}

	// Feed remote candidates
	for _, c := range offer.Candidates {
		agent.AddRemoteCandidate(candidateFromJSON(c))
	}

	// Send answer
	candJSONs := candidatesToJSON(candidates)
	if err := e.sendAnswerWithCandidates(offer, candJSONs); err != nil {
		agent.Close()
		return nil, err
	}

	// Run connectivity checks
	e.setState(SessionChecking)
	if err := agent.StartChecks(ctx); err != nil {
		agent.Close()
		return nil, fmt.Errorf("ICE checks failed: %w", err)
	}

	pair := agent.GetSelectedPair()
	if pair == nil {
		agent.Close()
		return nil, fmt.Errorf("no ICE pair selected")
	}

	// Hole punching
	e.setState(SessionPunching)
	var sessionIDBytes [16]byte
	copy(sessionIDBytes[:], offer.SessionID)

	punchEng := NewPunchEngine(PunchConfig{
		SessionID:  sessionIDBytes,
		UDPConn:    agent.GetUDPConn(),
		RemoteAddr: &pair.Remote.Addr,
		Role:       PunchRoleResponder,
		Logger:     e.logger,
	})
	session.punchEng = punchEng

	punchResult, err := punchEng.Punch(ctx)
	if err != nil {
		agent.Close()
		return nil, fmt.Errorf("hole punch failed: %w", err)
	}

	// WireGuard tunnel
	e.setState(SessionEstablishing)
	wgTunnel := NewWGTunnel(WGConfig{
		PrivateKey:    e.wgPrivKey,
		PeerPublicKey: offer.WGPublicKey,
		PeerAddr:      "10.7.0.1/32",
		Logger:        e.logger,
	})
	session.wgTunnel = wgTunnel

	bind := newNetBind(agent.GetUDPConn(), &punchResult.RemoteAddr)
	if err := wgTunnel.Start(bind); err != nil {
		agent.Close()
		return nil, fmt.Errorf("start wireguard: %w", err)
	}

	e.setState(SessionConnected)
	e.logger.Info("P2P connection established", "peer", offer.FromClientID, "session", offer.SessionID)

	transport := newP2PTransport(wgTunnel.TUN(), agent.GetUDPConn(), &punchResult.RemoteAddr)
	session.transport = transport
	return transport, nil
}

// HandleP2PAnswer processes an incoming P2P answer.
func (e *Engine) HandleP2PAnswer(ctx context.Context, answer *protocol.P2PAnswerMessage) error {
	v, ok := e.sessions.Load(answer.SessionID)
	if !ok {
		return fmt.Errorf("unknown session: %s", answer.SessionID)
	}
	session := v.(*Session)

	if !answer.Accept {
		session.State.Store(SessionFailed)
		return fmt.Errorf("P2P rejected: %s", answer.Reason)
	}

	agent := session.iceAgent
	if agent == nil {
		return fmt.Errorf("no ICE agent for session")
	}

	// Feed remote candidates
	for _, c := range answer.Candidates {
		agent.AddRemoteCandidate(candidateFromJSON(c))
	}

	// Run connectivity checks
	e.setState(SessionChecking)
	if err := agent.StartChecks(ctx); err != nil {
		return fmt.Errorf("ICE checks failed: %w", err)
	}

	pair := agent.GetSelectedPair()
	if pair == nil {
		return fmt.Errorf("no ICE pair selected")
	}

	// Hole punching
	e.setState(SessionPunching)
	var sessionIDBytes [16]byte
	copy(sessionIDBytes[:], answer.SessionID)

	punchEng := NewPunchEngine(PunchConfig{
		SessionID:  sessionIDBytes,
		UDPConn:    agent.GetUDPConn(),
		RemoteAddr: &pair.Remote.Addr,
		Role:       PunchRoleInitiator,
		Logger:     e.logger,
	})
	session.punchEng = punchEng

	punchResult, err := punchEng.Punch(ctx)
	if err != nil {
		return fmt.Errorf("hole punch failed: %w", err)
	}

	// WireGuard tunnel
	e.setState(SessionEstablishing)
	wgTunnel := NewWGTunnel(WGConfig{
		PrivateKey:    e.wgPrivKey,
		PeerPublicKey: answer.WGPublicKey,
		PeerAddr:      "10.7.0.2/32",
		Logger:        e.logger,
	})
	session.wgTunnel = wgTunnel

	bind := newNetBind(agent.GetUDPConn(), &punchResult.RemoteAddr)
	if err := wgTunnel.Start(bind); err != nil {
		return fmt.Errorf("start wireguard: %w", err)
	}

	e.setState(SessionConnected)
	e.logger.Info("P2P connection established", "peer", answer.FromClientID, "session", answer.SessionID)

	transport := newP2PTransport(wgTunnel.TUN(), agent.GetUDPConn(), &punchResult.RemoteAddr)
	session.transport = transport
	return nil
}

// Close shuts down the P2P engine and all sessions.
func (e *Engine) Close() {
	e.cancel()
	e.sessions.Range(func(key, value any) bool {
		session := value.(*Session)
		if session.wgTunnel != nil {
			session.wgTunnel.Close()
		}
		if session.iceAgent != nil {
			session.iceAgent.Close()
		}
		e.sessions.Delete(key)
		return true
	})
	e.setState(SessionClosed)
}

// --- Internal helpers ---

func (e *Engine) setState(state SessionState) {
	e.state.Store(state)
	if e.onStateChange != nil {
		e.onStateChange(state)
	}
}

func (e *Engine) sendAnswer(offer *protocol.P2POfferMessage, accept bool, reason string) {
	msg, _ := protocol.NewP2PAnswerMessage(offer.SessionID, e.config.ClientID, offer.FromClientID,
		"", "", nil, accept, reason)
	if msg != nil {
		e.config.Control.Send(msg)
	}
}

func (e *Engine) sendAnswerWithCandidates(offer *protocol.P2POfferMessage, candidates []protocol.ICECandidateJSON) error {
	natType := ""
	if e.natResult != nil {
		natType = string(e.natResult.Type)
	}
	msg, err := protocol.NewP2PAnswerMessage(offer.SessionID, e.config.ClientID, offer.FromClientID,
		natType, e.wgPubKey, candidates, true, "")
	if err != nil {
		return err
	}
	return e.config.Control.Send(msg)
}

func candidatesToJSON(candidates []Candidate) []protocol.ICECandidateJSON {
	result := make([]protocol.ICECandidateJSON, len(candidates))
	for i, c := range candidates {
		result[i] = protocol.ICECandidateJSON{
			ID:         c.ID,
			Type:       string(c.Type),
			Addr:       c.Addr.String(),
			Priority:   c.Priority,
			Foundation: c.Foundation,
		}
	}
	return result
}

func candidateFromJSON(j protocol.ICECandidateJSON) Candidate {
	addr, _ := net.ResolveUDPAddr("udp", j.Addr)
	c := Candidate{
		ID:         j.ID,
		Type:       CandidateType(j.Type),
		Priority:   j.Priority,
		Foundation: j.Foundation,
	}
	if addr != nil {
		c.Addr = *addr
	}
	return c
}

// sessionIDToBytes converts a session ID string to a 16-byte array.
func sessionIDToBytes(id string) [16]byte {
	var b [16]byte
	copy(b[:], id)
	return b
}

// --- p2pTransport implements the Transport interface over WireGuard TUN ---

type p2pTransport struct {
	tun        *netTun
	udpConn    *net.UDPConn
	remoteAddr *net.UDPAddr
}

func newP2PTransport(tun *netTun, udpConn *net.UDPConn, remote *net.UDPAddr) *p2pTransport {
	return &p2pTransport{tun: tun, udpConn: udpConn, remoteAddr: remote}
}

func (t *p2pTransport) Read(buf []byte) (int, error) {
	data, err := t.tun.ReadToApp()
	if err != nil {
		return 0, err
	}
	return copy(buf, data), nil
}

func (t *p2pTransport) Write(data []byte) (int, error) {
	if err := t.tun.WriteFromApp(data); err != nil {
		return 0, err
	}
	return len(data), nil
}

func (t *p2pTransport) Close() error {
	return t.tun.Close()
}

func (t *p2pTransport) LocalAddr() net.Addr {
	return t.udpConn.LocalAddr()
}

func (t *p2pTransport) RemoteAddr() net.Addr {
	return t.remoteAddr
}

// --- Suppress unused import warnings ---
var _ = binary.BigEndian
var _ = uuid.New
