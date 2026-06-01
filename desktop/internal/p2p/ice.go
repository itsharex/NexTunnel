package p2p

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/nextunnel/desktop/internal/nat"
	"github.com/pion/stun/v2"
)

// CandidateType identifies the type of ICE candidate.
type CandidateType string

const (
	CandidateHost           CandidateType = "host"
	CandidateServerReflexive CandidateType = "srflx"
	CandidateRelay          CandidateType = "relay"
)

// Candidate represents a single ICE transport candidate.
type Candidate struct {
	ID         string        `json:"id"`
	Type       CandidateType `json:"type"`
	Addr       net.UDPAddr   `json:"addr"`
	Priority   uint32        `json:"priority"`
	Foundation string        `json:"foundation"`
}

// ICERole defines the ICE agent role in a session.
type ICERole string

const (
	ICERoleControlling ICERole = "controlling"
	ICERoleControlled  ICERole = "controlled"
)

// AgentState represents the current state of the ICE agent.
type AgentState string

const (
	AgentStateNew        AgentState = "new"
	AgentStateGathering  AgentState = "gathering"
	AgentStateChecking   AgentState = "checking"
	AgentStateConnected  AgentState = "connected"
	AgentStateFailed     AgentState = "failed"
	AgentStateClosed     AgentState = "closed"
)

// PairState represents the state of a candidate pair.
type PairState string

const (
	PairStateWaiting    PairState = "waiting"
	PairStateInProgress PairState = "in_progress"
	PairStateSucceeded  PairState = "succeeded"
	PairStateFailed     PairState = "failed"
)

// CandidatePair is a local+remote candidate pair to be tested.
type CandidatePair struct {
	Local        *Candidate
	Remote       *Candidate
	State        PairState
	Priority     uint64
	Nominated    bool
	LastCheckAt  time.Time
}

// AgentConfig configures the ICE agent.
type AgentConfig struct {
	Role           ICERole
	STUNServer     string
	CheckInterval  time.Duration // interval between connectivity checks
	CheckTimeout   time.Duration // timeout per check
	MaxCheckTime   time.Duration // total time before giving up
	Logger         *slog.Logger
}

// DefaultAgentConfig returns sensible defaults.
func DefaultAgentConfig() AgentConfig {
	return AgentConfig{
		Role:          ICERoleControlling,
		CheckInterval: 50 * time.Millisecond,
		CheckTimeout:  500 * time.Millisecond,
		MaxCheckTime:  10 * time.Second,
	}
}

// Agent implements an ICE-lite agent for P2P candidate gathering and connectivity checks.
type Agent struct {
	config    AgentConfig
	state     atomic.Value // AgentState
	udpConn   *net.UDPConn
	stunClient *nat.Client

	localMu         sync.Mutex
	localCandidates []*Candidate

	remoteMu         sync.Mutex
	remoteCandidates []*Candidate

	checklistMu sync.Mutex
	checklist   []*CandidatePair

	pendingMu     sync.Mutex
	pendingChecks map[[stun.TransactionIDSize]byte]chan *stun.Message

	selectedPair atomic.Value // *CandidatePair
	wg           sync.WaitGroup
	readLoopStop chan struct{}

	onCandidate   func(Candidate)
	onStateChange func(AgentState)

	ctx    context.Context
	cancel context.CancelFunc
	logger *slog.Logger
}

// NewAgent creates a new ICE agent.
func NewAgent(cfg AgentConfig) *Agent {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	a := &Agent{
		config:        cfg,
		stunClient:    nat.NewClient(nat.WithLogger(cfg.Logger)),
		logger:        cfg.Logger,
		pendingChecks: make(map[[stun.TransactionIDSize]byte]chan *stun.Message),
		readLoopStop:  make(chan struct{}),
	}
	a.state.Store(AgentStateNew)
	return a
}

// SetCallbacks sets the candidate and state change callbacks.
func (a *Agent) SetCallbacks(onCandidate func(Candidate), onStateChange func(AgentState)) {
	a.onCandidate = onCandidate
	a.onStateChange = onStateChange
}

