//go:build windows

package p2p

import (
	"fmt"
	"net"

	"golang.zx2c4.com/wintun"
)

// kernelTUN implements TUNDevice using the Wintun driver on Windows.
type kernelTUN struct {
	adapter *wintun.Adapter
	session wintun.Session
	mtu     int
	localIP net.IP
	peerIP  net.IP
	name    string
}

// createKernelTUN creates a real TUN device using the Wintun driver on Windows.
func createKernelTUN(cfg TUNConfig) (TUNDevice, error) {
	name := cfg.Name
	if name == "" {
		name = "NexTunnel"
	}

	adapter, err := wintun.CreateAdapter(name, "NexTunnel", nil)
	if err != nil {
		return nil, fmt.Errorf("create wintun adapter: %w", err)
	}

	session, err := adapter.StartSession(0x400000) // 4MB ring buffer
	if err != nil {
		adapter.Close()
		return nil, fmt.Errorf("start wintun session: %w", err)
	}

	mtu := cfg.MTU
	if mtu == 0 {
		mtu = 1420
	}

	return &kernelTUN{
		adapter: adapter,
		session: session,
		mtu:     mtu,
		localIP: cfg.LocalIP,
		peerIP:  cfg.PeerIP,
		name:    name,
	}, nil
}

func (t *kernelTUN) ReadPacket(buf []byte) (int, error) {
	packet, err := t.session.ReceivePacket()
	if err != nil {
		return 0, err
	}
	n := copy(buf, packet)
	t.session.ReleaseReceivePacket(packet)
	return n, nil
}

func (t *kernelTUN) WritePacket(buf []byte) (int, error) {
	packet, err := t.session.AllocateSendPacket(len(buf))
	if err != nil {
		return 0, err
	}
	copy(packet, buf)
	t.session.SendPacket(packet)
	return len(buf), nil
}

func (t *kernelTUN) Close() error {
	t.session.End()
	t.adapter.Close()
	return nil
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
