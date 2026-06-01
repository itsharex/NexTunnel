package p2p

import (
	"context"
	"log/slog"
	"net"
	"testing"
	"time"
)

func TestBuildParsePunchPacket(t *testing.T) {
	var sessionID [16]byte
	copy(sessionID[:], "test-session-id!")
	nonce := uint64(12345678)

	pkt := buildPunchPacket(sessionID, punchFlagSYN, nonce)
	if len(pkt) != punchPacketSize {
		t.Errorf("packet size: got %d, want %d", len(pkt), punchPacketSize)
	}

	parsedID, flags, parsedNonce, ok := ParsePunchPacket(pkt)
	if !ok {
		t.Fatal("ParsePunchPacket failed")
	}
	if parsedID != sessionID {
		t.Error("session ID mismatch")
	}
	if flags != punchFlagSYN {
		t.Errorf("flags: got %d, want %d", flags, punchFlagSYN)
	}
	if parsedNonce != nonce {
		t.Errorf("nonce: got %d, want %d", parsedNonce, nonce)
	}
}

func TestParsePunchPacket_Invalid(t *testing.T) {
	// Too short
	_, _, _, ok := ParsePunchPacket([]byte{0x4E, 0x50})
	if ok {
		t.Error("expected failure for short packet")
	}

	// Wrong magic
	pkt := make([]byte, punchPacketSize)
	pkt[0] = 0xFF
	pkt[1] = 0xFF
	_, _, _, ok = ParsePunchPacket(pkt)
	if ok {
		t.Error("expected failure for wrong magic")
	}
}

func TestPunchEngine_Localhost(t *testing.T) {
	// Two punch engines on localhost should succeed
	connA, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer connA.Close()

	connB, err := net.ListenUDP("udp", &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: 0})
	if err != nil {
		t.Fatal(err)
	}
	defer connB.Close()

	var sessionID [16]byte
	copy(sessionID[:], "test-punch-1234!")

	addrA := connA.LocalAddr().(*net.UDPAddr)
	addrB := connB.LocalAddr().(*net.UDPAddr)

	engineA := NewPunchEngine(PunchConfig{
		SessionID:  sessionID,
		UDPConn:    connA,
		RemoteAddr: addrB,
		Role:       PunchRoleInitiator,
		Timeout:    5 * time.Second,
		Logger:     slog.Default(),
	})

	engineB := NewPunchEngine(PunchConfig{
		SessionID:  sessionID,
		UDPConn:    connB,
		RemoteAddr: addrA,
		Role:       PunchRoleResponder,
		Timeout:    5 * time.Second,
		Logger:     slog.Default(),
	})

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	errCh := make(chan error, 2)
	resultCh := make(chan *PunchResult, 2)

	go func() {
		r, err := engineA.Punch(ctx)
		resultCh <- r
		errCh <- err
	}()
	go func() {
		r, err := engineB.Punch(ctx)
		resultCh <- r
		errCh <- err
	}()

	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			t.Fatalf("Punch failed: %v", err)
		}
	}

	for i := 0; i < 2; i++ {
		r := <-resultCh
		if r == nil || !r.Success {
			t.Error("expected successful punch result")
		}
	}

	if engineA.GetState() != PunchStateVerified {
		t.Errorf("engine A state: got %s, want verified", engineA.GetState())
	}
	if engineB.GetState() != PunchStateVerified {
		t.Errorf("engine B state: got %s, want verified", engineB.GetState())
	}
}