// GatherCandidates collects host and server-reflexive candidates.
// It binds a single UDP socket for all candidates.
func (a *Agent) GatherCandidates(ctx context.Context) ([]Candidate, error) {
	a.ctx, a.cancel = context.WithCancel(ctx)
	a.setState(AgentStateGathering)

	// Bind a single UDP socket
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		return nil, fmt.Errorf("bind UDP: %w", err)
	}
	a.udpConn = conn

	// Start background STUN read loop to handle incoming connectivity checks
	a.startReadLoop()

	var candidates []Candidate

	// Gather host candidates from local interfaces
	hostCandidates := a.gatherHostCandidates()
	for _, c := range hostCandidates {
		candidates = append(candidates, c)
		a.addLocalCandidate(c)
		a.fireCandidate(c)
	}

	// Gather server-reflexive candidates via STUN
	if a.config.STUNServer != "" {
		srflxCandidate, err := a.gatherSrflxCandidate(ctx)
		if err != nil {
			a.logger.Warn("STUN srflx candidate failed", "error", err)
		} else {
			candidates = append(candidates, *srflxCandidate)
			a.addLocalCandidate(*srflxCandidate)
			a.fireCandidate(*srflxCandidate)
		}
	}

	a.logger.Info("candidate gathering complete", "count", len(candidates))
	return candidates, nil
}

// AddRemoteCandidate adds a remote candidate and generates new pairs.
func (a *Agent) AddRemoteCandidate(c Candidate) {
	a.remoteMu.Lock()
	cCopy := c
	a.remoteCandidates = append(a.remoteCandidates, &cCopy)
	remotes := make([]*Candidate, len(a.remoteCandidates))
	copy(remotes, a.remoteCandidates)
	a.remoteMu.Unlock()

	a.localMu.Lock()
	locals := make([]*Candidate, len(a.localCandidates))
	copy(locals, a.localCandidates)
	a.localMu.Unlock()

	// Generate pairs with the new remote candidate against all local candidates
	newRemote := remotes[len(remotes)-1]
	for _, local := range locals {
		pair := a.createPair(local, newRemote)
		a.addPair(pair)
	}
}

// StartChecks runs the ICE connectivity check loop.
// Returns when a pair is nominated or the timeout expires.
func (a *Agent) StartChecks(ctx context.Context) error {
	a.setState(AgentStateChecking)

	deadline := time.After(a.config.MaxCheckTime)
	ticker := time.NewTicker(a.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			a.setState(AgentStateFailed)
			return ctx.Err()
		case <-deadline:
			a.setState(AgentStateFailed)
			return fmt.Errorf("ICE checks timed out after %v", a.config.MaxCheckTime)
		case <-ticker.C:
			pair := a.getNextWaitingPair()
			if pair == nil {
				// No more waiting pairs - check if we have a success
				if selected := a.getSelectedPair(); selected != nil {
					return nil
				}
				// Check if all pairs failed
				if a.allPairsFailed() {
					a.setState(AgentStateFailed)
					return fmt.Errorf("all candidate pairs failed")
				}
				continue
			}

			a.wg.Add(1)
			go func(p *CandidatePair) {
				defer a.wg.Done()
				a.checkPair(ctx, p)
			}(pair)
		}
	}
}

// GetSelectedPair returns the nominated candidate pair, or nil.
func (a *Agent) GetSelectedPair() *CandidatePair {
	v := a.selectedPair.Load()
	if v == nil {
		return nil
	}
	return v.(*CandidatePair)
}

// GetUDPConn returns the underlying UDP socket for P2P data.
func (a *Agent) GetUDPConn() *net.UDPConn {
	return a.udpConn
}

// GetState returns the current agent state.
func (a *Agent) GetState() AgentState {
	return a.state.Load().(AgentState)
}

// StopReadLoop stops the background STUN read loop without closing the UDP socket.
// Call this before handing the socket off to PunchEngine.
func (a *Agent) StopReadLoop() {
	select {
	case <-a.readLoopStop:
	default:
		close(a.readLoopStop)
	}
	a.wg.Wait()
}

// Close shuts down the ICE agent.
func (a *Agent) Close() error {
	a.setState(AgentStateClosed)
	if a.cancel != nil {
		a.cancel()
	}
	if a.udpConn != nil {
		return a.udpConn.Close()
	}
	return nil
}

// --- Internal methods ---

