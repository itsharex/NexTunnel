package relay

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/nextunnel/pkg/protocol"
	"github.com/nextunnel/pkg/types"
)

// ClientConn manages one connected tunnel client's control connection.
type ClientConn struct {
	clientID string
	conn     *protocol.Conn
	server   *Server
	logger   *slog.Logger

	proxiesMu sync.Mutex
	proxies   map[string]*Proxy

	heartbeatTimer *time.Timer

	ctx    context.Context
	cancel context.CancelFunc
}

// NewClientConn creates a new client connection handler.
func NewClientConn(clientID string, conn *protocol.Conn, server *Server, logger *slog.Logger) *ClientConn {
	ctx, cancel := context.WithCancel(server.ctx)
	cc := &ClientConn{
		clientID: clientID,
		conn:     conn,
		server:   server,
		logger:   logger.With("client", clientID),
		proxies:  make(map[string]*Proxy),
		ctx:      ctx,
		cancel:   cancel,
	}
	return cc
}

// readLoop continuously reads messages from the control connection.
func (cc *ClientConn) readLoop() {
	defer cc.cleanup()

	// Reset heartbeat timer on each message
	cc.resetHeartbeat()

	for {
		msg, err := cc.conn.Read()
		if err != nil {
			select {
			case <-cc.ctx.Done():
				return
			default:
				cc.logger.Error("control conn read error", "error", err)
				return
			}
		}

		cc.resetHeartbeat()

		switch msg.Type {
		case protocol.TypeNewProxy:
			cc.handleNewProxy(msg)
		case protocol.TypeCloseProxy:
			cc.handleCloseProxy(msg)
		case protocol.TypeHeartbeat:
			if err := cc.conn.Write(protocol.NewHeartbeatResp()); err != nil {
				cc.logger.Error("failed to send heartbeat response", "error", err)
				return
			}
		case protocol.TypeHeartbeatResp:
			// Timer already reset above
		default:
			cc.logger.Warn("unexpected message type on control conn", "type", msg.Type)
		}
	}
}

func (cc *ClientConn) handleNewProxy(msg *protocol.Message) {
	payload, err := msg.DecodePayload()
	if err != nil {
		cc.logger.Error("invalid NewProxy payload", "error", err)
		return
	}
	np := payload.(*protocol.NewProxyMessage)

	cc.proxiesMu.Lock()
	if len(cc.proxies) >= cc.server.config.MaxProxiesPerClient {
		cc.proxiesMu.Unlock()
		cc.sendProxyResp(np.ProxyName, false, 0, "max proxies exceeded")
		return
	}
	if _, exists := cc.proxies[np.ProxyName]; exists {
		cc.proxiesMu.Unlock()
		cc.sendProxyResp(np.ProxyName, false, 0, "proxy name already in use")
		return
	}
	cc.proxiesMu.Unlock()

	info := types.ProxyInfo{
		ProxyName:  np.ProxyName,
		ProxyType:  types.ProxyType(np.ProxyType),
		LocalAddr:  np.LocalAddr,
		RemotePort: np.RemotePort,
		Status:     types.ProxyStatusActive,
	}

	proxy := NewProxy(info, cc, cc.logger)
	if err := proxy.Start(cc.server.config.BindAddr); err != nil {
		cc.logger.Error("failed to start proxy", "proxy", np.ProxyName, "error", err)
		cc.sendProxyResp(np.ProxyName, false, 0, fmt.Sprintf("failed to listen: %v", err))
		return
	}

	cc.proxiesMu.Lock()
	cc.proxies[np.ProxyName] = proxy
	cc.proxiesMu.Unlock()

	// Register in server's proxy map
	cc.server.registerProxy(cc.clientID+"/"+np.ProxyName, proxy)

	cc.sendProxyResp(np.ProxyName, true, proxy.RemotePort(), "")
	cc.logger.Info("proxy registered", "proxy", np.ProxyName, "remotePort", proxy.RemotePort())
}

func (cc *ClientConn) sendProxyResp(name string, success bool, port uint16, errMsg string) {
	msg, err := protocol.NewNewProxyRespMessage(name, success, port, errMsg)
	if err != nil {
		cc.logger.Error("failed to create NewProxyResp", "error", err)
		return
	}
	if err := cc.conn.Write(msg); err != nil {
		cc.logger.Error("failed to send NewProxyResp", "error", err)
	}
}

func (cc *ClientConn) handleCloseProxy(msg *protocol.Message) {
	payload, err := msg.DecodePayload()
	if err != nil {
		cc.logger.Error("invalid CloseProxy payload", "error", err)
		return
	}
	cp := payload.(*protocol.CloseProxyMessage)

	cc.proxiesMu.Lock()
	proxy, ok := cc.proxies[cp.ProxyName]
	if ok {
		delete(cc.proxies, cp.ProxyName)
	}
	cc.proxiesMu.Unlock()

	if ok {
		proxy.Stop()
		cc.server.unregisterProxy(cc.clientID + "/" + cp.ProxyName)
		cc.logger.Info("proxy removed", "proxy", cp.ProxyName)
	}
}

func (cc *ClientConn) sendStartWorkConn(proxyName, sessionID string) error {
	msg, err := protocol.NewStartWorkConnMessage(proxyName, sessionID)
	if err != nil {
		return err
	}
	return cc.conn.Write(msg)
}

func (cc *ClientConn) resetHeartbeat() {
	if cc.heartbeatTimer != nil {
		cc.heartbeatTimer.Stop()
	}
	cc.heartbeatTimer = time.AfterFunc(cc.server.config.HeartbeatTimeout, func() {
		cc.logger.Warn("heartbeat timeout, closing connection")
		cc.cancel()
		cc.conn.Close()
	})
}

// getProxy returns the proxy for the given name, or nil.
func (cc *ClientConn) getProxy(name string) *Proxy {
	cc.proxiesMu.Lock()
	defer cc.proxiesMu.Unlock()
	return cc.proxies[name]
}

// cleanup is called when the control connection is lost.
func (cc *ClientConn) cleanup() {
	cc.logger.Info("client connection cleanup")

	if cc.heartbeatTimer != nil {
		cc.heartbeatTimer.Stop()
	}

	cc.cancel()

	cc.proxiesMu.Lock()
	proxies := make(map[string]*Proxy, len(cc.proxies))
	for k, v := range cc.proxies {
		proxies[k] = v
	}
	cc.proxies = make(map[string]*Proxy)
	cc.proxiesMu.Unlock()

	for name, proxy := range proxies {
		proxy.Stop()
		cc.server.unregisterProxy(cc.clientID + "/" + name)
	}

	cc.server.removeClient(cc.clientID)
	cc.conn.Close()
}
