package controlplane

import (
	"fmt"

	"github.com/nextunnel/pkg/ipam"
)

// VirtualNetworkRoute 是控制面下发给节点的虚拟网络路由条目。
type VirtualNetworkRoute struct {
	Destination string `json:"destination"`
	Gateway     string `json:"gateway"`
	Interface   string `json:"interface"`
	Metric      int    `json:"metric"`
}

// VirtualNetworkConfig 是节点接入虚拟网络所需的最小配置。
type VirtualNetworkConfig struct {
	NodeID    string                `json:"node_id"`
	VirtualIP string                `json:"virtual_ip"`
	Subnet    string                `json:"subnet"`
	Gateway   string                `json:"gateway"`
	Interface string                `json:"interface"`
	MTU       int                   `json:"mtu"`
	Routes    []VirtualNetworkRoute `json:"routes"`
}

// VirtualNetworkManager 负责控制面虚拟 IP 分配和路由配置生成。
type VirtualNetworkManager struct {
	allocator   *ipam.IPAM
	store       Store
	subnet      string
	gateway     string
	iface       string
	mtu         int
	routeMetric int
}

// NewVirtualNetworkManager 创建虚拟网络管理器，并从持久化存储恢复已有地址分配。
func NewVirtualNetworkManager(cfg ControlPlaneConfig, store Store) (*VirtualNetworkManager, error) {
	allocator, err := ipam.NewIPAM(cfg.VirtualSubnet, cfg.VirtualGateway)
	if err != nil {
		return nil, fmt.Errorf("create virtual network IPAM: %w", err)
	}

	manager := &VirtualNetworkManager{
		allocator:   allocator,
		store:       store,
		subnet:      cfg.VirtualSubnet,
		gateway:     cfg.VirtualGateway,
		iface:       cfg.VirtualInterface,
		mtu:         cfg.VirtualMTU,
		routeMetric: cfg.VirtualRouteMetric,
	}
	if err := manager.restoreAllocations(); err != nil {
		return nil, err
	}
	return manager, nil
}

// AssignNode 为节点分配或复用虚拟 IP，并将结果写回节点模型。
func (m *VirtualNetworkManager) AssignNode(node *NodeInfo) error {
	if node == nil {
		return fmt.Errorf("node is nil")
	}
	if node.NodeID == "" {
		return fmt.Errorf("node_id is required")
	}

	if existingIP, err := m.store.GetIPAllocation(node.NodeID); err == nil {
		// 将持久化结果回填到内存分配器，避免同一地址在后续分配中被复用。
		if err := m.allocator.Reserve(node.NodeID, existingIP); err != nil {
			return fmt.Errorf("reserve existing virtual IP for %s: %w", node.NodeID, err)
		}
		node.VirtualIP = existingIP.String()
		node.Subnet = m.subnet
		return nil
	}

	allocatedIP, err := m.allocator.Allocate(node.NodeID)
	if err != nil {
		return fmt.Errorf("allocate virtual IP for %s: %w", node.NodeID, err)
	}
	if err := m.store.SaveIPAllocation(node.NodeID, allocatedIP); err != nil {
		m.allocator.Release(node.NodeID)
		return fmt.Errorf("persist virtual IP for %s: %w", node.NodeID, err)
	}

	node.VirtualIP = allocatedIP.String()
	node.Subnet = m.subnet
	return nil
}

// ReleaseNode 释放节点虚拟 IP，节点注销或过期清理时调用。
func (m *VirtualNetworkManager) ReleaseNode(nodeID string) error {
	m.allocator.Release(nodeID)
	if err := m.store.DeleteIPAllocation(nodeID); err != nil {
		return fmt.Errorf("delete virtual IP allocation for %s: %w", nodeID, err)
	}
	return nil
}

// BuildConfig 生成节点可直接消费的虚拟网络和路由配置。
func (m *VirtualNetworkManager) BuildConfig(nodeID string) (*VirtualNetworkConfig, error) {
	allocatedIP, err := m.store.GetIPAllocation(nodeID)
	if err != nil {
		node, nodeErr := m.store.GetNode(nodeID)
		if nodeErr != nil {
			return nil, fmt.Errorf("get virtual IP for %s: %w", nodeID, err)
		}
		// 兼容历史数据：节点已注册但 ip_allocations 缺失时，在路由读取路径补齐分配。
		if assignErr := m.AssignNode(node); assignErr != nil {
			return nil, fmt.Errorf("repair virtual IP for %s: %w", nodeID, assignErr)
		}
		if saveErr := m.store.SaveNode(node); saveErr != nil {
			return nil, fmt.Errorf("persist repaired virtual IP for %s: %w", nodeID, saveErr)
		}
		allocatedIP, err = m.store.GetIPAllocation(nodeID)
		if err != nil {
			return nil, fmt.Errorf("get repaired virtual IP for %s: %w", nodeID, err)
		}
	}

	return &VirtualNetworkConfig{
		NodeID:    nodeID,
		VirtualIP: allocatedIP.String(),
		Subnet:    m.subnet,
		Gateway:   m.gateway,
		Interface: m.iface,
		MTU:       m.mtu,
		Routes: []VirtualNetworkRoute{
			{
				Destination: m.subnet,
				Gateway:     m.gateway,
				Interface:   m.iface,
				Metric:      m.routeMetric,
			},
		},
	}, nil
}

func (m *VirtualNetworkManager) restoreAllocations() error {
	allocations, err := m.store.ListIPAllocations()
	if err != nil {
		return fmt.Errorf("list virtual IP allocations: %w", err)
	}
	for nodeID, allocatedIP := range allocations {
		if err := m.allocator.Reserve(nodeID, allocatedIP); err != nil {
			return fmt.Errorf("restore virtual IP allocation %s=%s: %w", nodeID, allocatedIP, err)
		}
	}
	return nil
}