// startReadLoop starts a background goroutine that reads incoming UDP packets
// and responds to STUN Binding Requests (for ICE connectivity checks).
func (a *Agent) startReadLoop() {
	a.wg.Add(1)
	go func() {
		defer a.wg.Done()
		buf := make([]byte, 1500)
		for {
			select {
			case <-a.ctx.Done():
				return
			case <-a.readLoopStop:
				return
			default:
			}

			a.udpConn.SetReadDeadline(time.Now().Add(1 * time.Second))
			n, from, err := a.udpConn.ReadFromUDP(buf)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue // read deadline expired, loop back to check ctx
				}
				select {
				case <-a.ctx.Done():
					return
				default:
					a.logger.Debug("UDP read error", "error", err)
					continue
				}
			}

			// Try to parse as STUN message
			msg := new(stun.Message)
			msg.Raw = append(msg.Raw[:0], buf[:n]...)
			if err := msg.Decode(); err != nil {
				continue // Not a STUN message, ignore
			}

			if msg.Type == stun.BindingRequest {
				a.handleSTUNRequest(msg, from)
			} else if msg.Type.Class == stun.ClassSuccessResponse {
				// Dispatch STUN Binding Response to pending connectivity check
				a.pendingMu.Lock()
				ch, ok := a.pendingChecks[msg.TransactionID]
				if ok {
					delete(a.pendingChecks, msg.TransactionID)
				}
				a.pendingMu.Unlock()
				if ok {
					select {
					case ch <- msg:
					default:
					}
				}
			}
		}
	}()
}

// handleSTUNRequest responds to a STUN Binding Request from a peer.
func (a *Agent) handleSTUNRequest(req *stun.Message, from *net.UDPAddr) {
	resp, err := stun.Build(
		stun.BindingSuccess,
		&stun.XORMappedAddress{IP: from.IP, Port: from.Port},
		stun.NewTransactionIDSetter(req.TransactionID),
	)
	if err != nil {
		a.logger.Debug("failed to build STUN response", "error", err)
		return
	}
	if _, err := a.udpConn.WriteToUDP(resp.Raw, from); err != nil {
		a.logger.Debug("failed to send STUN response", "error", err)
	}
}

func (a *Agent) gatherHostCandidates() []Candidate {
	var candidates []Candidate

	addrs, err := net.InterfaceAddrs()
	if err != nil {
		a.logger.Warn("failed to enumerate interfaces", "error", err)
		return candidates
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() || ipNet.IP.To4() == nil {
			continue // Skip non-IPv4 and loopback
		}

		// Use the already-bound UDP socket's port
		localAddr := a.udpConn.LocalAddr().(*net.UDPAddr)
		c := Candidate{
			ID:         uuid.New().String(),
			Type:       CandidateHost,
			Addr:       net.UDPAddr{IP: ipNet.IP, Port: localAddr.Port},
			Priority:   computePriority(CandidateHost, 65535, 1),
			Foundation: computeFoundation(CandidateHost, ipNet.IP.String(), ""),
		}
		candidates = append(candidates, c)
	}

	return candidates
}

func (a *Agent) gatherSrflxCandidate(ctx context.Context) (*Candidate, error) {
	binding, err := a.stunClient.BindingRequest(ctx, a.config.STUNServer, a.udpConn)
	if err != nil {
		return nil, err
	}

	c := Candidate{
		ID:         uuid.New().String(),
		Type:       CandidateServerReflexive,
		Addr:       binding.MappedAddr,
		Priority:   computePriority(CandidateServerReflexive, 65535, 1),
		Foundation: computeFoundation(CandidateServerReflexive, binding.MappedAddr.IP.String(), a.config.STUNServer),
	}
	return &c, nil
}

func (a *Agent) createPair(local, remote *Candidate) *CandidatePair {
	pairPriority := computePairPriority(local.Priority, remote.Priority)
	return &CandidatePair{
		Local:    local,
		Remote:   remote,
		State:    PairStateWaiting,
		Priority: pairPriority,
	}
}

func (a *Agent) addLocalCandidate(c Candidate) {
	a.localMu.Lock()
	a.localCandidates = append(a.localCandidates, &c)
	a.localMu.Unlock()
}

func (a *Agent) addPair(pair *CandidatePair) {
	a.checklistMu.Lock()
	a.checklist = append(a.checklist, pair)
	sort.Slice(a.checklist, func(i, j int) bool {
		return a.checklist[i].Priority > a.checklist[j].Priority
	})
	a.checklistMu.Unlock()
}

func (a *Agent) getNextWaitingPair() *CandidatePair {
	a.checklistMu.Lock()
	defer a.checklistMu.Unlock()

	for _, pair := range a.checklist {
		if pair.State == PairStateWaiting {
			pair.State = PairStateInProgress
			pair.LastCheckAt = time.Now()
			return pair
		}
	}
	return nil
}

