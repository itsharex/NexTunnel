package protocol

import "errors"

var (
	// ErrPayloadTooLarge is returned when a message payload exceeds the maximum size.
	ErrPayloadTooLarge = errors.New("protocol: payload exceeds maximum size")

	// ErrUnknownMsgType is returned for unrecognized message type bytes.
	ErrUnknownMsgType = errors.New("protocol: unknown message type")

	// ErrConnClosed is returned when operating on a closed connection.
	ErrConnClosed = errors.New("protocol: connection closed")
)
