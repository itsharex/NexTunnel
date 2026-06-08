package p2p

import (
	"net"
	"os"
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
}

// CurrentPlatform returns the capabilities of the current platform.
// This is a stub; production code should detect at runtime.
func CurrentPlatform() PlatformCapabilities {
	return PlatformCapabilities{
		HasKernelTUN:         false, // netTun is userspace only
		HasUserspaceNetstack: true,  // netTun provides channel-based packet passing
		NeedsAdminPrivilege:  false,
		PlatformName:         detectPlatform(),
	}
}

func detectPlatform() string {
	// Use runtime.GOOS in production
	if _, err := os.Stat("/dev/net/tun"); err == nil {
		return "linux"
	}
	return "unknown"
}
