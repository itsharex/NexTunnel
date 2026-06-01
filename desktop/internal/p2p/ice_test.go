package p2p

import (
	"context"
	"net"
	"testing"
	"time"
)

func TestComputePriority(t *testing.T) {
	hostP := computePriority(CandidateHost, 65535, 1)
	srflxP := computePriority(CandidateServerReflexive, 65535, 1)
	relayP := computePriority(CandidateRelay, 65535, 1)

	if hostP <= srflxP {
		t.Errorf("host priority (%d) should be > srflx priority (%d)", hostP, srflxP)
	}
	if srflxP <= relayP {
		t.Errorf("srflx priority (%d) should be > relay priority (%d)", srflxP, relayP)
	}
}

func TestComputePairPriority(t *testing.T) {
	p1 := computePairPriority(100, 200)
	p2 := computePairPriority(200, 200)
	p3 := computePairPriority(300, 200)

	if p2 <= p1 {
		t.Errorf("equal priorities should rank higher than asymmetric")
	}
	if p3 <= p2 {
		t.Errorf("higher controlling priority should rank higher")
	}
}

func TestComputeFoundation(t *testing.T) {
	f1 := computeFoundation(CandidateHost, "192.168.1.1", "")
	f2 := computeFoundation(CandidateHost, "192.168.1.1", "")
	f3 := computeFoundation(CandidateHost, "10.0.0.1", "")

	if f1 != f2 {
		t.Error("same inputs should produce same foundation")
	}
	if f1 == f3 {
		t.Error("different inputs should produce different foundation")
	}
}

func TestAgent_GatherHostCandidates(t *testing.T) {
	cfg := DefaultAgentConfig()
	agent := NewAgent(cfg)
	defer agent.Close()

	ctx := context.Background()
	candidates, err := agent.GatherCandidates(ctx)
	if err != nil {
		t.Fatalf("GatherCandidates failed: %v", err)
	}

	if len(candidates) == 0 {
		t.Error("expected at least one host candidate")
	}

	for _, c := range candidates {
		if c.Type != CandidateHost {
			t.Errorf("expected host candidate, got %s", c.Type)
		}
		if c.Addr.Port == 0 {
			t.Error("candidate port should not be 0")
		}
		if c.Priority == 0 {
			t.Error("candidate priority should not be 0")
		}
		if c.ID == "" {
			t.Error("candidate ID should not be empty")
		}
	}

	if agent.GetUDPConn() == nil {
		t.Error("UDP socket should be bound after gathering")
	}
}

func TestAgent_LocalConnectivity(t *testing.T) {
	// Two agents on localhost should find host->host candidates
	cfgA := DefaultAgentConfig()
	cfgA.Role = ICERoleControlling
	agentA := NewAgent(cfgA)
	defer agentA.Close()

	cfgB := DefaultAgentConfig()
	cfgB.Role = ICERoleControlled
	agentB := NewAgent(cfgB)
	defer agentB.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Gather candidates for both agents
	candsA, err := agentA.GatherCandidates(ctx)
	if err != nil {
		t.Fatalf("Agent A gather failed: %v", err)
	}
	t.Logf("Agent A candidates: %d", len(candsA))

	candsB, err := agentB.GatherCandidates(ctx)
	if err != nil {
		t.Fatalf("Agent B gather failed: %v", err)
	}
	t.Logf("Agent B candidates: %d", len(candsB))

	// Exchange candidates (simulate signaling)
	for _, c := range candsB {
		agentA.AddRemoteCandidate(c)
	}
	for _, c := range candsA {
		agentB.AddRemoteCandidate(c)
	}

	// Start connectivity checks on both sides
	errCh := make(chan error, 2)
	go func() { errCh <- agentA.StartChecks(ctx) }()
	go func() { errCh <- agentB.StartChecks(ctx) }()

	// Wait for both agents to complete
	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			t.Fatalf("Agent check failed: %v", err)
		}
	}

	// Both agents should have selected pairs
	pairA := agentA.GetSelectedPair()
	pairB := agentB.GetSelectedPair()

	if pairA == nil {
		t.Fatal("Agent A should have a selected pair")
	}
	if pairB == nil {
		t.Fatal("Agent B should have a selected pair")
	}

	t.Logf("Agent A selected: local=%s remote=%s", pairA.Local.Addr.String(), pairA.Remote.Addr.String())
	t.Logf("Agent B selected: local=%s remote=%s", pairB.Local.Addr.String(), pairB.Remote.Addr.String())

	if agentA.GetState() != AgentStateConnected {
		t.Errorf("Agent A state: got %s, want connected", agentA.GetState())
	}
	if agentB.GetState() != AgentStateConnected {
		t.Errorf("Agent B state: got %s, want connected", agentB.GetState())
	}
}

func TestAgent_AddRemoteCandidate(t *testing.T) {
	agent := NewAgent(DefaultAgentConfig())
	defer agent.Close()

	// Must bind UDP first
	conn, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4zero, Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	agent.udpConn = conn
	agent.ctx, agent.cancel = context.WithCancel(context.Background())

	// Add a local candidate first
	localC := Candidate{
		ID:       "local-1",
		Type:     CandidateHost,
		Addr:     net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345},
		Priority: computePriority(CandidateHost, 65535, 1),
	}
	agent.addLocalCandidate(localC)

	// Add remote candidate
	remoteC := Candidate{
		ID:       "remote-1",
		Type:     CandidateHost,
		Addr:     net.UDPAddr{IP: net.ParseIP("127.0.0.1"), Port: 54321},
		Priority: computePriority(CandidateHost, 65535, 1),
	}
	agent.AddRemoteCandidate(remoteC)

	agent.checklistMu.Lock()
	count := len(agent.checklist)
	agent.checklistMu.Unlock()

	if count != 1 {
		t.Errorf("expected 1 pair, got %d", count)
	}
}