func (a *Agent) checkPair(ctx context.Context, pair *CandidatePair) {
	// Build a STUN Binding Request
	msg, err := stun.Build(stun.TransactionID, stun.BindingRequest)
	if err != nil {
		a.updatePairState(pair, PairStateFailed)
		return
	}

	// Register the pending check
	respCh := make(chan *stun.Message, 1)
	a.pendingMu.Lock()
	a.pendingChecks[msg.TransactionID] = respCh
	a.pendingMu.Unlock()

	// Send the STUN request to the remote candidate
	remoteAddr := &net.UDPAddr{IP: pair.Remote.Addr.IP, Port: pair.Remote.Addr.Port}
	if _, err := a.udpConn.WriteToUDP(msg.Raw, remoteAddr); err != nil {
		a.pendingMu.Lock()
		delete(a.pendingChecks, msg.TransactionID)
		a.pendingMu.Unlock()
		a.logger.Debug("connectivity check send failed",
			"local", pair.Local.Addr.String(),
			"remote", pair.Remote.Addr.String(),
			"error", err)
		a.updatePairState(pair, PairStateFailed)
		return
	}

	// Wait for response or timeout
	select {
	case <-respCh:
		a.logger.Info("connectivity check succeeded",
			"local", pair.Local.Addr.String(),
			"remote", pair.Remote.Addr.String())
		a.updatePairState(pair, PairStateSucceeded)

		// Nominate the first successful pair
		if a.selectedPair.Load() == nil {
			pair.Nominated = true
			a.selectedPair.Store(pair)
			a.setState(AgentStateConnected)
			a.logger.Info("ICE pair nominated",
				"local", pair.Local.Addr.String(),
				"remote", pair.Remote.Addr.String())
		}
	case <-time.After(a.config.CheckTimeout):
		a.pendingMu.Lock()
		delete(a.pendingChecks, msg.TransactionID)
		a.pendingMu.Unlock()
		a.logger.Debug("connectivity check timed out",
			"local", pair.Local.Addr.String(),
			"remote", pair.Remote.Addr.String())
		a.updatePairState(pair, PairStateFailed)
	case <-ctx.Done():
		a.pendingMu.Lock()
		delete(a.pendingChecks, msg.TransactionID)
		a.pendingMu.Unlock()
		a.updatePairState(pair, PairStateFailed)
	}
}

func (a *Agent) updatePairState(pair *CandidatePair, state PairState) {
	a.checklistMu.Lock()
	defer a.checklistMu.Unlock()
	pair.State = state
}

func (a *Agent) getSelectedPair() *CandidatePair {
	v := a.selectedPair.Load()
	if v == nil {
		return nil
	}
	return v.(*CandidatePair)
}

func (a *Agent) allPairsFailed() bool {
	a.checklistMu.Lock()
	defer a.checklistMu.Unlock()

	if len(a.checklist) == 0 {
		return false // No pairs yet
	}
	for _, pair := range a.checklist {
		if pair.State != PairStateFailed {
			return false
		}
	}
	return true
}

func (a *Agent) setState(state AgentState) {
	a.state.Store(state)
	if a.onStateChange != nil {
		a.onStateChange(state)
	}
}

func (a *Agent) fireCandidate(c Candidate) {
	if a.onCandidate != nil {
		a.onCandidate(c)
	}
}

// --- Priority computation ---

func computePriority(cType CandidateType, localPref uint32, componentID uint32) uint32 {
	var typePref uint32
	switch cType {
	case CandidateHost:
		typePref = 126
	case CandidateServerReflexive:
		typePref = 100
	case CandidateRelay:
		typePref = 0
	}
	return typePref<<24 + localPref<<8 + (256 - componentID)
}

func computePairPriority(controllingPriority, controlledPriority uint32) uint64 {
	// g = 2^32 * MIN(G,D) + 2 * MAX(G,D) + (G > D ? 1 : 0)
	g := uint64(controllingPriority)
	d := uint64(controlledPriority)
	min := g
	max := d
	if g > d {
		min = d
		max = g
	}
	return (1<<32)*min + 2*max + boolToUint64(g > d)
}

func boolToUint64(b bool) uint64 {
	if b {
		return 1
	}
	return 0
}

func computeFoundation(cType CandidateType, baseAddr, serverAddr string) string {
	h := sha256.New()
	h.Write([]byte(string(cType)))
	h.Write([]byte(baseAddr))
	h.Write([]byte(serverAddr))
	return hex.EncodeToString(h.Sum(nil))[:8]
}
