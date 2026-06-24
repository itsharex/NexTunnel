// Package types provides shared types used across NexTunnel client and server components.
package types

import "time"

// ProxyType defines the type of tunnel proxy.
type ProxyType string

const (
	ProxyTypeTCP  ProxyType = "tcp"
	ProxyTypeHTTP ProxyType = "http"
	ProxyTypeUDP  ProxyType = "udp"
)

// ProxyStatus represents the runtime status of a proxy.
type ProxyStatus string

const (
	ProxyStatusActive   ProxyStatus = "active"
	ProxyStatusInactive ProxyStatus = "inactive"
	ProxyStatusError    ProxyStatus = "error"
)

// TunnelConfig holds the client-side configuration for a single tunnel.
type TunnelConfig struct {
	Name       string    `json:"name"`
	ProxyType  ProxyType `json:"proxy_type"`
	LocalAddr  string    `json:"local_addr"`
	RemotePort uint16    `json:"remote_port"`
	ServerAddr string    `json:"server_addr"`
}

// ProxyInfo describes the runtime state of a proxy tunnel.
type ProxyInfo struct {
	ProxyName  string      `json:"proxy_name"`
	ProxyType  ProxyType   `json:"proxy_type"`
	LocalAddr  string      `json:"local_addr"`
	RemotePort uint16      `json:"remote_port"`
	Status     ProxyStatus `json:"status"`
	BytesIn    int64       `json:"bytes_in"`
	BytesOut   int64       `json:"bytes_out"`
	Sessions   int64       `json:"sessions"`
}

// ClientInfo holds metadata about a connected tunnel client.
type ClientInfo struct {
	ClientID    string    `json:"client_id"`
	ConnectedAt time.Time `json:"connected_at"`
	ProxyNames  []string  `json:"proxy_names"`
}

// --- Phase 3 shared types (Intelligent Scheduling) ---

// PathType identifies the type of network path for the scheduler.
type PathType string

const (
	PathTypeUDPP2P      PathType = "udp_p2p"
	PathTypeQUICP2P     PathType = "quic_p2p"
	PathTypeTCPP2P      PathType = "tcp_p2p"
	PathTypeNearbyRelay PathType = "nearby_relay"
	PathTypeGlobalRelay PathType = "global_relay"
)

// LinkMetricsSnapshot is a serializable snapshot of link quality metrics.
type LinkMetricsSnapshot struct {
	PathType  PathType      `json:"path_type"`
	RTT       time.Duration `json:"rtt"`
	LossRate  float64       `json:"loss_rate"`
	Bandwidth int64         `json:"bandwidth_bps"`
	Active    bool          `json:"active"`
}

// RelayNodeInfo describes a relay server node.
type RelayNodeInfo struct {
	Addr   string        `json:"addr"`
	Region string        `json:"region"`
	RTT    time.Duration `json:"rtt"`
	Active bool          `json:"active"`
}
