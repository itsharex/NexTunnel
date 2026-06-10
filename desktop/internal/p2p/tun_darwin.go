//go:build darwin

package p2p

import (
	"fmt"
	"net"
	"os"
	"syscall"
	"unsafe"
)

// kernelTUN implements TUNDevice using macOS utun kernel interface.
type kernelTUN struct {
	file    *os.File
	name    string
	mtu     int
	localIP net.IP
	peerIP  net.IP
}

const (
	AF_SYSTEM    = 32
	SYS_CONTROL  = 2
	UTUN_OPT_IFNAME = 2
)

// createKernelTUN creates a real TUN device via utun on macOS.
func createKernelTUN(cfg TUNConfig) (TUNDevice, error) {
	fd, err := syscall.Socket(AF_SYSTEM, syscall.SOCK_DGRAM, SYS_CONTROL)
	if err != nil {
		return nil, fmt.Errorf("socket AF_SYSTEM: %w", err)
	}

	// Find utun control ID
	var ctlInfo struct {
		ctlID  uint32
		flags  uint32
		reserved [32]byte
		name   [96]byte
	}
	copy(ctlInfo.name[:], "com.apple.net.utun_control")

	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL, uintptr(fd), 0xC0644E03, uintptr(unsafe.Pointer(&ctlInfo)))
	if errno != 0 {
		syscall.Close(fd)
		return nil, fmt.Errorf("ioctl CTLIOCGINFO: %w", errno)
	}

	// Connect to utun
	var sc struct {
		scLen    uint8
		scFamily uint8
		ssPad1   [2]byte
		scID     uint32
		scUnit   uint32
		reserved [28]byte
	}
	sc.scLen = uint8(unsafe.Sizeof(sc))
	sc.scFamily = AF_SYSTEM
	sc.scID = ctlInfo.ctlID
	sc.scUnit = 0 // auto-assign

	_, _, errno = syscall.Syscall(syscall.SYS_CONNECT, uintptr(fd), uintptr(unsafe.Pointer(&sc)), uintptr(unsafe.Sizeof(sc)))
	if errno != 0 {
		syscall.Close(fd)
		return nil, fmt.Errorf("connect utun: %w", errno)
	}

	// Get device name
	var nameBuf [16]byte
	nameLen := uint32(len(nameBuf))
	_, _, errno = syscall.Syscall6(syscall.SYS_GETSOCKOPT, uintptr(fd), SYS_CONTROL, UTUN_OPT_IFNAME, uintptr(unsafe.Pointer(&nameBuf[0])), uintptr(unsafe.Pointer(&nameLen)), 0)
	if errno != 0 {
		syscall.Close(fd)
		return nil, fmt.Errorf("getsockopt utun name: %w", errno)
	}
	devName := string(nameBuf[:nameLen-1])

	file := os.NewFile(uintptr(fd), "/dev/"+devName)
	mtu := cfg.MTU
	if mtu == 0 {
		mtu = 1420
	}

	return &kernelTUN{
		file:    file,
		name:    devName,
		mtu:     mtu,
		localIP: cfg.LocalIP,
		peerIP:  cfg.PeerIP,
	}, nil
}

func (t *kernelTUN) ReadPacket(buf []byte) (int, error) {
	// macOS utun prepends a 4-byte family header
	hdr := make([]byte, 4)
	if _, err := t.file.Read(hdr); err != nil {
		return 0, err
	}
	return t.file.Read(buf)
}

func (t *kernelTUN) WritePacket(buf []byte) (int, error) {
	// macOS utun prepends a 4-byte family header
	version := buf[0] >> 4
	hdr := []byte{0, 0, 0, 0}
	if version == 4 {
		hdr[3] = 2 // AF_INET
	} else {
		hdr[3] = 30 // AF_INET6
	}
	if _, err := t.file.Write(hdr); err != nil {
		return 0, err
	}
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
