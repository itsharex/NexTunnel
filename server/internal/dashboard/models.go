package dashboard

import "time"

// NodeStatus represents the dashboard view of a node.
type NodeStatus struct {
	NodeID      string    `json:"node_id"`
	Region      string    `json:"region"`
	NATType     string    `json:"nat_type"`
	Online      bool      `json:"online"`
	ConnectedAt time.Time `json:"connected_at"`
	LastSeen    time.Time `json:"last_seen"`
	RxBytes     int64     `json:"rx_bytes"`
	TxBytes     int64     `json:"tx_bytes"`
}

// TrafficStats represents traffic statistics for a node or global view.
type TrafficStats struct {
	NodeID      string    `json:"node_id,omitempty"`
	RxBytes     int64     `json:"rx_bytes"`
	TxBytes     int64     `json:"tx_bytes"`
	RxBandwidth float64   `json:"rx_bandwidth_bps"`
	TxBandwidth float64   `json:"tx_bandwidth_bps"`
	Connections int       `json:"connections"`
	Timestamp   time.Time `json:"timestamp"`
}

// ACLRuleView represents an ACL rule for the dashboard.
type ACLRuleView struct {
	ID        string    `json:"id"`
	Source    string    `json:"source"`
	Target    string    `json:"target"`
	Action    string    `json:"action"`
	Protocol  string    `json:"protocol"`
	Priority  int       `json:"priority"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

// Alert represents a system alert.
type Alert struct {
	ID        string    `json:"id"`
	Level     string    `json:"level"` // "info", "warning", "critical"
	Message   string    `json:"message"`
	NodeID    string    `json:"node_id,omitempty"`
	CreatedAt time.Time `json:"created_at"`
	Acked     bool      `json:"acked"`
}

// User represents a dashboard user.
type User struct {
	ID           string `json:"id"`
	Username     string `json:"username"`
	Role         string `json:"role"` // "admin", "operator", "viewer"
	Email        string `json:"email"`
	PasswordHash string `json:"-"`
}

// LoginRequest is the request body for user login.
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// LoginResponse is the response body for successful login.
type LoginResponse struct {
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expires_at"`
	User      *User     `json:"user"`
}

// APIResponse is a standard API response wrapper.
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}
