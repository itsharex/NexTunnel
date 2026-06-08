package p2p

import (
	"fmt"
	"net"
	"os"

	"golang.zx2c4.com/wireguard/tun"
)

// netTun implements both tun.Device (for wireguard-go) and TUNDevice (for platform abstraction).
// It is a userspace MVP/test channel that validates WireGuard device orchestration and packet flow.
// Production deployments should use platform-specific TUN (Wintun/utun/dev-net-tun) or a
// userspace netstack (gvisor/netstack) for full IP routing.
type netTun struct {
	events   chan tun.Event
	incoming chan []byte // packets from the application -> WireGuard reads them
	mtu      int
	closed   bool
	localIP  net.IP
	peerIP   net.IP
}

// newNetTun creates a virtual TUN device.
func newNetTun(mtu int) *netTun {
	if mtu == 0 {
		mtu = 1420
	}
	return &netTun{
		events:   make(chan tun.Event, 1),
		incoming: make(chan []byte, 256),
		mtu:      mtu,
		localIP:  net.ParseIP("10.7.0.1"),
		peerIP:   net.ParseIP("10.7.0.2"),
	}
}

func (t *netTun) File() *os.File { return nil }

// Read reads packets from the application side into WireGuard's buffer.
func (t *netTun) Read(bufs [][]byte, sizes []int, offset int) (int, error) {
	data, ok := <-t.incoming
	if !ok {
		return 0, os.ErrClosed
	}
	n := copy(bufs[0][offset:], data)
	sizes[0] = n
	return 1, nil
}

// Write receives packets from WireGuard and delivers them to the application.
// These are decrypted packets that WireGuard wants to send to the "network".
func (t *netTun) Write(bufs [][]byte, offset int) (int, error) {
	if t.closed {
		return 0, os.ErrClosed
	}
	count := 0
	for _, buf := range bufs {
		pkt := make([]byte, len(buf)-offset)
		copy(pkt, buf[offset:])
		select {
		case t.incoming <- pkt:
			count++
		default:
			// Drop if channel full
		}
	}
	return count, nil
}

func (t *netTun) MTU() (int, error)        { return t.mtu, nil }
func (t *netTun) Name() (string, error)    { return "netTun", nil }
func (t *netTun) Events() <-chan tun.Event { return t.events }

func (t *netTun) Close() error {
	if t.closed {
		return nil
	}
	t.closed = true
	close(t.events)
	close(t.incoming)
	return nil
}

func (t *netTun) BatchSize() int { return 1 }

// LocalAddr returns the local IP address of this TUN device.
func (t *netTun) LocalAddr() net.IP { return t.localIP }

// PeerAddr returns the peer IP address of this TUN device.
func (t *netTun) PeerAddr() net.IP { return t.peerIP }

// ReadPacket reads a single packet from the TUN device (TUNDevice interface).
func (t *netTun) ReadPacket(buf []byte) (int, error) {
	data, ok := <-t.incoming
	if !ok {
		return 0, os.ErrClosed
	}
	return copy(buf, data), nil
}

// WritePacket writes a single packet to the TUN device (TUNDevice interface).
func (t *netTun) WritePacket(buf []byte) (int, error) {
	if t.closed {
		return 0, os.ErrClosed
	}
	pkt := make([]byte, len(buf))
	copy(pkt, buf)
	select {
	case t.incoming <- pkt:
		return len(buf), nil
	default:
		return 0, fmt.Errorf("tun input buffer full")
	}
}

// WriteFromApp feeds a packet from the application into WireGuard.
func (t *netTun) WriteFromApp(data []byte) error {
	if t.closed {
		return fmt.Errorf("tun device closed")
	}
	pkt := make([]byte, len(data))
	copy(pkt, data)
	select {
	case t.incoming <- pkt:
		return nil
	default:
		return fmt.Errorf("tun input buffer full")
	}
}

// ReadToApp reads a decrypted packet from WireGuard destined for the application.
func (t *netTun) ReadToApp() ([]byte, error) {
	data, ok := <-t.incoming
	if !ok {
		return nil, os.ErrClosed
	}
	return data, nil
}
