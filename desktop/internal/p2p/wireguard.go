package p2p

import (
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"log/slog"
	"net"
	"net/netip"
	"sync/atomic"
	"time"

	"golang.zx2c4.com/wireguard/conn"
	"golang.zx2c4.com/wireguard/device"
)

// WGConfig configures a WireGuard tunnel.
type WGConfig struct {
	PrivateKey    string
	PeerPublicKey string
	PresharedKey  string
	PeerAddr      string
	MTU           int
	Logger        *slog.Logger
}

// WGState represents the WireGuard tunnel state.
type WGState string

const (
	WGStateInitializing WGState = "initializing"
	WGStateConnected    WGState = "connected"
	WGStateDisconnected WGState = "disconnected"
	WGStateError        WGState = "error"
)

// WGTunnel wraps a wireguard-go device with the netTun virtual adapter.
type WGTunnel struct {
	config WGConfig
	dev    *device.Device
	tun    *netTun
	status atomic.Value
	logger *slog.Logger
}

// NewWGTunnel creates a new WireGuard tunnel.
func NewWGTunnel(cfg WGConfig) *WGTunnel {
	if cfg.MTU == 0 {
		cfg.MTU = 1420
	}
	if cfg.Logger == nil {
		cfg.Logger = slog.Default()
	}
	t := &WGTunnel{config: cfg, logger: cfg.Logger}
	t.status.Store(WGStateInitializing)
	return t
}

// Start initializes the WireGuard device with the given bind.
func (t *WGTunnel) Start(bind conn.Bind) error {
	t.tun = newNetTun(t.config.MTU)
	wgLogger := device.NewLogger(device.LogLevelError, "wireguard")
	t.dev = device.NewDevice(t.tun, bind, wgLogger)

	// WireGuard IPC expects hex-encoded keys
	privHex, err := b64ToHex(t.config.PrivateKey)
	if err != nil {
		return fmt.Errorf("invalid private key: %w", err)
	}
	pubHex, err := b64ToHex(t.config.PeerPublicKey)
	if err != nil {
		return fmt.Errorf("invalid peer public key: %w", err)
	}

	ipcConf := fmt.Sprintf("private_key=%s\npublic_key=%s\nallowed_ip=%s\npersistent_keepalive_interval=25\n",
		privHex, pubHex, t.config.PeerAddr)
	if t.config.PresharedKey != "" {
		pskHex, _ := b64ToHex(t.config.PresharedKey)
		ipcConf += fmt.Sprintf("preshared_key=%s\n", pskHex)
	}
	if err := t.dev.IpcSet(ipcConf); err != nil {
		t.dev.Close()
		t.status.Store(WGStateError)
		return fmt.Errorf("configure wireguard: %w", err)
	}
	if err := t.dev.Up(); err != nil {
		t.dev.Close()
		t.status.Store(WGStateError)
		return fmt.Errorf("wireguard up: %w", err)
	}
	t.status.Store(WGStateConnected)
	t.logger.Info("WireGuard tunnel started")
	return nil
}

// TUN returns the virtual TUN device for application-level packet I/O.
func (t *WGTunnel) TUN() *netTun { return t.tun }

// Close shuts down the WireGuard device.
func (t *WGTunnel) Close() error {
	if t.dev != nil {
		t.dev.Close()
	}
	t.status.Store(WGStateDisconnected)
	t.logger.Info("WireGuard tunnel closed")
	return nil
}

// Status returns the current tunnel state.
func (t *WGTunnel) Status() WGState {
	return t.status.Load().(WGState)
}

// netBind implements conn.Bind using an existing net.UDPConn.
type netBind struct {
	udpConn  *net.UDPConn
	endpoint *net.UDPAddr
	closed   atomic.Bool
}

func newNetBind(c *net.UDPConn, remoteAddr *net.UDPAddr) *netBind {
	return &netBind{udpConn: c, endpoint: remoteAddr}
}

func (b *netBind) Open(port uint16) ([]conn.ReceiveFunc, uint16, error) {
	localPort := uint16(b.udpConn.LocalAddr().(*net.UDPAddr).Port)
	fn := func(bufs [][]byte, sizes []int, eps []conn.Endpoint) (int, error) {
		if b.closed.Load() {
			return 0, net.ErrClosed
		}
		b.udpConn.SetReadDeadline(time.Now().Add(time.Second))
		n, _, err := b.udpConn.ReadFromUDP(bufs[0])
		if err != nil {
			if b.closed.Load() {
				return 0, net.ErrClosed
			}
			if ne, ok := err.(net.Error); ok && ne.Timeout() {
				return 0, nil
			}
			return 0, err
		}
		sizes[0] = n
		eps[0] = newNetEP(*b.endpoint)
		return 1, nil
	}
	return []conn.ReceiveFunc{fn}, localPort, nil
}

func (b *netBind) Close() error {
	b.closed.Store(true)
	return nil
}
func (b *netBind) SetMark(uint32) error { return nil }
func (b *netBind) BatchSize() int       { return 1 }

func (b *netBind) Send(bufs [][]byte, _ conn.Endpoint) error {
	for _, buf := range bufs {
		if _, err := b.udpConn.WriteToUDP(buf, b.endpoint); err != nil {
			return err
		}
	}
	return nil
}

func (b *netBind) ParseEndpoint(s string) (conn.Endpoint, error) {
	addr, err := net.ResolveUDPAddr("udp", s)
	if err != nil {
		return nil, err
	}
	return newNetEP(*addr), nil
}

// netEP implements conn.Endpoint.
type netEP struct {
	addr    net.UDPAddr
	addrIP  netip.Addr
}

func newNetEP(addr net.UDPAddr) *netEP {
	ep := &netEP{addr: addr}
	if ip4 := addr.IP.To4(); ip4 != nil {
		ep.addrIP, _ = netip.AddrFromSlice(ip4)
	}
	return ep
}

func (e *netEP) ClearSrc()           {}
func (e *netEP) DstIP() netip.Addr   { return e.addrIP }
func (e *netEP) SrcIP() netip.Addr   { return netip.Addr{} }
func (e *netEP) DstToBytes() []byte  { return []byte(e.addr.String()) }
func (e *netEP) DstToString() string { return e.addr.String() }
func (e *netEP) SrcToString() string { return "" }

// b64ToHex converts a base64-encoded key to hex-encoded string for WireGuard IPC.
func b64ToHex(b64Key string) (string, error) {
	raw, err := base64.StdEncoding.DecodeString(b64Key)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(raw), nil
}
