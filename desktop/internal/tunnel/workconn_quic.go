package tunnel

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"time"

	"github.com/nextunnel/pkg/protocol"

	q "github.com/quic-go/quic-go"
)

// QUICWorkConnOpener opens work connections over QUIC streams.
// It maintains a persistent QUIC connection to the relay server
// and opens new streams for each work connection.
type QUICWorkConnOpener struct {
	ServerAddr string
	TLSConfig  *tls.Config
	ALPN       string

	conn   *q.Conn
	ctx    context.Context
	cancel context.CancelFunc
}

// NewQUICWorkConnOpener creates a new QUIC work connection opener.
// Call Connect() before using OpenWorkConn().
func NewQUICWorkConnOpener(serverAddr string, tlsCfg *tls.Config) *QUICWorkConnOpener {
	if tlsCfg == nil {
		tlsCfg = &tls.Config{MinVersion: tls.VersionTLS13}
	}
	tlsCfg = tlsCfg.Clone()
	if tlsCfg.MinVersion == 0 {
		tlsCfg.MinVersion = tls.VersionTLS13
	}
	if len(tlsCfg.NextProtos) == 0 {
		tlsCfg.NextProtos = []string{"nextunnel-quic-relay"}
	}
	ctx, cancel := context.WithCancel(context.Background())
	return &QUICWorkConnOpener{
		ServerAddr: serverAddr,
		TLSConfig:  tlsCfg,
		ALPN:       "nextunnel-quic-relay",
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Connect establishes the persistent QUIC connection to the relay server.
func (o *QUICWorkConnOpener) Connect(ctx context.Context) error {
	quicCfg := &q.Config{
		MaxIncomingStreams:    100,
		MaxIncomingUniStreams: 10,
		Allow0RTT:             true,
	}

	dialCtx, dialCancel := context.WithTimeout(ctx, 10*time.Second)
	defer dialCancel()

	conn, err := q.DialAddr(dialCtx, o.ServerAddr, o.TLSConfig, quicCfg)
	if err != nil {
		return fmt.Errorf("quic dial %s: %w", o.ServerAddr, err)
	}
	o.conn = conn
	return nil
}

// OpenWorkConn opens a new QUIC stream and sends the WorkConn handshake.
func (o *QUICWorkConnOpener) OpenWorkConn(proxyName, sessionID, authToken string) (net.Conn, error) {
	if o.conn == nil {
		return nil, fmt.Errorf("QUIC connection not established")
	}

	stream, err := o.conn.OpenStreamSync(o.ctx)
	if err != nil {
		return nil, fmt.Errorf("open quic stream: %w", err)
	}

	// Wrap the QUIC stream as a net.Conn for protocol framing
	streamConn := newQUICStreamNetConn(stream, o.conn.LocalAddr(), o.conn.RemoteAddr())
	pconn := protocol.NewConn(streamConn)

	workMsg, err := protocol.NewWorkConnMessageWithToken(proxyName, sessionID, authToken)
	if err != nil {
		stream.Close()
		return nil, fmt.Errorf("create work conn message: %w", err)
	}

	if err := pconn.Write(workMsg); err != nil {
		stream.Close()
		return nil, fmt.Errorf("send work conn message: %w", err)
	}

	// Return the stream-based net.Conn for raw data forwarding
	return streamConn, nil
}

// Close shuts down the QUIC connection.
func (o *QUICWorkConnOpener) Close() error {
	o.cancel()
	if o.conn != nil {
		return o.conn.CloseWithError(0, "closing")
	}
	return nil
}

// IsConnected returns whether the QUIC connection is active.
func (o *QUICWorkConnOpener) IsConnected() bool {
	return o.conn != nil
}

// --- QUIC stream → net.Conn adapter ---

type quicStreamNetConn struct {
	stream     *q.Stream
	localAddr  net.Addr
	remoteAddr net.Addr
}

func newQUICStreamNetConn(stream *q.Stream, localAddr, remoteAddr net.Addr) *quicStreamNetConn {
	return &quicStreamNetConn{
		stream:     stream,
		localAddr:  localAddr,
		remoteAddr: remoteAddr,
	}
}

func (c *quicStreamNetConn) Read(p []byte) (int, error)  { return c.stream.Read(p) }
func (c *quicStreamNetConn) Write(p []byte) (int, error) { return c.stream.Write(p) }
func (c *quicStreamNetConn) Close() error {
	c.stream.CancelRead(0)
	return c.stream.Close()
}
func (c *quicStreamNetConn) LocalAddr() net.Addr  { return c.localAddr }
func (c *quicStreamNetConn) RemoteAddr() net.Addr { return c.remoteAddr }
func (c *quicStreamNetConn) SetDeadline(t time.Time) error {
	if err := c.stream.SetReadDeadline(t); err != nil {
		return err
	}
	return c.stream.SetWriteDeadline(t)
}
func (c *quicStreamNetConn) SetReadDeadline(t time.Time) error  { return c.stream.SetReadDeadline(t) }
func (c *quicStreamNetConn) SetWriteDeadline(t time.Time) error { return c.stream.SetWriteDeadline(t) }
