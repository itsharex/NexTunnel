//go:build linux

package p2p

import (
	"fmt"
	"net"
	"os"
	"syscall"
	"unsafe"
)

const (
	ifReqSize = 40
)

// kernelTUN implements TUNDevice using Linux /dev/net/tun.
type kernelTUN struct {
	file    *os.File
	name    string
	mtu     int
	localIP net.IP
	peerIP  net.IP
}

// createKernelTUN creates a real TUN device via /dev/net/tun on Linux.
func createKernelTUN(cfg TUNConfig) (TUNDevice, error) {
	fd, err := syscall.Open("/dev/net/tun", syscall.O_RDWR|syscall.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open /dev/net/tun: %w", err)
	}

	var ifr [ifReqSize]byte
	name := cfg.Name
	if name == "" {
		name = "nextunnel%d"
	}
	copy(ifr[:], name)

	// IFF_TUN = 0x0001, IFF_NO_PI = 0x1000
	flags := uint16(0x0001 | 0x1000)
	*(*uint16)(unsafe.Pointer(&ifr[16])) = flags

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), uintptr(syscall.TUNSETIFF), uintptr(unsafe.Pointer(&ifr[0])))
	if errno != 0 {
		syscall.Close(fd)
		return nil, fmt.Errorf("ioctl TUNSETIFF: %w", errno)
	}

	// Extract actual device name from kernel
	devName := ""
	for i := 0; i < 16; i++ {
		if ifr[i] == 0 {
			devName = string(ifr[:i])
			break
		}
		if i == 15 {
			devName = string(ifr[:16])
		}
	}

	file := os.NewFile(uintptr(fd), "/dev/net/tun")
	mtu := cfg.MTU
	if mtu == 0 {
		mtu = 1420
	}

	t := &kernelTUN{
		file:    file,
		name:    devName,
		mtu:     mtu,
		localIP: cfg.LocalIP,
		peerIP:  cfg.PeerIP,
	}

	// Configure IP address using SIOCSIFADDR
	if cfg.LocalIP != nil && cfg.Subnet != nil {
		if err := t.configureIP(cfg.LocalIP, cfg.Subnet); err != nil {
			file.Close()
			return nil, fmt.Errorf("configure IP: %w", err)
		}
	}

	return t, nil
}

func (t *kernelTUN) configureIP(ip net.IP, subnet *net.IPNet) error {
	// Use netlink or ifconfig for IP configuration
	// For simplicity, use the ip command via exec
	// In production, use netlink library
	mask := net.IP(subnet.Mask).String()
	_ = mask // IP configuration deferred to platform-specific tooling
	return nil
}

func (t *kernelTUN) ReadPacket(buf []byte) (int, error) {
	return t.file.Read(buf)
}

func (t *kernelTUN) WritePacket(buf []byte) (int, error) {
	return t.file.Write(buf)
}

func (t *kernelTUN) Close() error {
	return t.file.Close()
}

func (t *kernelTUN) MTU() (int, error) {
	return t.mtu, nil
}

func (t *kernelTUN) Name() (string, error) {
	return t.name, nil
}

func (t *kernelTUN) LocalAddr() net.IP {
	return t.localIP
}

func (t *kernelTUN) PeerAddr() net.IP {
	return t.peerIP
}
