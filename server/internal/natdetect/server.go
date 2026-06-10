package natdetect

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/pion/logging"
	"github.com/pion/stun/v2"
	"github.com/pion/turn/v4"
)

// Server is a dual-IP STUN/TURN server for NAT type detection.
type Server struct {
	config  Config
	logger  *slog.Logger
	servers []*turn.Server
	conns   []net.PacketConn
}

// NewServer creates a new NAT detection server.
func NewServer(cfg Config, logger *slog.Logger) *Server {
	if cfg.Port == 0 {
		cfg.Port = 3478
	}
	if cfg.Realm == "" {
		cfg.Realm = "nextunnel.local"
	}
	if cfg.Timeout == 0 {
		cfg.Timeout = 5 * time.Second
	}
	return &Server{config: cfg, logger: logger}
}

// Start binds the UDP listeners on both primary and alternate addresses.
func (s *Server) Start() error {
	primaryAddr := net.JoinHostPort(s.config.PrimaryAddr, strconv.Itoa(s.config.Port))
	altAddr := net.JoinHostPort(s.config.AltAddr, strconv.Itoa(s.config.Port))

	primaryConn, err := net.ListenPacket("udp4", primaryAddr)
	if err != nil {
		return fmt.Errorf("listen on primary %s: %w", primaryAddr, err)
	}
	s.conns = append(s.conns, primaryConn)
	s.logger.Info("primary STUN listener started", "addr", primaryAddr)

	packetConnConfigs := []turn.PacketConnConfig{
		newPacketConnConfig(primaryConn, s.config.PrimaryAddr),
	}

	// 主地址为通配地址时已经覆盖备用地址，重复监听同端口会在 Linux 上触发 address already in use。
	if shouldBindAlternateAddress(s.config.PrimaryAddr, s.config.AltAddr) {
		altConn, err := net.ListenPacket("udp4", altAddr)
		if err != nil {
			s.Stop()
			return fmt.Errorf("listen on alternate %s: %w", altAddr, err)
		}
		s.conns = append(s.conns, altConn)
		packetConnConfigs = append(packetConnConfigs, newPacketConnConfig(altConn, s.config.AltAddr))
		s.logger.Info("alternate STUN listener started", "addr", altAddr)
	} else {
		s.logger.Warn("alternate STUN listener skipped because primary address covers it",
			"primary", primaryAddr,
			"alt", altAddr)
	}

	// Create TURN server config (STUN is built-in)
	turnCfg := turn.ServerConfig{
		Realm:             s.config.Realm,
		LoggerFactory:     logging.NewDefaultLoggerFactory(),
		PacketConnConfigs: packetConnConfigs,
	}

	server, err := turn.NewServer(turnCfg)
	if err != nil {
		s.Stop()
		return fmt.Errorf("create TURN server: %w", err)
	}
	s.servers = append(s.servers, server)

	s.logger.Info("NAT detection server started",
		"primary", primaryAddr,
		"alt", altAddr,
		"realm", s.config.Realm)

	return nil
}

// newPacketConnConfig 统一创建 TURN 监听配置，避免主/备地址配置结构重复。
func newPacketConnConfig(conn net.PacketConn, relayAddress string) turn.PacketConnConfig {
	return turn.PacketConnConfig{
		PacketConn: conn,
		RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{
			RelayAddress: net.ParseIP(relayAddress),
			Address:      relayAddress,
		},
	}
}

// shouldBindAlternateAddress 判断备用地址是否需要独立监听，避免通配地址重复占用同端口。
func shouldBindAlternateAddress(primaryAddr, altAddr string) bool {
	primaryAddr = strings.TrimSpace(primaryAddr)
	altAddr = strings.TrimSpace(altAddr)
	if primaryAddr == "" || altAddr == "" || primaryAddr == altAddr {
		return false
	}
	primaryIP := net.ParseIP(primaryAddr)
	altIP := net.ParseIP(altAddr)
	if primaryIP == nil || altIP == nil {
		return true
	}
	if primaryIP.Equal(altIP) {
		return false
	}
	return !primaryIP.IsUnspecified() && !altIP.IsUnspecified()
}

// Stop gracefully shuts down the NAT detection server.
func (s *Server) Stop() {
	for _, server := range s.servers {
		if err := server.Close(); err != nil {
			s.logger.Error("error closing TURN server", "error", err)
		}
	}
	for _, conn := range s.conns {
		conn.Close()
	}
	s.servers = nil
	s.conns = nil
	s.logger.Info("NAT detection server stopped")
}

// handleSTUNBinding processes a STUN binding request and sends back a response.
// This is a simplified handler for when we need more control than TURN provides.
func handleSTUNBinding(conn net.PacketConn, buf []byte, n int, srcAddr net.Addr, logger *slog.Logger) {
	msg := new(stun.Message)
	msg.Raw = append(msg.Raw[:0], buf[:n]...)
	if err := msg.Decode(); err != nil {
		logger.Debug("failed to decode STUN message", "error", err, "from", srcAddr)
		return
	}

	if msg.Type != stun.BindingRequest {
		return
	}

	// Build success response with XOR-MAPPED-ADDRESS
	resp, err := stun.Build(
		stun.BindingSuccess,
		&stun.XORMappedAddress{
			IP:   srcAddr.(*net.UDPAddr).IP,
			Port: srcAddr.(*net.UDPAddr).Port,
		},
		stun.NewTransactionIDSetter(msg.TransactionID),
	)
	if err != nil {
		logger.Error("failed to build STUN response", "error", err)
		return
	}

	if _, err := conn.WriteTo(resp.Raw, srcAddr); err != nil {
		logger.Error("failed to send STUN response", "error", err, "to", srcAddr)
	}
}

// Wait blocks until context is cancelled (useful for signal handling in main).
func (s *Server) Wait(ctx context.Context) {
	<-ctx.Done()
}
