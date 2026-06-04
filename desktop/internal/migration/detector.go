package migration

import (
	"context"
	"net"
	"sync"
	"time"
)

// NetworkDetector detects changes in network interfaces.
type NetworkDetector interface {
	Start(ctx context.Context) error
	Stop()
	Events() <-chan NetworkEvent
	CurrentInterfaces() ([]net.Interface, error)
}

// PollingDetector polls net.Interfaces() at regular intervals.
type PollingDetector struct {
	interval time.Duration
	events   chan NetworkEvent
	known    map[string]string // iface name -> addr

	mu     sync.Mutex
	ctx    context.Context
	cancel context.CancelFunc
}

// NewPollingDetector creates a polling-based network detector.
func NewPollingDetector(interval time.Duration) *PollingDetector {
	return &PollingDetector{
		interval: interval,
		events:   make(chan NetworkEvent, 16),
		known:    make(map[string]string),
	}
}

// Start begins polling for network changes.
func (d *PollingDetector) Start(ctx context.Context) error {
	d.ctx, d.cancel = context.WithCancel(ctx)
	d.snapshot() // initial snapshot
	go d.pollLoop()
	return nil
}

// Stop stops the detector.
func (d *PollingDetector) Stop() {
	if d.cancel != nil {
		d.cancel()
	}
}

// Events returns the channel of network events.
func (d *PollingDetector) Events() <-chan NetworkEvent {
	return d.events
}

// CurrentInterfaces returns the current network interfaces.
func (d *PollingDetector) CurrentInterfaces() ([]net.Interface, error) {
	return net.Interfaces()
}

func (d *PollingDetector) pollLoop() {
	ticker := time.NewTicker(d.interval)
	defer ticker.Stop()

	for {
		select {
		case <-d.ctx.Done():
			return
		case <-ticker.C:
			d.detectChanges()
		}
	}
}

func (d *PollingDetector) detectChanges() {
	ifaces, err := net.Interfaces()
	if err != nil {
		return
	}

	current := make(map[string]string)
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}
		current[iface.Name] = addrs[0].String()
	}

	d.mu.Lock()
	defer d.mu.Unlock()

	// Detect new or changed interfaces
	for name, addr := range current {
		oldAddr, exists := d.known[name]
		if !exists {
			d.emitEvent(NetworkEvent{
				Type:      "interface_added",
				Interface: name,
				NewAddr:   addr,
				Timestamp: time.Now(),
			})
		} else if oldAddr != addr {
			d.emitEvent(NetworkEvent{
				Type:      "address_changed",
				Interface: name,
				OldAddr:   oldAddr,
				NewAddr:   addr,
				Timestamp: time.Now(),
			})
		}
	}

	// Detect removed interfaces
	for name, addr := range d.known {
		if _, exists := current[name]; !exists {
			d.emitEvent(NetworkEvent{
				Type:      "interface_removed",
				Interface: name,
				OldAddr:   addr,
				Timestamp: time.Now(),
			})
		}
	}

	d.known = current
}

func (d *PollingDetector) snapshot() {
	ifaces, err := net.Interfaces()
	if err != nil {
		return
	}
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 || iface.Flags&net.FlagLoopback != 0 {
			continue
		}
		addrs, err := iface.Addrs()
		if err != nil || len(addrs) == 0 {
			continue
		}
		d.known[iface.Name] = addrs[0].String()
	}
}

func (d *PollingDetector) emitEvent(evt NetworkEvent) {
	select {
	case d.events <- evt:
	default:
	}
}
