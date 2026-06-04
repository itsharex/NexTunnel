package relay

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// RelayManager manages multiple relay clients and selects the best one.
type RelayManager struct {
	config  RelayManagerConfig
	clients sync.Map // serverAddr -> *RelayClient
	active  atomic.Value // *RelayClient

	ctx    context.Context
	cancel context.CancelFunc
	logger *slog.Logger
}

// NewRelayManager creates a new relay manager.
func NewRelayManager(cfg RelayManagerConfig, opts ...RelayOption) *RelayManager {
	for _, o := range opts {
		o(&cfg)
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}

	ctx, cancel := context.WithCancel(context.Background())
	m := &RelayManager{
		config: cfg,
		ctx:    ctx,
		cancel: cancel,
		logger: cfg.Logger,
	}
	return m
}

// Start connects to all configured relay servers and begins probing.
func (m *RelayManager) Start(ctx context.Context) error {
	m.ctx, m.cancel = context.WithCancel(ctx)

	for _, relayCfg := range m.config.Relays {
		client := NewRelayClient(relayCfg)
		m.clients.Store(relayCfg.ServerAddr, client)

		go func(c *RelayClient) {
			if err := c.Connect(m.ctx); err != nil {
				m.logger.Warn("relay connect failed", "addr", c.ServerAddr(), "error", err)
			}
		}(client)
	}

	// Start probing loop
	go m.probeLoop()

	m.logger.Info("relay manager started", "relays", len(m.config.Relays))
	return nil
}

// Stop disconnects all relay clients.
func (m *RelayManager) Stop() {
	m.cancel()
	m.clients.Range(func(_, value any) bool {
		value.(*RelayClient).Close()
		return true
	})
	m.logger.Info("relay manager stopped")
}

// ActiveRelay returns the currently active relay client, or nil.
func (m *RelayManager) ActiveRelay() *RelayClient {
	v := m.active.Load()
	if v == nil {
		return nil
	}
	return v.(*RelayClient)
}

// AllRelays returns all relay clients.
func (m *RelayManager) AllRelays() []*RelayClient {
	var result []*RelayClient
	m.clients.Range(func(_, value any) bool {
		result = append(result, value.(*RelayClient))
		return true
	})
	return result
}

// SwitchTo switches the active relay to the given server address.
func (m *RelayManager) SwitchTo(serverAddr string) error {
	v, ok := m.clients.Load(serverAddr)
	if !ok {
		return fmt.Errorf("relay not found: %s", serverAddr)
	}
	client := v.(*RelayClient)
	if !client.IsConnected() {
		return fmt.Errorf("relay not connected: %s", serverAddr)
	}
	m.active.Store(client)
	m.logger.Info("active relay switched", "addr", serverAddr, "latency", client.Latency())
	return nil
}

// SelectBest returns the connected relay with the lowest latency.
func (m *RelayManager) SelectBest() *RelayClient {
	var candidates []*RelayClient
	m.clients.Range(func(_, value any) bool {
		c := value.(*RelayClient)
		if c.IsConnected() {
			candidates = append(candidates, c)
		}
		return true
	})

	if len(candidates) == 0 {
		return nil
	}

	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].Latency() < candidates[j].Latency()
	})

	return candidates[0]
}

// probeLoop periodically probes all relays and selects the best.
func (m *RelayManager) probeLoop() {
	ticker := time.NewTicker(m.config.ProbeInterval)
	defer ticker.Stop()

	// Initial selection
	time.Sleep(500 * time.Millisecond) // wait for connections
	if best := m.SelectBest(); best != nil {
		m.active.Store(best)
		m.logger.Info("initial relay selected", "addr", best.ServerAddr(), "latency", best.Latency())
	}

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.evaluateRelays()
		}
	}
}

// evaluateRelays checks all relays and switches if a better one is found.
func (m *RelayManager) evaluateRelays() {
	best := m.SelectBest()
	if best == nil {
		return
	}

	active := m.ActiveRelay()
	if active == nil || best.ServerAddr() != active.ServerAddr() {
		m.active.Store(best)
		m.logger.Info("relay auto-switched",
			"to", best.ServerAddr(),
			"latency", best.Latency())
	}
}
