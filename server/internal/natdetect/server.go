package natdetect

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"time"

	"github.com/pion/stun/v2"
	"github.com/pion/turn/v4"
	"github.com/pion/logging"
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
	pionLogger := logging.NewDefaultLoggerFactory().NewLogger("natdetect")
	_ = pionLogger

	// Create UDP listeners on both addresses
	primaryAddr := fmt.Sprintf("%s:%d", s.config.PrimaryAddr, s.config.Port)
	altAddr := fmt.Sprintf("%s:%d", s.config.AltAddr, s.config.Port)

	primaryConn, err := net.ListenPacket("udp4", primaryAddr)
	if err != nil {
		return fmt.Errorf("listen on primary %s: %w", primaryAddr, err)
	}
	s.conns = append(s.conns, primaryConn)
	s.logger.Info("primary STUN listener started", "addr", primaryAddr)

	altConn, err := net.ListenPacket("udp4", altAddr)
	if err != nil {
		primaryConn.Close()
		return fmt.Errorf("listen on alternate %s: %w", altAddr, err)
	}
	s.conns = append(s.conns, altConn)
	s.logger.Info("alternate STUN listener started", "addr", altAddr)

	// Create TURN server config (STUN is built-in)
	turnCfg := turn.ServerConfig{
		Realm:       s.config.Realm,
		LoggerFactory: logging.NewDefaultLoggerFactory(),
		PacketConnConfigs: []turn.PacketConnConfig{
			{
				PacketConn:            primaryConn,
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{RelayAddress: net.ParseIP(s.config.PrimaryAddr), Address: s.config.PrimaryAddr},
			},
			{
				PacketConn:            altConn,
				RelayAddressGenerator: &turn.RelayAddressGeneratorStatic{RelayAddress: net.ParseIP(s.config.AltAddr), Address: s.config.AltAddr},
			},
		},
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
