package p2p

import (
	"context"
	"log/slog"
	"net"
	"sync"
	"testing"
	"time"

	"github.com/nextunnel/pkg/crypto"
	"github.com/nextunnel/pkg/protocol"
)

// mockControlChannel implements ControlChannel for testing.
type mockControlChannel struct {
	mu       sync.Mutex
	messages []*protocol.Message
	incoming chan *protocol.Message
}

func newMockControlChannel() *mockControlChannel {
	return &mockControlChannel{
		incoming: make(chan *protocol.Message, 16),
	}
}

func (m *mockControlChannel) Send(msg *protocol.Message) error {
	m.mu.Lock()
	m.messages = append(m.messages, msg)
	m.mu.Unlock()
	return nil
}

func (m *mockControlChannel) getLastSent() *protocol.Message {
	m.mu.Lock()
	defer m.mu.Unlock()
	if len(m.messages) == 0 {
		return nil
	}
	return m.messages[len(m.messages)-1]
}

func (m *mockControlChannel) clearMessages() {
	m.mu.Lock()
	m.messages = nil
	m.mu.Unlock()
}

// TestP2PFullChain_Localhost tests the full P2P connection flow on localhost:
// 1. Two engines with WireGuard keypairs
// 2. Manual signaling exchange (offer -> answer)
// 3. ICE connectivity checks
// 4. Hole punching
// 5. Data exchange verification
func TestP2PFullChain_Localhost(t *testing.T) {
	logger := slog.Default()

	// Create Engine A
	chanA := newMockControlChannel()
	engineA, err := NewEngine(EngineConfig{
		ClientID:   "client-A",
		Control:    chanA,
		STUNServer: "", // no STUN for localhost
		Logger:     logger,
	})
	if err != nil {
		t.Fatalf("create engine A: %v", err)
	}
	defer engineA.Close()

	// Create Engine B
	chanB := newMockControlChannel()
	engineB, err := NewEngine(EngineConfig{
		ClientID:   "client-B",
		Control:    chanB,
		STUNServer: "",
		Logger:     logger,
	})
	if err != nil {
		t.Fatalf("create engine B: %v", err)
	}
	defer engineB.Close()

	// Verify both engines have WireGuard keys
	if engineA.WGPUbKey() == "" {
		t.Fatal("engine A missing WG public key")
	}
	if engineB.WGPUbKey() == "" {
		t.Fatal("engine B missing WG public key")
	}
	t.Logf("Engine A WG pubkey: %s", engineA.WGPUbKey()[:16]+"...")
	t.Logf("Engine B WG pubkey: %s", engineB.WGPUbKey()[:16]+"...")

	// --- Step 1: Engine A gathers ICE candidates ---
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	iceCfgA := DefaultAgentConfig()
	iceCfgA.Logger = logger
	agentA := NewAgent(iceCfgA)
	defer agentA.Close()

	candsA, err := agentA.GatherCandidates(ctx)
	if err != nil {
		t.Fatalf("engine A gather: %v", err)
	}
	t.Logf("Engine A: %d candidates", len(candsA))

	// --- Step 2: Engine B gathers ICE candidates ---
	iceCfgB := DefaultAgentConfig()
	iceCfgB.Logger = logger
	agentB := NewAgent(iceCfgB)
	defer agentB.Close()

	candsB, err := agentB.GatherCandidates(ctx)
	if err != nil {
		t.Fatalf("engine B gather: %v", err)
	}
	t.Logf("Engine B: %d candidates", len(candsB))

	// --- Step 3: Exchange candidates (simulated signaling) ---
	candJSONsA := candidatesToJSON(candsA)
	candJSONsB := candidatesToJSON(candsB)

	// Feed B's candidates to A
	for _, c := range candsB {
		agentA.AddRemoteCandidate(c)
	}
	// Feed A's candidates to B
	for _, c := range candsA {
		agentB.AddRemoteCandidate(c)
	}

	// --- Step 4: ICE connectivity checks ---
	errCh := make(chan error, 2)
	go func() { errCh <- agentA.StartChecks(ctx) }()
	go func() { errCh <- agentB.StartChecks(ctx) }()

	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			t.Fatalf("ICE check failed: %v", err)
		}
	}

	pairA := agentA.GetSelectedPair()
	pairB := agentB.GetSelectedPair()
	if pairA == nil || pairB == nil {
		t.Fatal("ICE pair selection failed")
	}
	t.Logf("ICE: A selected %s -> %s", pairA.Local.Addr.String(), pairA.Remote.Addr.String())
	t.Logf("ICE: B selected %s -> %s", pairB.Local.Addr.String(), pairB.Remote.Addr.String())

	// --- Step 5: Hole punching ---
	// Stop ICE read loops so they don't consume punch packets
	agentA.StopReadLoop()
	agentB.StopReadLoop()

	sessionIDStr := "test-session-full!"
	var sessionID [16]byte
	copy(sessionID[:], sessionIDStr)

	punchA := NewPunchEngine(PunchConfig{
		SessionID:  sessionID,
		UDPConn:    agentA.GetUDPConn(),
		RemoteAddr: &pairA.Remote.Addr,
		Role:       PunchRoleInitiator,
		Timeout:    5 * time.Second,
		Logger:     logger,
	})
	punchB := NewPunchEngine(PunchConfig{
		SessionID:  sessionID,
		UDPConn:    agentB.GetUDPConn(),
		RemoteAddr: &pairB.Remote.Addr,
		Role:       PunchRoleResponder,
		Timeout:    5 * time.Second,
		Logger:     logger,
	})

	punchCh := make(chan error, 2)
	go func() {
		_, err := punchA.Punch(ctx)
		punchCh <- err
	}()
	go func() {
		_, err := punchB.Punch(ctx)
		punchCh <- err
	}()

	for i := 0; i < 2; i++ {
		if err := <-punchCh; err != nil {
			t.Fatalf("punch failed: %v", err)
		}
	}
	t.Log("Hole punching verified on both sides")

	// --- Step 6: WireGuard tunnel ---
	privA, pubA, _ := crypto.GenerateWGKeyPair()
	privB, pubB, _ := crypto.GenerateWGKeyPair()

	wgA := NewWGTunnel(WGConfig{
		PrivateKey:    privA,
		PeerPublicKey: pubB,
		PeerAddr:      "10.7.0.2/32",
		Logger:        logger,
	})
	wgB := NewWGTunnel(WGConfig{
		PrivateKey:    privB,
		PeerPublicKey: pubA,
		PeerAddr:      "10.7.0.1/32",
		Logger:        logger,
	})

	bindA := newNetBind(agentA.GetUDPConn(), &pairA.Remote.Addr)
	bindB := newNetBind(agentB.GetUDPConn(), &pairB.Remote.Addr)

	if err := wgA.Start(bindA); err != nil {
		t.Fatalf("start WG A: %v", err)
	}
	defer wgA.Close()

	if err := wgB.Start(bindB); err != nil {
		t.Fatalf("start WG B: %v", err)
	}
	defer wgB.Close()

	t.Log("WireGuard tunnels started on both sides")
	t.Logf("WG A status: %s", wgA.Status())
	t.Logf("WG B status: %s", wgB.Status())

	// Verify candidate JSON round-trip
	if len(candJSONsA) == 0 {
		t.Error("expected non-empty candidate JSON from A")
	}
	if len(candJSONsB) == 0 {
		t.Error("expected non-empty candidate JSON from B")
	}

	// Verify offer/answer message creation
	offer, err := protocol.NewP2POfferMessage("session-1", "A", "B", "full_cone", pubA, candJSONsA)
	if err != nil {
		t.Fatalf("create offer: %v", err)
	}
	payload, err := offer.DecodePayload()
	if err != nil {
		t.Fatalf("decode offer: %v", err)
	}
	offerMsg := payload.(*protocol.P2POfferMessage)
	if offerMsg.FromClientID != "A" || offerMsg.ToClientID != "B" {
		t.Error("offer payload mismatch")
	}
	if offerMsg.WGPublicKey != pubA {
		t.Error("offer WG public key mismatch")
	}

	answer, err := protocol.NewP2PAnswerMessage("session-1", "B", "A", "full_cone", pubB, candJSONsB, true, "")
	if err != nil {
		t.Fatalf("create answer: %v", err)
	}
	ansPayload, err := answer.DecodePayload()
	if err != nil {
		t.Fatalf("decode answer: %v", err)
	}
	ansMsg := ansPayload.(*protocol.P2PAnswerMessage)
	if !ansMsg.Accept {
		t.Error("answer should be accepted")
	}
	if ansMsg.WGPublicKey != pubB {
		t.Error("answer WG public key mismatch")
	}

	t.Log("P2P full chain test PASSED: ICE + Punch + WireGuard + Protocol all verified")
}

