package quic

import (
	"net"

	q "github.com/quic-go/quic-go"
)

// StreamAdapter wraps a *quic.Stream to satisfy io.ReadWriteCloser
// and provides net.Addr methods for Transport interface compliance.
type StreamAdapter struct {
	stream     *q.Stream
	localAddr  net.Addr
	remoteAddr net.Addr
}

// NewStreamAdapter creates a new adapter around a QUIC stream.
func NewStreamAdapter(s *q.Stream, local, remote net.Addr) *StreamAdapter {
	return &StreamAdapter{
		stream:     s,
		localAddr:  local,
		remoteAddr: remote,
	}
}

// Read reads data from the QUIC stream.
func (a *StreamAdapter) Read(p []byte) (int, error) {
	return a.stream.Read(p)
}

// Write writes data to the QUIC stream.
func (a *StreamAdapter) Write(p []byte) (int, error) {
	return a.stream.Write(p)
}

// Close closes both directions of the QUIC stream.
func (a *StreamAdapter) Close() error {
	a.stream.CancelRead(0)
	return a.stream.Close()
}

// LocalAddr returns the local network address.
func (a *StreamAdapter) LocalAddr() net.Addr {
	return a.localAddr
}

// RemoteAddr returns the remote network address.
func (a *StreamAdapter) RemoteAddr() net.Addr {
	return a.remoteAddr
}
