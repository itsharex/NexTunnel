package protocol

import (
	"encoding/binary"
	"io"
	"net"
	"sync"
)

// maxPayloadSize is the maximum allowed payload size (16 MB).
const maxPayloadSize = 16 * 1024 * 1024

// headerSize is the wire format header size: 1 byte type + 4 bytes length.
const headerSize = 5

// ReadMessage reads a single protocol message from the reader.
func ReadMessage(r io.Reader) (*Message, error) {
	var header [headerSize]byte
	if _, err := io.ReadFull(r, header[:]); err != nil {
		return nil, err
	}

	msgType := MsgType(header[0])
	length := binary.BigEndian.Uint32(header[1:5])

	if length > maxPayloadSize {
		return nil, ErrPayloadTooLarge
	}

	var payload []byte
	if length > 0 {
		payload = make([]byte, length)
		if _, err := io.ReadFull(r, payload); err != nil {
			return nil, err
		}
	}

	return &Message{Type: msgType, Payload: payload}, nil
}

// WriteMessage writes a single protocol message to the writer.
func WriteMessage(w io.Writer, msg *Message) error {
	length := uint32(len(msg.Payload))
	if length > maxPayloadSize {
		return ErrPayloadTooLarge
	}

	var header [headerSize]byte
	header[0] = byte(msg.Type)
	binary.BigEndian.PutUint32(header[1:5], length)

	if _, err := w.Write(header[:]); err != nil {
		return err
	}

	if length > 0 {
		if _, err := w.Write(msg.Payload); err != nil {
			return err
		}
	}

	return nil
}

// Conn wraps a net.Conn with thread-safe read and write operations.
type Conn struct {
	conn   net.Conn
	readMu sync.Mutex
	writeMu sync.Mutex
	closed bool
	closeMu sync.RWMutex
}

// NewConn wraps a net.Conn for protocol-level communication.
func NewConn(conn net.Conn) *Conn {
	return &Conn{conn: conn}
}

// Read reads the next protocol message from the connection.
func (c *Conn) Read() (*Message, error) {
	c.readMu.Lock()
	defer c.readMu.Unlock()

	c.closeMu.RLock()
	if c.closed {
		c.closeMu.RUnlock()
		return nil, ErrConnClosed
	}
	c.closeMu.RUnlock()

	return ReadMessage(c.conn)
}

// Write writes a protocol message to the connection.
func (c *Conn) Write(msg *Message) error {
	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.closeMu.RLock()
	if c.closed {
		c.closeMu.RUnlock()
		return ErrConnClosed
	}
	c.closeMu.RUnlock()

	return WriteMessage(c.conn, msg)
}

// Close closes the underlying connection.
func (c *Conn) Close() error {
	c.closeMu.Lock()
	if c.closed {
		c.closeMu.Unlock()
		return nil
	}
	c.closed = true
	c.closeMu.Unlock()

	return c.conn.Close()
}

// RemoteAddr returns the remote network address.
func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}

// Underlying returns the raw net.Conn for direct I/O after protocol handshake.
func (c *Conn) Underlying() net.Conn {
	return c.conn
}
