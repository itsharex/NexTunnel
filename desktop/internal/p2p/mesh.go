package p2p

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/nextunnel/pkg/protocol"
)

// MeshState tracks the overall mesh network state.
type MeshState string

const (
	MeshStateOffline     MeshState = "offline"
	MeshStateJoining     MeshState = "joining"
	MeshStateConnected   MeshState = "connected"
	MeshStateDiscovering MeshState = "discovering"
)

// PeerInfo describes a peer in the mesh network.
type PeerInfo struct {
	ClientID  string    `json:"client_id"`
	NATType   string    `json:"nat_type"`
	WGPubKey  string    `json:"wg_pub_key"`
	Connected bool      `json:"connected"`
	Subnet    string    `json:"subnet,omitempty"`
	LastSeen  time.Time `json:"last_seen"`
}

// RouteEntry maps a peer to its transport and metadata.
type RouteEntry struct {
	PeerID    string
	Transport Transport
	Subnet    string
	AddedAt   time.Time
	BytesIn   atomic.Int64
	BytesOut  atomic.Int64
}

// MeshConfig configures the mesh router.
type MeshConfig struct {
	ClientID      string
	Control       ControlChannel
	Engine        *Engine
	PingInterval  time.Duration // how often to ping peers
	PeerTimeout   time.Duration // how long before a peer is considered offline
	MaxPeers      int
	Logger        *slog.Logger
}

// DefaultMeshConfig returns sensible defaults.
func DefaultMeshConfig() MeshConfig {
	return MeshConfig{
		PingInterval: 15 * time.Second,
		PeerTimeout:  60 * time.Second,
		MaxPeers:     32,
	}
}

// MeshRouter manages multiple P2P peer connections forming a mesh network.
type MeshRouter struct {
	config MeshConfig
	state  atomic.Value // MeshState

	peersMu sync.RWMutex
	peers   map[string]*PeerInfo   // clientID -> PeerInfo
	routes  map[string]*RouteEntry // clientID -> RouteEntry

	localSubnet string // this node's assigned subnet

	ctx    context.Context
	cancel context.CancelFunc
	logger *slog.Logger
}

// NewMeshRouter creates a new mesh router.
func NewMeshRouter(cfg MeshConfig) *MeshRouter {
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	m := &MeshRouter{
		config: cfg,
		peers:  make(map[string]*PeerInfo),
		routes: make(map[string]*RouteEntry),
		logger: cfg.Logger,
	}
	m.state.Store(MeshStateOffline)
	return m
}

// State returns the current mesh state.
func (m *MeshRouter) State() MeshState {
	return m.state.Load().(MeshState)
}

// Peers returns a snapshot of all known peers.
func (m *MeshRouter) Peers() []PeerInfo {
	m.peersMu.RLock()
	defer m.peersMu.RUnlock()

	result := make([]PeerInfo, 0, len(m.peers))
	for _, p := range m.peers {
		result = append(result, *p)
	}
	return result
}

// ConnectedPeers returns the count of connected peers.
func (m *MeshRouter) ConnectedPeers() int {
	m.peersMu.RLock()
	defer m.peersMu.RUnlock()

	count := 0
	for _, p := range m.peers {
		if p.Connected {
			count++
		}
	}
	return count
}

// GetRoute returns the transport for a given peer ID, or nil.
func (m *MeshRouter) GetRoute(peerID string) Transport {
	m.peersMu.RLock()
	defer m.peersMu.RUnlock()

	if entry, ok := m.routes[peerID]; ok {
		return entry.Transport
	}
	return nil
}

// JoinMesh sends a mesh join message and starts the peer discovery loop.
func (m *MeshRouter) JoinMesh(ctx context.Context, subnet string) error {
	m.ctx, m.cancel = context.WithCancel(ctx)
	m.localSubnet = subnet
	m.state.Store(MeshStateJoining)

	// Announce ourselves to the relay server
	joinMsg, err := protocol.NewMeshJoinMessage(
		m.config.ClientID,
		m.config.Engine.WGPUbKey(),
		m.config.Engine.GetNATType(),
		subnet,
	)
	if err != nil {
		return fmt.Errorf("create mesh join: %w", err)
	}
	if err := m.config.Control.Send(joinMsg); err != nil {
		return fmt.Errorf("send mesh join: %w", err)
	}

	// Start background loops
	go m.healthCheckLoop()
	go m.peerTimeoutLoop()

	m.state.Store(MeshStateConnected)
	m.logger.Info("joined mesh network", "subnet", subnet)
	return nil
}

