package scheduler

import (
	"net"
	"time"

	"github.com/nextunnel/desktop/internal/probe"
)

// PathType identifies the type of network path.
type PathType int

const (
	PathUDPP2P      PathType = iota // Priority 1: UDP direct
	PathQUICP2P                     // Priority 2: QUIC direct
	PathTCPP2P                      // Priority 3: TCP P2P fallback
	PathNearbyRelay                 // Priority 4: Geo-close relay
	PathGlobalRelay                 // Priority 5: Any relay
)

// PathTypeName maps PathType to human-readable names.
var PathTypeName = map[PathType]string{
	PathUDPP2P:      "udp_p2p",
	PathQUICP2P:     "quic_p2p",
	PathTCPP2P:      "tcp_p2p",
	PathNearbyRelay: "nearby_relay",
	PathGlobalRelay: "global_relay",
}

func (p PathType) String() string {
	if name, ok := PathTypeName[p]; ok {
		return name
	}
	return "unknown"
}

// PathState represents the availability of a path.
type PathState string

const (
	PathAvailable   PathState = "available"
	PathDegraded    PathState = "degraded"
	PathUnavailable PathState = "unavailable"
)

// Transport is the interface a path's underlying transport must satisfy.
type Transport interface {
	Read([]byte) (int, error)
	Write([]byte) (int, error)
	Close() error
	LocalAddr() net.Addr
	RemoteAddr() net.Addr
}

// Path represents a single network path with its metrics.
type Path struct {
	ID         string
	Type       PathType
	Transport  Transport
	Prober     *probe.Prober
	State      PathState
	Metrics    probe.LinkMetrics
	ManualLock bool
	CreatedAt  time.Time
}

// Priority returns the numeric priority (lower = better).
func (p *Path) Priority() int {
	return int(p.Type) + 1
}