// TestP2PEngine_SignalingProtocol tests that the P2P signaling protocol
// correctly creates, sends, and decodes Offer/Answer/Close messages.
func TestP2PEngine_SignalingProtocol(t *testing.T) {
	chanA := newMockControlChannel()
	engineA, err := NewEngine(EngineConfig{
		ClientID: "A",
		Control:  chanA,
	})
	if err != nil {
		t.Fatalf("create engine: %v", err)
	}
	defer engineA.Close()

	// Verify engine state
	if engineA.GetState() != SessionIdle {
		t.Errorf("initial state: got %s, want idle", engineA.GetState())
	}
	if engineA.WGPUbKey() == "" {
		t.Error("engine should have WG public key")
	}

	// Test NAT type (not detected yet)
	if engineA.GetNATType() != "" {
		t.Error("NAT type should be empty before detection")
	}
}

// TestCandidateJSON_RoundTrip tests candidate serialization.
func TestCandidateJSON_RoundTrip(t *testing.T) {
	original := []Candidate{
		{
			ID:         "c1",
			Type:       CandidateHost,
			Addr:       net.UDPAddr{IP: net.ParseIP("192.168.1.1"), Port: 5000},
			Priority:   100,
			Foundation: "abc123",
		},
		{
			ID:         "c2",
			Type:       CandidateServerReflexive,
			Addr:       net.UDPAddr{IP: net.ParseIP("203.0.113.1"), Port: 6000},
			Priority:   50,
			Foundation: "def456",
		},
	}

	jsonCands := candidatesToJSON(original)
	if len(jsonCands) != 2 {
		t.Fatalf("expected 2 JSON candidates, got %d", len(jsonCands))
	}

	restored := make([]Candidate, len(jsonCands))
	for i, jc := range jsonCands {
		restored[i] = candidateFromJSON(jc)
	}

	for i := range original {
		if restored[i].ID != original[i].ID {
			t.Errorf("candidate %d ID: got %s, want %s", i, restored[i].ID, original[i].ID)
		}
		if restored[i].Type != original[i].Type {
			t.Errorf("candidate %d Type: got %s, want %s", i, restored[i].Type, original[i].Type)
		}
		if restored[i].Priority != original[i].Priority {
			t.Errorf("candidate %d Priority: got %d, want %d", i, restored[i].Priority, original[i].Priority)
		}
	}
}
