package nat

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"testing"
)

// mockSTUNClient implements STUNClient for testing the detection decision tree.
type mockSTUNClient struct {
	// binding maps server address to the binding response it should return
	bindings map[string]*STUNBinding
	// bindingErrors maps server address to an error to return
	bindingErrors map[string]error
	// altBindings maps server address to the binding response for BindingRequestFromAlt
	altBindings map[string]*STUNBinding
	altErrors   map[string]error
}

func newMockSTUN() *mockSTUNClient {
	return &mockSTUNClient{
		bindings:      make(map[string]*STUNBinding),
		bindingErrors: make(map[string]error),
		altBindings:   make(map[string]*STUNBinding),
		altErrors:     make(map[string]error),
	}
}

func (m *mockSTUNClient) BindingRequest(_ context.Context, serverAddr string, _ *net.UDPConn) (*STUNBinding, error) {
	if err, ok := m.bindingErrors[serverAddr]; ok {
		return nil, err
	}
	if b, ok := m.bindings[serverAddr]; ok {
		return b, nil
	}
	return nil, fmt.Errorf("no mock binding for %s", serverAddr)
}

func (m *mockSTUNClient) BindingRequestFromAlt(_ context.Context, serverAddr string, _ *net.UDPConn) (*STUNBinding, error) {
	if err, ok := m.altErrors[serverAddr]; ok {
		return nil, err
	}
	if b, ok := m.altBindings[serverAddr]; ok {
		return b, nil
	}
	return nil, fmt.Errorf("no mock alt binding for %s", serverAddr)
}

func TestDetect_Blocked(t *testing.T) {
	mock := newMockSTUN()
	// No bindings configured -> STUN requests fail -> Blocked
	d := NewDetector("1.1.1.1:3478", "2.2.2.2:3478", mock, slog.Default())
	result, err := d.Detect(context.Background())
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if result.Type != NATBlocked {
		t.Errorf("expected Blocked, got %s", result.Type)
	}
}

func TestDetect_FullCone(t *testing.T) {
	mock := newMockSTUN()
	primary := "1.1.1.1:3478"
	alt := "2.2.2.2:3478"

	// Test I succeeds: mapped address is different from local (behind NAT)
	mock.bindings[primary] = &STUNBinding{
		MappedAddr: net.UDPAddr{IP: net.ParseIP("203.0.113.1"), Port: 50000},
	}
	// Test II (alt) succeeds -> Full Cone
	mock.altBindings[primary] = &STUNBinding{
		MappedAddr: net.UDPAddr{IP: net.ParseIP("203.0.113.1"), Port: 50000},
	}
	// Test III (alt server direct) - needed but won't be reached for Full Cone
	mock.bindings[alt] = &STUNBinding{
		MappedAddr: net.UDPAddr{IP: net.ParseIP("203.0.113.1"), Port: 50000},
	}

	d := NewDetector(primary, alt, mock, slog.Default())
	result, err := d.Detect(context.Background())
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if result.Type != NATFullCone {
		t.Errorf("expected Full Cone, got %s", result.Type)
	}
	if result.PublicAddr != "203.0.113.1:50000" {
		t.Errorf("unexpected public addr: %s", result.PublicAddr)
	}
}

func TestDetect_Symmetric(t *testing.T) {
	mock := newMockSTUN()
	primary := "1.1.1.1:3478"
	alt := "2.2.2.2:3478"

	// Test I succeeds with one mapped address
	mock.bindings[primary] = &STUNBinding{
		MappedAddr: net.UDPAddr{IP: net.ParseIP("203.0.113.1"), Port: 50000},
	}
	// Test II (alt) fails -> not Full Cone
	mock.altErrors[primary] = fmt.Errorf("no alt response")
	// Test III: different mapped address from alternate server -> Symmetric
	mock.bindings[alt] = &STUNBinding{
		MappedAddr: net.UDPAddr{IP: net.ParseIP("203.0.113.1"), Port: 50001},
	}

	d := NewDetector(primary, alt, mock, slog.Default())
	result, err := d.Detect(context.Background())
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if result.Type != NATSymmetric {
		t.Errorf("expected Symmetric, got %s", result.Type)
	}
}

func TestDetect_Restricted(t *testing.T) {
	mock := newMockSTUN()
	primary := "1.1.1.1:3478"
	alt := "2.2.2.2:3478"

	// Test I succeeds
	mock.bindings[primary] = &STUNBinding{
		MappedAddr: net.UDPAddr{IP: net.ParseIP("203.0.113.1"), Port: 50000},
	}
	// Test II (alt) fails -> not Full Cone
	mock.altErrors[primary] = fmt.Errorf("no alt response")
	// Test III: SAME mapped address from alternate server -> Restricted
	mock.bindings[alt] = &STUNBinding{
		MappedAddr: net.UDPAddr{IP: net.ParseIP("203.0.113.1"), Port: 50000},
	}

	d := NewDetector(primary, alt, mock, slog.Default())
	result, err := d.Detect(context.Background())
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if result.Type != NATRestricted {
		t.Errorf("expected Restricted, got %s", result.Type)
	}
}

func TestDetect_PortRestricted(t *testing.T) {
	mock := newMockSTUN()
	primary := "1.1.1.1:3478"
	alt := "2.2.2.2:3478"

	// Test I succeeds
	mock.bindings[primary] = &STUNBinding{
		MappedAddr: net.UDPAddr{IP: net.ParseIP("203.0.113.1"), Port: 50000},
	}
	// Test II (alt) fails -> not Full Cone
	mock.altErrors[primary] = fmt.Errorf("no alt response")
	// Test III fails: can't reach alternate server -> assume Port Restricted
	mock.bindingErrors[alt] = fmt.Errorf("can't reach alt server")

	d := NewDetector(primary, alt, mock, slog.Default())
	result, err := d.Detect(context.Background())
	if err != nil {
		t.Fatalf("Detect failed: %v", err)
	}
	if result.Type != NATPortRestricted {
		t.Errorf("expected Port Restricted, got %s", result.Type)
	}
}
