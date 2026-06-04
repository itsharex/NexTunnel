package edge

import (
	"fmt"
	"sort"
	"sync"
)

// Registry manages edge node registration and lookup.
type Registry struct {
	mu    sync.RWMutex
	nodes map[string]*EdgeNode

	onRegister   func(*EdgeNode)
	onDeregister func(*EdgeNode)
}

// NewRegistry creates a new edge node registry.
func NewRegistry() *Registry {
	return &Registry{
		nodes: make(map[string]*EdgeNode),
	}
}

// OnRegister sets a callback for node registration events.
func (r *Registry) OnRegister(fn func(*EdgeNode)) {
	r.onRegister = fn
}

// OnDeregister sets a callback for node deregistration events.
func (r *Registry) OnDeregister(fn func(*EdgeNode)) {
	r.onDeregister = fn
}

// Register adds a new edge node to the registry.
func (r *Registry) Register(node *EdgeNode) error {
	if node == nil {
		return fmt.Errorf("node is nil")
	}

	r.mu.Lock()
	if _, exists := r.nodes[node.ID]; exists {
		r.mu.Unlock()
		return fmt.Errorf("node %q already registered", node.ID)
	}
	r.nodes[node.ID] = node
	r.mu.Unlock()

	if r.onRegister != nil {
		r.onRegister(node)
	}
	return nil
}

// Deregister removes a node from the registry.
func (r *Registry) Deregister(nodeID string) error {
	r.mu.Lock()
	node, ok := r.nodes[nodeID]
	if !ok {
		r.mu.Unlock()
		return fmt.Errorf("node %q not found", nodeID)
	}
	delete(r.nodes, nodeID)
	r.mu.Unlock()

	if r.onDeregister != nil {
		r.onDeregister(node)
	}
	return nil
}

// Get returns a node by ID.
func (r *Registry) Get(nodeID string) (*EdgeNode, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	n, ok := r.nodes[nodeID]
	return n, ok
}

// List returns all registered nodes.
func (r *Registry) List() []*EdgeNode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	result := make([]*EdgeNode, 0, len(r.nodes))
	for _, n := range r.nodes {
		result = append(result, n)
	}
	return result
}

// ListByRegion returns nodes filtered by region.
func (r *Registry) ListByRegion(region string) []*EdgeNode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*EdgeNode
	for _, n := range r.nodes {
		if n.Region == region {
			result = append(result, n)
		}
	}
	return result
}

// ListHealthy returns only healthy nodes, sorted by latency (ascending).
func (r *Registry) ListHealthy() []*EdgeNode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*EdgeNode
	for _, n := range r.nodes {
		if n.GetStatus() == StatusHealthy {
			result = append(result, n)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Latency < result[j].Latency
	})
	return result
}

// ListHealthyByRegion returns healthy nodes in a region sorted by latency.
func (r *Registry) ListHealthyByRegion(region string) []*EdgeNode {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*EdgeNode
	for _, n := range r.nodes {
		if n.Region == region && n.GetStatus() == StatusHealthy {
			result = append(result, n)
		}
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Latency < result[j].Latency
	})
	return result
}

// Count returns the total number of registered nodes.
func (r *Registry) Count() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.nodes)
}

// CountHealthy returns the number of healthy nodes.
func (r *Registry) CountHealthy() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	count := 0
	for _, n := range r.nodes {
		if n.GetStatus() == StatusHealthy {
			count++
		}
	}
	return count
}

// Regions returns all unique regions.
func (r *Registry) Regions() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	seen := make(map[string]struct{})
	for _, n := range r.nodes {
		seen[n.Region] = struct{}{}
	}
	result := make([]string, 0, len(seen))
	for region := range seen {
		result = append(result, region)
	}
	sort.Strings(result)
	return result
}
