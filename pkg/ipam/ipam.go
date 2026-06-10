// Package ipam provides IP Address Management for virtual networks.
package ipam

import (
	"encoding/binary"
	"fmt"
	"net"
	"sync"
)

// Allocator manages IP address allocation within a subnet.
type Allocator interface {
	Allocate(nodeID string) (net.IP, error)
	Reserve(nodeID string, ip net.IP) error
	Release(nodeID string)
	GetAllocation(nodeID string) (net.IP, bool)
	ListAllocations() map[string]net.IP
}

// IPAM implements IP address allocation from a CIDR subnet.
type IPAM struct {
	mu        sync.Mutex
	subnet    *net.IPNet
	gateway   net.IP
	allocated map[string]net.IP // nodeID -> IP
	nextHost  uint32            // next host offset to try
	network   uint32            // network address as uint32
	broadcast uint32            // broadcast address as uint32
}

// NewIPAM creates an IPAM allocator for the given CIDR subnet.
// The gateway is reserved and will not be allocated.
func NewIPAM(subnetCIDR string, gatewayIP string) (*IPAM, error) {
	_, subnet, err := net.ParseCIDR(subnetCIDR)
	if err != nil {
		return nil, fmt.Errorf("parse subnet CIDR %q: %w", subnetCIDR, err)
	}
	networkIP4 := subnet.IP.To4()
	if networkIP4 == nil {
		return nil, fmt.Errorf("subnet %s is not an IPv4 CIDR", subnetCIDR)
	}

	gateway := net.ParseIP(gatewayIP).To4()
	if gateway == nil {
		return nil, fmt.Errorf("invalid gateway IP %q", gatewayIP)
	}

	if !subnet.Contains(gateway) {
		return nil, fmt.Errorf("gateway %s not in subnet %s", gatewayIP, subnetCIDR)
	}

	ones, bits := subnet.Mask.Size()
	hostBits := uint(bits - ones)
	if hostBits < 2 {
		return nil, fmt.Errorf("subnet %s too small (need at least /30)", subnetCIDR)
	}

	networkIP := ipToUint32(networkIP4)
	bcast := networkIP | ((1 << hostBits) - 1)

	return &IPAM{
		subnet:    subnet,
		gateway:   gateway,
		allocated: make(map[string]net.IP),
		nextHost:  1,
		network:   networkIP,
		broadcast: bcast,
	}, nil
}

// Allocate assigns an IP address to the given node.
// If the node already has an allocation, it returns the existing IP.
func (m *IPAM) Allocate(nodeID string) (net.IP, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for existing allocation
	if ip, ok := m.allocated[nodeID]; ok {
		return ip, nil
	}

	// Collect allocated IPs for collision check
	used := make(map[uint32]bool, len(m.allocated)+2)
	used[ipToUint32(m.subnet.IP.To4())] = true // network address
	used[m.broadcast] = true                   // broadcast address
	used[ipToUint32(m.gateway)] = true         // gateway
	for _, ip := range m.allocated {
		used[ipToUint32(ip.To4())] = true
	}

	// Find next available IP
	start := m.nextHost
	for offset := m.nextHost; m.network+offset < m.broadcast; offset++ {
		candidate := m.network + offset
		if !used[candidate] {
			ip := uint32ToIP(candidate)
			m.allocated[nodeID] = ip
			m.nextHost = offset + 1
			return ip, nil
		}
	}

	// Wrap around from beginning
	for offset := uint32(1); offset < start; offset++ {
		candidate := m.network + offset
		if candidate >= m.broadcast {
			break
		}
		if !used[candidate] {
			ip := uint32ToIP(candidate)
			m.allocated[nodeID] = ip
			m.nextHost = offset + 1
			return ip, nil
		}
	}

	return nil, fmt.Errorf("subnet %s exhausted", m.subnet.String())
}

// Reserve 为已有节点保留指定 IP，主要用于从持久化存储恢复分配状态。
func (m *IPAM) Reserve(nodeID string, ip net.IP) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	candidateIP := ip.To4()
	if candidateIP == nil {
		return fmt.Errorf("invalid IPv4 address %q", ip)
	}
	if !m.subnet.Contains(candidateIP) {
		return fmt.Errorf("IP %s not in subnet %s", candidateIP, m.subnet.String())
	}

	candidate := ipToUint32(candidateIP)
	if candidate == m.network || candidate == m.broadcast || candidate == ipToUint32(m.gateway) {
		return fmt.Errorf("IP %s is reserved in subnet %s", candidateIP, m.subnet.String())
	}

	if existingIP, ok := m.allocated[nodeID]; ok {
		if existingIP.Equal(candidateIP) {
			return nil
		}
		return fmt.Errorf("node %s already reserved IP %s", nodeID, existingIP)
	}
	for existingNodeID, existingIP := range m.allocated {
		if existingIP.Equal(candidateIP) {
			return fmt.Errorf("IP %s already reserved by node %s", candidateIP, existingNodeID)
		}
	}

	m.allocated[nodeID] = append(net.IP(nil), candidateIP...)
	if nextHost := candidate - m.network + 1; nextHost > m.nextHost && nextHost < m.broadcast-m.network {
		m.nextHost = nextHost
	}
	return nil
}

// Release frees the IP allocated to the given node.
func (m *IPAM) Release(nodeID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.allocated, nodeID)
}

// GetAllocation returns the IP allocated to a node.
func (m *IPAM) GetAllocation(nodeID string) (net.IP, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	ip, ok := m.allocated[nodeID]
	return ip, ok
}

// ListAllocations returns all current IP allocations.
func (m *IPAM) ListAllocations() map[string]net.IP {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make(map[string]net.IP, len(m.allocated))
	for k, v := range m.allocated {
		result[k] = v
	}
	return result
}

// Subnet returns the managed subnet CIDR.
func (m *IPAM) Subnet() *net.IPNet {
	return m.subnet
}

// Gateway returns the gateway IP.
func (m *IPAM) Gateway() net.IP {
	return m.gateway
}

func ipToUint32(ip net.IP) uint32 {
	return binary.BigEndian.Uint32(ip)
}

func uint32ToIP(n uint32) net.IP {
	ip := make(net.IP, 4)
	binary.BigEndian.PutUint32(ip, n)
	return ip
}
