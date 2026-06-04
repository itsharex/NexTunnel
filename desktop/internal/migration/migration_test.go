package migration

import (
	"context"
	"net"
	"sync"
	"testing"
	"time"
)

// mockDetector is a test detector that can fire events on demand.
type mockDetector struct {
	events  chan NetworkEvent
	ifaces  []net.Interface
	started bool
}

func newMockDetector() *mockDetector {
	return &mockDetector{
		events: make(chan NetworkEvent, 16),
	}
}

func (d *mockDetector) Start(ctx context.Context) error {
	d.started = true
	return nil
}

func (d *mockDetector) Stop() {
	d.started = false
}

func (d *mockDetector) Events() <-chan NetworkEvent {
	return d.events
}

func (d *mockDetector) CurrentInterfaces() ([]net.Interface, error) {
	return d.ifaces, nil
}

func (d *mockDetector) FireEvent(evt NetworkEvent) {
	d.events <- evt
}

func TestMigrator_InterfaceChange(t *testing.T) {
	det := newMockDetector()
	cfg := DefaultMigrationConfig()

	var mu sync.Mutex
	var events []MigrationState

	m := NewMigrator(cfg, det)
	m.OnMigration(func(state MigrationState, evt NetworkEvent) {
		mu.Lock()
		events = append(events, state)
		mu.Unlock()
	})

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := m.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	if m.State() != MigrationDetecting {
		t.Errorf("state = %v, want detecting", m.State())
	}

	// Fire an address change event
	det.FireEvent(NetworkEvent{
		Type:      "address_changed",
		Interface: "eth0",
		OldAddr:   "192.168.1.100/24",
		NewAddr:   "10.0.0.50/24",
		Timestamp: time.Now(),
	})

	// Wait for event processing
	time.Sleep(200 * time.Millisecond)

	mu.Lock()
	t.Logf("Migration events: %v", events)
	mu.Unlock()

	// Should have gone through Migrating -> Success -> Detecting
	if m.State() != MigrationDetecting {
		t.Errorf("final state = %v, want detecting", m.State())
	}

	cancel()
	m.Stop()

	if m.State() != MigrationIdle {
		t.Errorf("stopped state = %v, want idle", m.State())
	}
}

func TestMigrator_Timeout(t *testing.T) {
	det := newMockDetector()
	cfg := DefaultMigrationConfig()
	cfg.MigrationTimeout = 100 * time.Millisecond

	m := NewMigrator(cfg, det)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	m.Start(ctx)

	// Fire event
	det.FireEvent(NetworkEvent{
		Type:      "interface_removed",
		Interface: "wlan0",
		OldAddr:   "192.168.1.100/24",
		Timestamp: time.Now(),
	})

	time.Sleep(200 * time.Millisecond)

	cancel()
	m.Stop()
}

func TestMigrator_PacketLoss(t *testing.T) {
	det := newMockDetector()
	cfg := DefaultMigrationConfig()
	cfg.MaxPacketLoss = 5

	m := NewMigrator(cfg, det)
	if m.config.MaxPacketLoss != 5 {
		t.Errorf("MaxPacketLoss = %d, want 5", m.config.MaxPacketLoss)
	}
}

func TestNetworkDetector_Poll(t *testing.T) {
	det := NewPollingDetector(100 * time.Millisecond)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := det.Start(ctx); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// List current interfaces
	ifaces, err := det.CurrentInterfaces()
	if err != nil {
		t.Fatalf("CurrentInterfaces: %v", err)
	}

	t.Logf("Current interfaces: %d", len(ifaces))
	for _, iface := range ifaces {
		if iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		t.Logf("  %s: flags=%v", iface.Name, iface.Flags)
	}

	// Wait a bit for polling
	time.Sleep(300 * time.Millisecond)

	cancel()
	det.Stop()
}

func TestMigrationStates(t *testing.T) {
	states := []MigrationState{
		MigrationIdle,
		MigrationDetecting,
		MigrationMigrating,
		MigrationSuccess,
		MigrationFailed,
	}

	for _, s := range states {
		if s == "" {
			t.Error("empty state")
		}
		t.Logf("State: %s", s)
	}
}
