package tunnel

import (
	"fmt"
	"net"
	"time"

	"github.com/nextunnel/pkg/protocol"
)

// WorkConnOpener abstracts how work connections are opened to the relay server.
// TCP and QUIC implementations allow the tunnel to use different transports
// for data plane connections while sharing the same control plane.
type WorkConnOpener interface {
	// OpenWorkConn opens a new work connection to the relay server.
	// The returned net.Conn is already framed with the WorkConn protocol message.
	OpenWorkConn(proxyName, sessionID, authToken string) (net.Conn, error)
}

// TCPWorkConnOpener opens work connections over plain TCP.
type TCPWorkConnOpener struct {
	ServerAddr string
}

// OpenWorkConn dials the relay server over TCP and sends the WorkConn handshake.
func (o *TCPWorkConnOpener) OpenWorkConn(proxyName, sessionID, authToken string) (net.Conn, error) {
	serverConn, err := net.DialTimeout("tcp", o.ServerAddr, 10*time.Second)
	if err != nil {
		return nil, fmt.Errorf("dial server for work conn: %w", err)
	}

	pconn := protocol.NewConn(serverConn)

	workMsg, err := protocol.NewWorkConnMessageWithToken(proxyName, sessionID, authToken)
	if err != nil {
		serverConn.Close()
		return nil, fmt.Errorf("create work conn message: %w", err)
	}

	if err := pconn.Write(workMsg); err != nil {
		serverConn.Close()
		return nil, fmt.Errorf("send work conn message: %w", err)
	}

	// Return the raw net.Conn (not protocol.Conn) because after the WorkConn
	// handshake, data flows as raw bytes.
	return serverConn, nil
}
