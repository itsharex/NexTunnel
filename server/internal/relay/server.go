package relay

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"sync"

	"github.com/nextunnel/pkg/protocol"
)

// Server is the main relay server that manages client connections and proxy listeners.
type Server struct {
	config *Config
	logger *slog.Logger

	controlListener net.Listener

	clientsMu sync.RWMutex
	clients   map[string]*ClientConn

	proxiesMu sync.RWMutex
	proxies   map[string]*Proxy

	ctx    context.Context
	cancel context.CancelFunc
}

// NewServer creates a new relay server.
func NewServer(cfg *Config, logger *slog.Logger) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		config:  cfg,
		logger:  logger,
		clients: make(map[string]*ClientConn),
		proxies: make(map[string]*Proxy),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Run starts the relay server, listening for client connections.
func (s *Server) Run() error {
	addr := fmt.Sprintf("%s:%d", s.config.BindAddr, s.config.ControlPort)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen on %s: %w", addr, err)
	}
	s.controlListener = ln
	s.logger.Info("relay server started", "addr", addr)

	go s.acceptLoop()
	return nil
}

// Addr returns the control listener address, useful when port 0 is used.
func (s *Server) Addr() net.Addr {
	if s.controlListener != nil {
		return s.controlListener.Addr()
	}
	return nil
}

func (s *Server) acceptLoop() {
	for {
		conn, err := s.controlListener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
				s.logger.Error("accept error", "error", err)
				continue
			}
		}

		go s.handleNewConnection(conn)
	}
}

// handleNewConnection reads the first message to determine if this is a
// control connection (TypeAuth) or work connection (TypeWorkConn).
func (s *Server) handleNewConnection(conn net.Conn) {
	pconn := protocol.NewConn(conn)

	msg, err := pconn.Read()
	if err != nil {
		s.logger.Error("failed to read first message", "remote", conn.RemoteAddr(), "error", err)
		pconn.Close()
		return
	}

	switch msg.Type {
	case protocol.TypeAuth:
		s.handleControlConn(pconn, msg)
	case protocol.TypeWorkConn:
		s.handleWorkConn(pconn, msg)
	default:
		s.logger.Warn("unexpected first message type", "type", msg.Type, "remote", conn.RemoteAddr())
		pconn.Close()
	}
}

func (s *Server) handleControlConn(conn *protocol.Conn, authMsg *protocol.Message) {
	payload, err := authMsg.DecodePayload()
	if err != nil {
		s.logger.Error("invalid auth payload", "error", err)
		conn.Close()
		return
	}
	auth := payload.(*protocol.AuthMessage)

	if auth.Version != protocol.ProtocolVersion {
		resp, _ := protocol.NewAuthRespMessage(false, fmt.Sprintf("unsupported protocol version: %d", auth.Version))
		conn.Write(resp)
		conn.Close()
		return
	}

	clientID := auth.ClientID
	if clientID == "" {
		resp, _ := protocol.NewAuthRespMessage(false, "empty client ID")
		conn.Write(resp)
		conn.Close()
		return
	}

	// Check for duplicate client ID
	s.clientsMu.RLock()
	_, exists := s.clients[clientID]
	s.clientsMu.RUnlock()
	if exists {
		resp, _ := protocol.NewAuthRespMessage(false, "client ID already connected")
		conn.Write(resp)
		conn.Close()
		return
	}

	// Accept the client
	resp, _ := protocol.NewAuthRespMessage(true, "")
	if err := conn.Write(resp); err != nil {
		s.logger.Error("failed to send auth response", "error", err)
		conn.Close()
		return
	}

	cc := NewClientConn(clientID, conn, s, s.logger)
	s.clientsMu.Lock()
	s.clients[clientID] = cc
	s.clientsMu.Unlock()

	s.logger.Info("client connected", "client", clientID, "remote", conn.RemoteAddr())
	cc.readLoop() // blocks until disconnect
}

