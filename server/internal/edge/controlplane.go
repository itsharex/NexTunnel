package edge

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// ControlPlaneClient communicates with the NexTunnel Control Plane HTTP API
// to register edge nodes and send heartbeats.
type ControlPlaneClient struct {
	config     ControlPlaneConfig
	httpClient *http.Client
	mu         sync.Mutex
	heartbeats map[string]context.CancelFunc // nodeID -> cancel
	logger     *slog.Logger
}

// ControlPlaneConfig configures the Control Plane client.
type ControlPlaneConfig struct {
	// BaseURL is the Control Plane HTTP API address (e.g. "http://localhost:8081").
	BaseURL string

	// BearerToken is the optional authentication token for the Control Plane.
	BearerToken string

	// HeartbeatInterval is how often to send heartbeats for registered nodes.
	HeartbeatInterval time.Duration

	// Timeout is the HTTP request timeout.
	Timeout time.Duration

	Logger *slog.Logger
}

// DefaultControlPlaneConfig returns sensible defaults.
func DefaultControlPlaneConfig() ControlPlaneConfig {
	return ControlPlaneConfig{
		BaseURL:           "http://localhost:8081",
		HeartbeatInterval: 10 * time.Second,
		Timeout:           5 * time.Second,
		Logger:            slog.Default(),
	}
}

// NewControlPlaneClient creates a new Control Plane client.
func NewControlPlaneClient(cfg ControlPlaneConfig) *ControlPlaneClient {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	return &ControlPlaneClient{
		config: cfg,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		heartbeats: make(map[string]context.CancelFunc),
		logger:     cfg.Logger,
	}
}

// nodeRegistration is the payload for POST /api/v1/nodes.
type nodeRegistration struct {
	ID       string            `json:"id"`
	Addr     string            `json:"addr"`
	Region   string            `json:"region"`
	Role     string            `json:"role"`
	Tags     map[string]string `json:"tags,omitempty"`
	Capacity int               `json:"capacity"`
}

// RegisterNode registers an edge node with the Control Plane.
func (c *ControlPlaneClient) RegisterNode(ctx context.Context, node *EdgeNode) error {
	payload := nodeRegistration{
		ID:       node.ID,
		Addr:     node.Addr,
		Region:   node.Region,
		Role:     string(node.Role),
		Tags:     node.Tags,
		Capacity: node.Capacity,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal registration: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/nodes", c.config.BaseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.config.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.BearerToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("register node %q: %w", node.ID, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("register node %q: status %d", node.ID, resp.StatusCode)
	}

	c.logger.Info("edge node registered with Control Plane",
		"node", node.ID, "region", node.Region)
	return nil
}

// heartbeatPayload is the payload for POST /api/v1/nodes/{id}/heartbeat.
type heartbeatPayload struct {
	Status    string        `json:"status"`
	Latency   time.Duration `json:"latency_ns"`
	LastSeen  time.Time     `json:"last_seen"`
}

// SendHeartbeat sends a heartbeat for a specific node.
func (c *ControlPlaneClient) SendHeartbeat(ctx context.Context, node *EdgeNode) error {
	payload := heartbeatPayload{
		Status:   string(node.GetStatus()),
		Latency:  node.Latency,
		LastSeen: node.LastSeen,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal heartbeat: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/nodes/%s/heartbeat", c.config.BaseURL, node.ID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	if c.config.BearerToken != "" {
		req.Header.Set("Authorization", "Bearer "+c.config.BearerToken)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("heartbeat node %q: %w", node.ID, err)
	}
	defer resp.Body.Close()
	io.Copy(io.Discard, resp.Body)

	if resp.StatusCode >= 400 {
		return fmt.Errorf("heartbeat node %q: status %d", node.ID, resp.StatusCode)
	}
	return nil
}

// StartHeartbeatLoop begins periodic heartbeats for a registered node.
func (c *ControlPlaneClient) StartHeartbeatLoop(node *EdgeNode) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Cancel existing heartbeat if any
	if cancel, ok := c.heartbeats[node.ID]; ok {
		cancel()
	}

	ctx, cancel := context.WithCancel(context.Background())
	c.heartbeats[node.ID] = cancel

	go func() {
		ticker := time.NewTicker(c.config.HeartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := c.SendHeartbeat(ctx, node); err != nil {
					c.logger.Warn("heartbeat failed",
						"node", node.ID, "error", err)
				}
			}
		}
	}()
}

// StopHeartbeatLoop stops the heartbeat loop for a node.
func (c *ControlPlaneClient) StopHeartbeatLoop(nodeID string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if cancel, ok := c.heartbeats[nodeID]; ok {
		cancel()
		delete(c.heartbeats, nodeID)
	}
}

// StopAll stops all heartbeat loops.
func (c *ControlPlaneClient) StopAll() {
	c.mu.Lock()
	defer c.mu.Unlock()

	for id, cancel := range c.heartbeats {
		cancel()
		delete(c.heartbeats, id)
	}
}

// ConnectRegistry wires the Control Plane client into an edge Registry's
// OnRegister/OnDeregister callbacks so that:
//   - When a node is registered, it is reported to the Control Plane and heartbeats begin.
//   - When a node is deregistered, heartbeats stop.
func (c *ControlPlaneClient) ConnectRegistry(reg *Registry) {
	reg.OnRegister(func(node *EdgeNode) {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := c.RegisterNode(ctx, node); err != nil {
			c.logger.Error("failed to register edge node with Control Plane",
				"node", node.ID, "error", err)
			return
		}
		c.StartHeartbeatLoop(node)
	})

	reg.OnDeregister(func(node *EdgeNode) {
		c.StopHeartbeatLoop(node.ID)
		c.logger.Info("edge node deregistered, heartbeats stopped",
			"node", node.ID)
	})
}
