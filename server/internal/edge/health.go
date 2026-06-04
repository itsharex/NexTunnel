package edge

import (
	"context"
	"fmt"
	"net"
	"sync"
	"time"
)

// HealthChecker monitors edge node health via periodic probes.
type HealthChecker struct {
	config   EdgeConfig
	registry *Registry
	mu       sync.Mutex
	running  bool
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// NewHealthChecker creates a new health checker.
func NewHealthChecker(cfg EdgeConfig, reg *Registry) *HealthChecker {
	return &HealthChecker{
		config:   cfg,
		registry: reg,
	}
}

// Start begins periodic health checking.
func (h *HealthChecker) Start(ctx context.Context) {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return
	}
	h.running = true
	ctx, h.cancel = context.WithCancel(ctx)
	h.mu.Unlock()

	h.config.Logger.Info("health checker started", "interval", h.config.HeartbeatInterval)

	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		ticker := time.NewTicker(h.config.HeartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				h.config.Logger.Info("health checker stopped")
				return
			case <-ticker.C:
				h.probeAll(ctx)
			}
		}
	}()
}

// Stop halts health checking.
func (h *HealthChecker) Stop() {
	h.mu.Lock()
	if !h.running {
		h.mu.Unlock()
		return
	}
	h.running = false
	h.cancel()
	h.mu.Unlock()
	h.wg.Wait()
}

// probeAll checks all registered nodes concurrently.
func (h *HealthChecker) probeAll(ctx context.Context) {
	nodes := h.registry.List()
	if len(nodes) == 0 {
		return
	}

	sem := make(chan struct{}, h.config.ProbeConcurrency)
	var wg sync.WaitGroup

	for _, node := range nodes {
		wg.Add(1)
		sem <- struct{}{}
		go func(n *EdgeNode) {
			defer wg.Done()
			defer func() { <-sem }()
			h.probeNode(ctx, n)
		}(node)
	}

	wg.Wait()
}

// probeNode checks a single node's health.
func (h *HealthChecker) probeNode(ctx context.Context, node *EdgeNode) {
	probeCtx, cancel := context.WithTimeout(ctx, h.config.HealthCheckTimeout)
	defer cancel()

	start := time.Now()
	err := h.tcpProbe(probeCtx, node.Addr)
	latency := time.Since(start)

	if err != nil {
		node.RecordFailure()
		fails := node.ConsecutiveFails()

		if fails >= h.config.UnhealthyThreshold && node.GetStatus() == StatusHealthy {
			node.SetStatus(StatusUnhealthy)
			node.SetUnhealthySince(time.Now())
			h.config.Logger.Warn("node marked unhealthy",
				"id", node.ID, "fails", fails, "region", node.Region)
		}

		// Check for auto-deregistration
		if node.GetStatus() == StatusUnhealthy {
			since := node.UnhealthySince()
			if !since.IsZero() && time.Since(since) > h.config.DeregisterTimeout {
				h.config.Logger.Warn("auto-deregistering unhealthy node",
					"id", node.ID, "unhealthy_duration", time.Since(since))
				node.SetStatus(StatusOffline)
				_ = h.registry.Deregister(node.ID)
			}
		}
	} else {
		node.RecordSuccess(latency)
		okCount := node.ConsecutiveOKs()

		if node.GetStatus() == StatusUnhealthy && okCount >= h.config.HealthyThreshold {
			node.SetStatus(StatusHealthy)
			node.SetUnhealthySince(time.Time{})
			h.config.Logger.Info("node recovered to healthy",
				"id", node.ID, "latency", latency, "region", node.Region)
		}
	}
}

// tcpProbe attempts a TCP connection to the node's address.
func (h *HealthChecker) tcpProbe(ctx context.Context, addr string) error {
	d := net.Dialer{Timeout: h.config.HealthCheckTimeout}
	conn, err := d.DialContext(ctx, "tcp", addr)
	if err != nil {
		return fmt.Errorf("tcp probe failed: %w", err)
	}
	conn.Close()
	return nil
}