func (s *Server) handleWorkConn(conn *protocol.Conn, workMsg *protocol.Message) {
	payload, err := workMsg.DecodePayload()
	if err != nil {
		s.logger.Error("invalid WorkConn payload", "error", err)
		conn.Close()
		return
	}
	wc := payload.(*protocol.WorkConnMessage)

	// Find the client that owns this proxy
	s.clientsMu.RLock()
	// Search through all clients for the proxy
	var targetCC *ClientConn
	for _, cc := range s.clients {
		if cc.getProxy(wc.ProxyName) != nil {
			targetCC = cc
			break
		}
	}
	s.clientsMu.RUnlock()

	if targetCC == nil {
		s.logger.Warn("work conn for unknown proxy", "proxy", wc.ProxyName)
		conn.Close()
		return
	}

	proxy := targetCC.getProxy(wc.ProxyName)
	if proxy == nil {
		s.logger.Warn("proxy not found for work conn", "proxy", wc.ProxyName)
		conn.Close()
		return
	}

	if err := proxy.DeliverWorkConn(wc.SessionID, conn.Underlying()); err != nil {
		s.logger.Warn("failed to deliver work conn", "proxy", wc.ProxyName, "session", wc.SessionID, "error", err)
		conn.Close()
	}
}

func (s *Server) registerProxy(key string, proxy *Proxy) {
	s.proxiesMu.Lock()
	s.proxies[key] = proxy
	s.proxiesMu.Unlock()
}

func (s *Server) unregisterProxy(key string) {
	s.proxiesMu.Lock()
	delete(s.proxies, key)
	s.proxiesMu.Unlock()
}

func (s *Server) removeClient(clientID string) {
	s.clientsMu.Lock()
	delete(s.clients, clientID)
	s.clientsMu.Unlock()
	s.logger.Info("client disconnected", "client", clientID)
}

// Shutdown gracefully stops the server.
func (s *Server) Shutdown(ctx context.Context) error {
	s.logger.Info("shutting down relay server")
	s.cancel()

	if s.controlListener != nil {
		s.controlListener.Close()
	}

	// Close all client connections
	s.clientsMu.Lock()
	clients := make(map[string]*ClientConn, len(s.clients))
	for k, v := range s.clients {
		clients[k] = v
	}
	s.clientsMu.Unlock()

	for _, cc := range clients {
		cc.conn.Close()
	}

	// Stop all proxies
	s.proxiesMu.Lock()
	proxies := make(map[string]*Proxy, len(s.proxies))
	for k, v := range s.proxies {
		proxies[k] = v
	}
	s.proxiesMu.Unlock()

	for _, proxy := range proxies {
		proxy.Stop()
	}

	s.logger.Info("relay server stopped")
	return nil
}

// Done returns a channel that is closed when the server shuts down.
func (s *Server) Done() <-chan struct{} {
	return s.ctx.Done()
}

// GetClientCount returns the number of connected clients.
func (s *Server) GetClientCount() int {
	s.clientsMu.RLock()
	defer s.clientsMu.RUnlock()
	return len(s.clients)
}

// GetProxyCount returns the number of active proxies.
func (s *Server) GetProxyCount() int {
	s.proxiesMu.RLock()
	defer s.proxiesMu.RUnlock()
	return len(s.proxies)
}

// ServerStats holds aggregate server statistics.
type ServerStats struct {
	Clients    int
	Proxies    int
	BytesIn    int64
	BytesOut   int64
	Sessions   int64
}

// GetStats returns aggregate server statistics.
func (s *Server) GetStats() ServerStats {
	s.clientsMu.RLock()
	clients := len(s.clients)
	s.clientsMu.RUnlock()

	s.proxiesMu.RLock()
	var bytesIn, bytesOut, sessions int64
	for _, p := range s.proxies {
		bi, bo, sc := p.Stats()
		bytesIn += bi
		bytesOut += bo
		sessions += sc
	}
	proxies := len(s.proxies)
	s.proxiesMu.RUnlock()

	return ServerStats{
		Clients:  clients,
		Proxies:  proxies,
		BytesIn:  bytesIn,
		BytesOut: bytesOut,
		Sessions: sessions,
	}
}