// HandleMeshPeerList processes a list of peers from the relay server.
func (m *MeshRouter) HandleMeshPeerList(msg *protocol.MeshPeerListMessage) {
	m.state.Store(MeshStateDiscovering)

	for _, peer := range msg.Peers {
		if peer.ClientID == m.config.ClientID {
			continue // skip self
		}

		m.peersMu.Lock()
		existing, exists := m.peers[peer.ClientID]
		if !exists {
			existing = &PeerInfo{
				ClientID: peer.ClientID,
				NATType:  peer.NATType,
				WGPubKey: peer.WGPubKey,
				Subnet:   peer.Subnet,
				LastSeen: time.Now(),
			}
			m.peers[peer.ClientID] = existing
			m.peersMu.Unlock()

			m.logger.Info("discovered new peer", "peer", peer.ClientID, "nat", peer.NATType)

			// Initiate P2P connection to this peer
			go m.connectToPeer(peer.ClientID)
		} else {
			existing.LastSeen = time.Now()
			existing.NATType = peer.NATType
			existing.WGPubKey = peer.WGPubKey
			existing.Subnet = peer.Subnet
			m.peersMu.Unlock()
		}
	}

	m.state.Store(MeshStateConnected)
}

// HandleMeshPing processes a ping from a peer.
func (m *MeshRouter) HandleMeshPing(fromClientID string) {
	m.peersMu.Lock()
	defer m.peersMu.Unlock()

	if peer, ok := m.peers[fromClientID]; ok {
		peer.LastSeen = time.Now()
	}

	// Send pong response
	pong, err := protocol.NewMeshPongMessage(m.config.ClientID, fromClientID)
	if err == nil {
		m.config.Control.Send(pong)
	}
}

// HandleMeshPong processes a pong from a peer.
func (m *MeshRouter) HandleMeshPong(fromClientID string) {
	m.peersMu.Lock()
	defer m.peersMu.Unlock()

	if peer, ok := m.peers[fromClientID]; ok {
		peer.LastSeen = time.Now()
	}
}

// LeaveMesh gracefully leaves the mesh network.
func (m *MeshRouter) LeaveMesh() {
	if m.cancel != nil {
		m.cancel()
	}

	// Notify peers
	leaveMsg, err := protocol.NewMeshLeaveMessage(m.config.ClientID)
	if err == nil {
		m.config.Control.Send(leaveMsg)
	}

	// Close all peer routes
	m.peersMu.Lock()
	for id, route := range m.routes {
		if route.Transport != nil {
			route.Transport.Close()
		}
		delete(m.routes, id)
	}
	for id, peer := range m.peers {
		peer.Connected = false
		delete(m.peers, id)
	}
	m.peersMu.Unlock()

	m.state.Store(MeshStateOffline)
	m.logger.Info("left mesh network")
}

// AddPeerRoute registers an established transport as a route to a peer.
func (m *MeshRouter) AddPeerRoute(peerID string, transport Transport, subnet string) {
	m.peersMu.Lock()
	defer m.peersMu.Unlock()

	// Close old transport if exists
	if old, ok := m.routes[peerID]; ok && old.Transport != nil {
		old.Transport.Close()
	}

	m.routes[peerID] = &RouteEntry{
		PeerID:    peerID,
		Transport: transport,
		Subnet:    subnet,
		AddedAt:   time.Now(),
	}

	if peer, ok := m.peers[peerID]; ok {
		peer.Connected = true
		peer.LastSeen = time.Now()
	}

	m.logger.Info("route added", "peer", peerID, "subnet", subnet)
}

// RemovePeerRoute removes a peer route and closes the transport.
func (m *MeshRouter) RemovePeerRoute(peerID string) {
	m.peersMu.Lock()
	defer m.peersMu.Unlock()

	if route, ok := m.routes[peerID]; ok {
		if route.Transport != nil {
			route.Transport.Close()
		}
		delete(m.routes, peerID)
	}

	if peer, ok := m.peers[peerID]; ok {
		peer.Connected = false
	}

	m.logger.Info("route removed", "peer", peerID)
}

