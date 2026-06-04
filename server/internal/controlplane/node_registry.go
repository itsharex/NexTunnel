package controlplane

import (
	"fmt"
	"log/slog"
	"sync"
	"time"
)

// NodeRegistry manages node registration and lifecycle.
type NodeRegistry struct {
	store  Store
	nodes  sync.Map // nodeID -> *NodeInfo
	logger *slog.Logger
}

// NewNodeRegistry creates a new node registry.
func NewNodeRegistry(store Store, logger *slog.Logger) *NodeRegistry {
	if logger == nil {
		logger = slog.Default()
	}
	return &NodeRegistry{store: store, logger: logger}
}

// Register adds a new node or updates an existing one.
func (r *NodeRegistry) Register(node *NodeInfo) error {
	node.ConnectedAt = time.Now()
	node.LastSeen = time.Now()

	if err := r.store.SaveNode(node); err != nil {
		return fmt.Errorf("save node: %w", err)
	}

	r.nodes.Store(node.NodeID, node)
	r.logger.Info("node registered", "id", node.NodeID, "region", node.Region, "nat", node.NATType)
	return nil
}

// Heartbeat updates the last-seen timestamp for a node.
func (r *NodeRegistry) Heartbeat(nodeID string) error {
	v, ok := r.nodes.Load(nodeID)
	if !ok {
		return fmt.Errorf("node not found: %s", nodeID)
	}
	node := v.(*NodeInfo)
	node.LastSeen = time.Now()
	return r.store.SaveNode(node)
}

// Deregister removes a node.
func (r *NodeRegistry) Deregister(nodeID string) error {
	r.nodes.Delete(nodeID)
	r.logger.Info("node deregistered", "id", nodeID)
	return r.store.DeleteNode(nodeID)
}

// Get returns a node by ID.
func (r *NodeRegistry) Get(nodeID string) (*NodeInfo, error) {
	v, ok := r.nodes.Load(nodeID)
	if !ok {
		return r.store.GetNode(nodeID)
	}
	return v.(*NodeInfo), nil
}

// List returns all registered nodes.
func (r *NodeRegistry) List() []*NodeInfo {
	var result []*NodeInfo
	r.nodes.Range(func(_, value any) bool {
		result = append(result, value.(*NodeInfo))
		return true
	})
	return result
}

// Count returns the number of registered nodes.
func (r *NodeRegistry) Count() int {
	count := 0
	r.nodes.Range(func(_, _ any) bool {
		count++
		return true
	})
	return count
}

// PruneStale removes nodes that haven't been seen within the timeout.
func (r *NodeRegistry) PruneStale(timeout time.Duration) int {
	cutoff := time.Now().Add(-timeout)
	pruned := 0

	r.nodes.Range(func(key, value any) bool {
		node := value.(*NodeInfo)
		if node.LastSeen.Before(cutoff) {
			r.nodes.Delete(key)
			r.store.DeleteNode(node.NodeID)
			pruned++
			r.logger.Info("stale node pruned", "id", node.NodeID, "last_seen", node.LastSeen)
		}
		return true
	})

	return pruned
}

// ACLRuleEngine evaluates access control rules.
type ACLRuleEngine struct {
	store  Store
	rules  sync.Map // ruleID -> *ACLRule
	logger *slog.Logger
}

// NewACLRuleEngine creates a new ACL engine.
func NewACLRuleEngine(store Store, logger *slog.Logger) *ACLRuleEngine {
	if logger == nil {
		logger = slog.Default()
	}
	return &ACLRuleEngine{store: store, logger: logger}
}

// AddRule adds or updates an ACL rule.
func (e *ACLRuleEngine) AddRule(rule *ACLRule) error {
	rule.CreatedAt = time.Now()
	if err := e.store.SaveACLRule(rule); err != nil {
		return fmt.Errorf("save rule: %w", err)
	}
	e.rules.Store(rule.ID, rule)
	e.logger.Info("ACL rule added", "id", rule.ID, "action", rule.Action, "source", rule.Source, "target", rule.Target)
	return nil
}

// RemoveRule removes an ACL rule.
func (e *ACLRuleEngine) RemoveRule(ruleID string) error {
	e.rules.Delete(ruleID)
	return e.store.DeleteACLRule(ruleID)
}

// Evaluate checks if a source can access a target.
// Default policy: deny all unless explicitly allowed.
func (e *ACLRuleEngine) Evaluate(source, target, protocol string, port int) bool {
	var bestRule *ACLRule
	bestPriority := -1

	e.rules.Range(func(_, value any) bool {
		rule := value.(*ACLRule)

		// Check expiry
		if rule.ExpiresAt != nil && rule.ExpiresAt.Before(time.Now()) {
			return true
		}

		// Match source
		if rule.Source != "*" && rule.Source != source {
			return true
		}

		// Match target
		if rule.Target != "*" && rule.Target != target {
			return true
		}

		// Match protocol
		if rule.Protocol != "*" && rule.Protocol != protocol {
			return true
		}

		// Match port
		if len(rule.Ports) > 0 {
			portMatch := false
			for _, p := range rule.Ports {
				if p == port {
					portMatch = true
					break
				}
			}
			if !portMatch {
				return true
			}
		}

		// Track highest priority matching rule
		if rule.Priority > bestPriority {
			bestPriority = rule.Priority
			bestRule = rule
		}

		return true
	})

	if bestRule == nil {
		return false // default deny
	}

	return bestRule.Action == "allow"
}

// ListRules returns all ACL rules.
func (e *ACLRuleEngine) ListRules() []*ACLRule {
	var result []*ACLRule
	e.rules.Range(func(_, value any) bool {
		result = append(result, value.(*ACLRule))
		return true
	})
	return result
}

// KeyExchange manages WireGuard key distribution and rotation.
type KeyExchange struct {
	store  Store
	logger *slog.Logger
}

// NewKeyExchange creates a new key exchange manager.
func NewKeyExchange(store Store, logger *slog.Logger) *KeyExchange {
	if logger == nil {
		logger = slog.Default()
	}
	return &KeyExchange{store: store, logger: logger}
}

// RegisterKey stores a node's public key.
func (k *KeyExchange) RegisterKey(nodeID, publicKey string, version int, expiry time.Duration) error {
	km := &KeyMaterial{
		NodeID:     nodeID,
		PublicKey:  publicKey,
		KeyVersion: version,
		RotatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(expiry),
	}
	if err := k.store.SaveKeyMaterial(km); err != nil {
		return fmt.Errorf("save key: %w", err)
	}
	k.logger.Info("key registered", "node", nodeID, "version", version)
	return nil
}

// GetPeerKey returns a peer's public key.
func (k *KeyExchange) GetPeerKey(nodeID string) (*KeyMaterial, error) {
	return k.store.GetKeyMaterial(nodeID)
}

// RotateKey creates a new key version for a node.
func (k *KeyExchange) RotateKey(nodeID, newPublicKey string) error {
	existing, err := k.store.GetKeyMaterial(nodeID)
	if err != nil {
		return fmt.Errorf("get existing key: %w", err)
	}

	km := &KeyMaterial{
		NodeID:     nodeID,
		PublicKey:  newPublicKey,
		KeyVersion: existing.KeyVersion + 1,
		RotatedAt:  time.Now(),
		ExpiresAt:  existing.ExpiresAt,
	}

	if err := k.store.SaveKeyMaterial(km); err != nil {
		return fmt.Errorf("save rotated key: %w", err)
	}

	k.logger.Info("key rotated", "node", nodeID, "new_version", km.KeyVersion)
	return nil
}
