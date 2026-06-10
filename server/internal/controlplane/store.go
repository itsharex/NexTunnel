package controlplane

import (
	"net"
	"time"
)

// Store defines the data persistence interface for the control plane.
type Store interface {
	// Node operations
	SaveNode(node *NodeInfo) error
	GetNode(nodeID string) (*NodeInfo, error)
	ListNodes() ([]*NodeInfo, error)
	DeleteNode(nodeID string) error

	// ACL operations
	SaveACLRule(rule *ACLRule) error
	GetACLRule(ruleID string) (*ACLRule, error)
	ListACLRules() ([]*ACLRule, error)
	DeleteACLRule(ruleID string) error

	// Key operations
	SaveKeyMaterial(km *KeyMaterial) error
	GetKeyMaterial(nodeID string) (*KeyMaterial, error)

	// IP allocation operations
	SaveIPAllocation(nodeID string, ip net.IP) error
	GetIPAllocation(nodeID string) (net.IP, error)
	DeleteIPAllocation(nodeID string) error
	ListIPAllocations() (map[string]net.IP, error)
}

// NodeInfo represents a registered node.
type NodeInfo struct {
	NodeID      string            `json:"node_id"`
	PublicKey   string            `json:"public_key"`
	NATType     string            `json:"nat_type"`
	Region      string            `json:"region"`
	Subnet      string            `json:"subnet"`
	Metadata    map[string]string `json:"metadata"`
	ConnectedAt time.Time         `json:"connected_at"`
	LastSeen    time.Time         `json:"last_seen"`
}

// ACLRule defines an access control rule.
type ACLRule struct {
	ID        string     `json:"id"`
	Source    string     `json:"source"`
	Target    string     `json:"target"`
	Action    string     `json:"action"` // "allow" or "deny"
	Protocol  string     `json:"protocol"`
	Ports     []int      `json:"ports"`
	Priority  int        `json:"priority"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
	CreatedAt time.Time  `json:"created_at"`
}

// KeyMaterial holds a node's WireGuard key material.
type KeyMaterial struct {
	NodeID     string    `json:"node_id"`
	PublicKey  string    `json:"public_key"`
	KeyVersion int       `json:"key_version"`
	RotatedAt  time.Time `json:"rotated_at"`
	ExpiresAt  time.Time `json:"expires_at"`
}

// Role defines an RBAC role.
type Role string

const (
	RoleAdmin  Role = "admin"
	RoleUser   Role = "user"
	RoleViewer Role = "viewer"
)
