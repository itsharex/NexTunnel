package p2p

import (
	"fmt"
	"net"
	"os"
	"runtime"
)

// TUNDevice abstracts a TUN network device across platforms.
// Implementations can be:
//   - UserspaceTUN: channel-based MVP for testing (current netTun)
//   - Windows: Wintun driver (wintun.dll)
//   - macOS: utun kernel interface
//   - Linux: /dev/net/tun ioctl
//
// This interface provides simple packet-level Read/Write,
// while the wireguard-go tun.Device interface uses batch operations.
// Implementations should satisfy both interfaces.
type TUNDevice interface {
	// ReadPacket reads a single packet from the TUN device.
	ReadPacket(buf []byte) (int, error)

	// WritePacket writes a single packet to the TUN device.
	WritePacket(buf []byte) (int, error)

	// Close shuts down the TUN device.
	Close() error

	// MTU returns the Maximum Transmission Unit.
	MTU() (int, error)

	// Name returns the device name (e.g., "utun3", "wg0").
	Name() (string, error)

	// LocalAddr returns the local IP address assigned to this TUN device.
	LocalAddr() net.IP

	// PeerAddr returns the peer IP address for point-to-point links.
	PeerAddr() net.IP
}

// TUNConfig configures a TUN device.
type TUNConfig struct {
	// Name is the desired device name (platform-specific).
	Name string

	// MTU is the Maximum Transmission Unit. Default 1420.
	MTU int

	// LocalIP is the IP address assigned to the local TUN endpoint.
	LocalIP net.IP

	// PeerIP is the IP address of the remote peer.
	PeerIP net.IP

	// Subnet is the virtual network CIDR (e.g., "10.7.0.0/24").
	Subnet *net.IPNet
}

// DefaultTUNConfig returns a TUNConfig with sensible defaults.
func DefaultTUNConfig() TUNConfig {
	_, subnet, _ := net.ParseCIDR("10.7.0.0/24")
	return TUNConfig{
		Name:    "nextunnel0",
		MTU:     1420,
		LocalIP: net.ParseIP("10.7.0.1"),
		PeerIP:  net.ParseIP("10.7.0.2"),
		Subnet:  subnet,
	}
}

// PlatformCapabilities describes what the current platform supports.
type PlatformCapabilities struct {
	// HasKernelTUN indicates if the platform has kernel TUN support.
	HasKernelTUN bool

	// HasUserspaceNetstack indicates if a userspace TCP/IP stack is available.
	HasUserspaceNetstack bool

	// NeedsAdminPrivilege indicates if TUN creation requires elevated privileges.
	NeedsAdminPrivilege bool

	// PlatformName is the OS name (e.g., "windows", "darwin", "linux").
	PlatformName string

	// ProductionMode marks the highest supported production data-plane mode.
	ProductionMode string

	// KernelTUNReady indicates if real system TUN can be used immediately.
	KernelTUNReady bool

	// UserspaceModeAllowed indicates if the current runtime can fall back to userspace-only mode.
	UserspaceModeAllowed bool

	// BlockingIssues lists missing production prerequisites.
	BlockingIssues []PlatformIssue

	// DegradedFeatures lists features that can still run but are not production TUN.
	DegradedFeatures []PlatformIssue

	// RecommendedActions gives operator-facing remediation steps.
	RecommendedActions []string

	// EnvironmentHints gives production deployment options when host prerequisites are missing.
	EnvironmentHints []string
}

// CreateTUN attempts to create a kernel TUN device. If that fails,
// it falls back to a userspace netTun channel-based device.
func CreateTUN(cfg TUNConfig) (TUNDevice, error) {
	dev, err := createKernelTUN(cfg)
	if err == nil {
		return dev, nil
	}
	// Fall back to userspace netTun
	return newNetTun(cfg.MTU), nil
}

// CreateKernelTUN 创建真实内核 TUN 设备；生产验证必须使用它，避免 netTun 回退掩盖驱动或权限问题。
func CreateKernelTUN(cfg TUNConfig) (TUNDevice, error) {
	dev, err := createKernelTUN(cfg)
	if err != nil {
		return nil, fmt.Errorf("create kernel tun on %s: %w", runtime.GOOS, err)
	}
	return dev, nil
}

// hasKernelTUNSupport checks if the current platform has kernel TUN available.
func hasKernelTUNSupport() bool {
	switch runtime.GOOS {
	case "linux":
		_, err := os.Stat("/dev/net/tun")
		return err == nil
	case "darwin", "windows":
		return true // utun and Wintun are always potentially available
	default:
		return false
	}
}
