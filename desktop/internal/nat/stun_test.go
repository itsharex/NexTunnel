package nat

import (
	"log/slog"
	"testing"
	"time"
)

func TestNewClient(t *testing.T) {
	c := NewClient()
	if c.timeout != 3*time.Second {
		t.Errorf("default timeout: got %v, want 3s", c.timeout)
	}
	if c.retries != 3 {
		t.Errorf("default retries: got %d, want 3", c.retries)
	}
}

func TestNewClient_WithOptions(t *testing.T) {
	logger := slog.Default()
	c := NewClient(
		WithTimeout(5*time.Second),
		WithRetries(5),
		WithLogger(logger),
	)
	if c.timeout != 5*time.Second {
		t.Errorf("timeout: got %v, want 5s", c.timeout)
	}
	if c.retries != 5 {
		t.Errorf("retries: got %d, want 5", c.retries)
	}
}

func TestNATType_String(t *testing.T) {
	tests := []struct {
		nat  NATType
		want string
	}{
		{NATOpenInternet, "Open Internet (no NAT)"},
		{NATFullCone, "Full Cone NAT"},
		{NATRestricted, "Restricted Cone NAT"},
		{NATPortRestricted, "Port Restricted Cone NAT"},
		{NATSymmetric, "Symmetric NAT"},
		{NATBlocked, "Blocked (UDP not available)"},
	}
	for _, tt := range tests {
		got := tt.nat.String()
		if got != tt.want {
			t.Errorf("NATType(%q).String() = %q, want %q", tt.nat, got, tt.want)
		}
	}
}

func TestNATResult_IsP2PPossible(t *testing.T) {
	tests := []struct {
		name    string
		local   NATType
		peer    NATType
		want    bool
	}{
		{"both full cone", NATFullCone, NATFullCone, true},
		{"full cone + restricted", NATFullCone, NATRestricted, true},
		{"symmetric + full cone", NATSymmetric, NATFullCone, true},
		{"symmetric + symmetric", NATSymmetric, NATSymmetric, false},
		{"blocked + anything", NATBlocked, NATFullCone, false},
		{"anything + blocked", NATFullCone, NATBlocked, false},
		{"restricted + port restricted", NATRestricted, NATPortRestricted, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := &NATResult{Type: tt.local}
			got := result.IsP2PPossible(tt.peer)
			if got != tt.want {
				t.Errorf("IsP2PPossible(%q) = %v, want %v", tt.peer, got, tt.want)
			}
		})
	}
}