// LookupRoute finds the best route for a target subnet or peer ID.
func (m *MeshRouter) LookupRoute(targetPeerID string) *RouteEntry {
	m.peersMu.RLock()
	defer m.peersMu.RUnlock()

	// Direct route
	if entry, ok := m.routes[targetPeerID]; ok {
		return entry
	}

	// Subnet-based lookup
	for _, entry := range m.routes {
		if entry.Subnet != "" && matchSubnet(entry.Subnet, targetPeerID) {
			return entry
		}
	}

	return nil
}

// Stats returns mesh network statistics.
func (m *MeshRouter) Stats() MeshStats {
	m.peersMu.RLock()
	defer m.peersMu.RUnlock()

	stats := MeshStats{
		TotalPeers:     len(m.peers),
		ConnectedPeers: 0,
		Routes:         len(m.routes),
		Subnet:         m.localSubnet,
	}
	for _, p := range m.peers {
		if p.Connected {
			stats.ConnectedPeers++
		}
	}
	return stats
}

// MeshStats holds mesh network statistics.
type MeshStats struct {
	TotalPeers    int    `json:"total_peers"`
	ConnectedPeers int   `json:"connected_peers"`
	Routes        int    `json:"routes"`
	Subnet        string `json:"subnet"`
}

// --- Internal methods ---

// connectToPeer initiates a P2P connection to a discovered peer.
func (m *MeshRouter) connectToPeer(peerID string) {
	m.peersMu.RLock()
	peer, ok := m.peers[peerID]
	m.peersMu.RUnlock()

	if !ok {
		return
	}

	m.logger.Info("connecting to peer", "peer", peerID)

	// Use the P2P engine to establish connection
	// The offer/answer flow is handled via the control channel
	offer, err := protocol.NewP2POfferMessage(
		fmt.Sprintf("mesh-%s-%s", m.config.ClientID, peerID),
		m.config.ClientID,
		peerID,
		m.config.Engine.GetNATType(),
		m.config.Engine.WGPUbKey(),
		nil, // candidates will be gathered by HandleP2POffer on the peer side
	)
	if err != nil {
		m.logger.Error("failed to create P2P offer", "peer", peerID, "error", err)
		return
	}

	if err := m.config.Control.Send(offer); err != nil {
		m.logger.Error("failed to send P2P offer", "peer", peerID, "error", err)
		return
	}

	m.logger.Debug("P2P offer sent", "peer", peerID, "nat", peer.NATType)
}

// healthCheckLoop periodically pings all known peers.
func (m *MeshRouter) healthCheckLoop() {
	ticker := time.NewTicker(m.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.peersMu.RLock()
			peers := make([]string, 0, len(m.peers))
			for id := range m.peers {
				peers = append(peers, id)
			}
			m.peersMu.RUnlock()

			for _, peerID := range peers {
				ping, err := protocol.NewMeshPingMessage(m.config.ClientID, peerID)
				if err == nil {
					m.config.Control.Send(ping)
				}
			}
		}
	}
}

// peerTimeoutLoop removes peers that haven't been seen recently.
func (m *MeshRouter) peerTimeoutLoop() {
	ticker := time.NewTicker(m.config.PingInterval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.peersMu.Lock()
			now := time.Now()
			for id, peer := range m.peers {
				if now.Sub(peer.LastSeen) > m.config.PeerTimeout {
					m.logger.Info("peer timed out", "peer", id, "lastSeen", peer.LastSeen)
					if route, ok := m.routes[id]; ok && route.Transport != nil {
						route.Transport.Close()
					}
					delete(m.routes, id)
					delete(m.peers, id)
				}
			}
			m.peersMu.Unlock()
		}
	}
}

// matchSubnet checks if a peer ID could be reachable via a subnet route.
// This is a simplified check - in production, this would use actual IP routing.
func matchSubnet(subnet, targetPeerID string) bool {
	_, ipNet, err := net.ParseCIDR(subnet)
	if err != nil {
		return false
	}
	// For mesh routing, we check if the target has a registered subnet
	// that overlaps with this route's subnet
	targetIP := net.ParseIP(targetPeerID)
	if targetIP == nil {
		return false
	}
	return ipNet.Contains(targetIP)
}
