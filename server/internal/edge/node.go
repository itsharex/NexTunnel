package edge

import (
	"fmt"
	"net"
	"sync/atomic"
	"time"
)

// NodeStatus represents the health status of an edge node.
type NodeStatus string

const (
	StatusHealthy   NodeStatus = "healthy"
	StatusUnhealthy NodeStatus = "unhealthy"
	StatusDraining  NodeStatus = "draining"
	StatusOffline   NodeStatus = "offline"
)

// NodeRole defines the capability of an edge node.
type NodeRole string

const (
	RoleRelay       NodeRole = "relay"
	RoleAccelerator NodeRole = "accelerator"
	RoleFull        NodeRole = "full" // relay + accelerator
)

// EdgeNode represents a single edge node in the global network.
type EdgeNode struct {
	ID         string            `json:"id"`
	Addr       string            `json:"addr"`
	Region     string            `json:"region"`
	Role       NodeRole          `json:"role"`
	Status     atomic.Value      `json:"-"` // NodeStatus
	Tags       map[string]string `json:"tags"`
	Capacity   int               `json:"capacity"` // max concurrent connections
	Registered time.Time         `json:"registered"`
	LastSeen   time.Time         `json:"last_seen"`
	Latency    time.Duration     `json:"latency"`

	// health tracking
	consecutiveFails atomic.Int32
	consecutiveOKs   atomic.Int32
	unhealthySince   atomic.Value // time.Time (zero if healthy)
}

// NewEdgeNode creates a new edge node with the given parameters.
func NewEdgeNode(id, addr, region string, role NodeRole, capacity int) (*EdgeNode, error) {
	if id == "" {
		return nil, fmt.Errorf("edge node id is required")
	}
	if addr == "" {
		return nil, fmt.Errorf("edge node addr is required")
	}
	if _, err := net.ResolveTCPAddr("tcp", addr); err != nil {
		return nil, fmt.Errorf("invalid addr %q: %w", addr, err)
	}

	n := &EdgeNode{
		ID:         id,
		Addr:       addr,
		Region:     region,
		Role:       role,
		Tags:       make(map[string]string),
		Capacity:   capacity,
		Registered: time.Now(),
		LastSeen:   time.Now(),
	}
	n.Status.Store(StatusHealthy)
	n.unhealthySince.Store(time.Time{})
	return n, nil
}

// GetStatus returns the current node status.
func (n *EdgeNode) GetStatus() NodeStatus {
	return n.Status.Load().(NodeStatus)
}

// SetStatus updates the node status.
func (n *EdgeNode) SetStatus(s NodeStatus) {
	n.Status.Store(s)
}

// RecordSuccess records a successful health probe.
func (n *EdgeNode) RecordSuccess(latency time.Duration) {
	n.consecutiveFails.Store(0)
	n.consecutiveOKs.Add(1)
	n.LastSeen = time.Now()
	n.Latency = latency
}

// RecordFailure records a failed health probe.
func (n *EdgeNode) RecordFailure() {
	n.consecutiveOKs.Store(0)
	n.consecutiveFails.Add(1)
}

// ConsecutiveFails returns the number of consecutive failures.
func (n *EdgeNode) ConsecutiveFails() int {
	return int(n.consecutiveFails.Load())
}

// ConsecutiveOKs returns the number of consecutive successes.
func (n *EdgeNode) ConsecutiveOKs() int {
	return int(n.consecutiveOKs.Load())
}

// SetUnhealthySince records when the node first became unhealthy.
func (n *EdgeNode) SetUnhealthySince(t time.Time) {
	n.unhealthySince.Store(t)
}

// UnhealthySince returns when the node became unhealthy.
func (n *EdgeNode) UnhealthySince() time.Time {
	return n.unhealthySince.Load().(time.Time)
}
