package nat

import "fmt"

// NATType represents the type of NAT detected by the client.
type NATType string

const (
	NATUnknown        NATType = "unknown"
	NATOpenInternet   NATType = "open_internet"
	NATFullCone       NATType = "full_cone"
	NATRestricted     NATType = "restricted"
	NATPortRestricted NATType = "port_restricted"
	NATSymmetric      NATType = "symmetric"
	NATBlocked        NATType = "blocked"
)

// NATResult holds the result of a NAT type detection.
type NATResult struct {
	Type       NATType `json:"type"`
	PublicAddr string  `json:"public_addr"` // server-reflexive address (ip:port)
	MappedPort uint16  `json:"mapped_port"`
	LocalAddr  string  `json:"local_addr"` // local address used for detection
}

// IsP2PPossible returns true if P2P hole punching is likely feasible
// with this NAT type against a peer with the given NAT type.
// Symmetric + Symmetric is the only combination where P2P is impossible.
func (r *NATResult) IsP2PPossible(peerNAT NATType) bool {
	if r.Type == NATSymmetric && peerNAT == NATSymmetric {
		return false
	}
	if r.Type == NATBlocked || peerNAT == NATBlocked {
		return false
	}
	return true
}

// String returns a human-readable description of the NAT type.
func (n NATType) String() string {
	switch n {
	case NATOpenInternet:
		return "Open Internet (no NAT)"
	case NATFullCone:
		return "Full Cone NAT"
	case NATRestricted:
		return "Restricted Cone NAT"
	case NATPortRestricted:
		return "Port Restricted Cone NAT"
	case NATSymmetric:
		return "Symmetric NAT"
	case NATBlocked:
		return "Blocked (UDP not available)"
	default:
		return fmt.Sprintf("Unknown NAT (%s)", string(n))
	}
}
