package p2p

import (
	"encoding/json"
	"net"
	"testing"
)

func TestCandidateExchangeJSONRoundTrip(t *testing.T) {
	exchange := CandidateExchange{
		SessionID: "0123456789abcdef0123456789abcdef",
		Role:      "initiator",
		Candidates: []Candidate{{
			ID:       "candidate-1",
			Type:     CandidateHost,
			Addr:     net.UDPAddr{IP: net.ParseIP("10.160.166.10"), Port: 49152},
			Priority: 100,
		}},
	}

	payload, err := json.Marshal(exchange)
	if err != nil {
		t.Fatalf("marshal exchange: %v", err)
	}
	var decoded CandidateExchange
	if err := json.Unmarshal(payload, &decoded); err != nil {
		t.Fatalf("unmarshal exchange: %v", err)
	}

	if decoded.SessionID != exchange.SessionID {
		t.Fatalf("session id mismatch: %s", decoded.SessionID)
	}
	if len(decoded.Candidates) != 1 {
		t.Fatalf("expected one candidate, got %d", len(decoded.Candidates))
	}
	if decoded.Candidates[0].Addr.String() != "10.160.166.10:49152" {
		t.Fatalf("candidate addr mismatch: %s", decoded.Candidates[0].Addr.String())
	}
}

func TestIsLANCandidate(t *testing.T) {
	tests := []struct {
		name string
		addr string
		want bool
	}{
		{name: "rfc1918 with port", addr: "10.160.166.44:19091", want: true},
		{name: "rfc1918 bare", addr: "192.168.1.10", want: true},
		{name: "public", addr: "8.8.8.8:53", want: false},
		{name: "invalid", addr: "not-an-ip", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsLANCandidate(tt.addr); got != tt.want {
				t.Fatalf("IsLANCandidate(%q) = %t, want %t", tt.addr, got, tt.want)
			}
		})
	}
}

func TestVerifySessionIDHandlesInvalidInput(t *testing.T) {
	if got := verifySessionID("not-hex"); got != ([16]byte{}) {
		t.Fatalf("invalid session id should produce zero value, got %x", got)
	}

	value := verifySessionID("010203")
	if value[0] != 1 || value[1] != 2 || value[2] != 3 {
		t.Fatalf("short session id was not copied correctly: %x", value)
	}
}

func TestDirectVerifySessionIDUsesInitiatorID(t *testing.T) {
	local := CandidateExchange{SessionID: "local-session"}
	remote := CandidateExchange{SessionID: "remote-session"}

	if got := directVerifySessionID("initiator", local, remote); got != "local-session" {
		t.Fatalf("initiator session id = %q", got)
	}
	if got := directVerifySessionID("responder", local, remote); got != "remote-session" {
		t.Fatalf("responder session id = %q", got)
	}
}
