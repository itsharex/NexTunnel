package controlplane

import (
	"fmt"
	"net"
	"sync"
)

// MemoryStore is an in-memory implementation of the Store interface.
// Suitable for testing and small deployments.
type MemoryStore struct {
	mu    sync.RWMutex
	nodes map[string]*NodeInfo
	acls  map[string]*ACLRule
	keys  map[string]*KeyMaterial
	ips   map[string]net.IP // nodeID -> allocated IP
}

// NewMemoryStore creates a new in-memory store.
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		nodes: make(map[string]*NodeInfo),
		acls:  make(map[string]*ACLRule),
		keys:  make(map[string]*KeyMaterial),
		ips:   make(map[string]net.IP),
	}
}

func (s *MemoryStore) SaveNode(node *NodeInfo) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.nodes[node.NodeID] = node
	return nil
}

func (s *MemoryStore) GetNode(nodeID string) (*NodeInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	node, ok := s.nodes[nodeID]
	if !ok {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}
	return node, nil
}

func (s *MemoryStore) ListNodes() ([]*NodeInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*NodeInfo, 0, len(s.nodes))
	for _, n := range s.nodes {
		result = append(result, n)
	}
	return result, nil
}

func (s *MemoryStore) DeleteNode(nodeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.nodes, nodeID)
	return nil
}

func (s *MemoryStore) SaveACLRule(rule *ACLRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.acls[rule.ID] = rule
	return nil
}

func (s *MemoryStore) GetACLRule(ruleID string) (*ACLRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rule, ok := s.acls[ruleID]
	if !ok {
		return nil, fmt.Errorf("rule not found: %s", ruleID)
	}
	return rule, nil
}

func (s *MemoryStore) ListACLRules() ([]*ACLRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*ACLRule, 0, len(s.acls))
	for _, r := range s.acls {
		result = append(result, r)
	}
	return result, nil
}

func (s *MemoryStore) DeleteACLRule(ruleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.acls, ruleID)
	return nil
}

func (s *MemoryStore) SaveKeyMaterial(km *KeyMaterial) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.keys[km.NodeID] = km
	return nil
}

func (s *MemoryStore) GetKeyMaterial(nodeID string) (*KeyMaterial, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	km, ok := s.keys[nodeID]
	if !ok {
		return nil, fmt.Errorf("key material not found: %s", nodeID)
	}
	return km, nil
}

func (s *MemoryStore) SaveIPAllocation(nodeID string, ip net.IP) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.ips[nodeID] = ip
	return nil
}

func (s *MemoryStore) GetIPAllocation(nodeID string) (net.IP, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	ip, ok := s.ips[nodeID]
	if !ok {
		return nil, fmt.Errorf("IP allocation not found: %s", nodeID)
	}
	return ip, nil
}

func (s *MemoryStore) DeleteIPAllocation(nodeID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.ips, nodeID)
	return nil
}

func (s *MemoryStore) ListIPAllocations() (map[string]net.IP, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make(map[string]net.IP, len(s.ips))
	for k, v := range s.ips {
		result[k] = v
	}
	return result, nil
}
