// Package protocol defines the wire protocol for NexTunnel control channel communication.
package protocol

import "encoding/json"

// MsgType identifies the type of a protocol message.
type MsgType uint8

const (
	TypeAuth           MsgType = 0x01
	TypeAuthResp       MsgType = 0x02
	TypeNewProxy       MsgType = 0x03
	TypeNewProxyResp   MsgType = 0x04
	TypeCloseProxy     MsgType = 0x05
	TypeStartWorkConn  MsgType = 0x06
	TypeWorkConn       MsgType = 0x07
	TypeHeartbeat      MsgType = 0x08
	TypeHeartbeatResp  MsgType = 0x09
)

// ProtocolVersion is the current protocol version.
const ProtocolVersion uint8 = 1

// Message represents a raw protocol frame with type and payload.
type Message struct {
	Type    MsgType
	Payload []byte
}

// --- Typed payload structs ---

// AuthMessage is the payload for TypeAuth.
type AuthMessage struct {
	Version  uint8  `json:"version"`
	ClientID string `json:"client_id"`
}

// AuthRespMessage is the payload for TypeAuthResp.
type AuthRespMessage struct {
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// NewProxyMessage is the payload for TypeNewProxy.
type NewProxyMessage struct {
	ProxyName  string `json:"proxy_name"`
	ProxyType  string `json:"proxy_type"`
	LocalAddr  string `json:"local_addr"`
	RemotePort uint16 `json:"remote_port"`
	// HTTP-specific fields (ignored for TCP tunnels)
	Domain     string `json:"domain,omitempty"`
	HostHeader string `json:"host_header,omitempty"`
	UseHTTPS   bool   `json:"use_https,omitempty"`
}

// NewProxyRespMessage is the payload for TypeNewProxyResp.
type NewProxyRespMessage struct {
	ProxyName  string `json:"proxy_name"`
	Success    bool   `json:"success"`
	RemotePort uint16 `json:"remote_port"`
	Error      string `json:"error,omitempty"`
}

// CloseProxyMessage is the payload for TypeCloseProxy.
type CloseProxyMessage struct {
	ProxyName string `json:"proxy_name"`
}

// StartWorkConnMessage is the payload for TypeStartWorkConn.
type StartWorkConnMessage struct {
	ProxyName string `json:"proxy_name"`
	SessionID string `json:"session_id"`
}

// WorkConnMessage is the payload for TypeWorkConn.
type WorkConnMessage struct {
	ProxyName string `json:"proxy_name"`
	SessionID string `json:"session_id"`
}

// --- Factory constructors ---

// NewAuthMessage creates a TypeAuth message.
func NewAuthMessage(clientID string) (*Message, error) {
	return marshalMessage(TypeAuth, &AuthMessage{
		Version:  ProtocolVersion,
		ClientID: clientID,
	})
}

// NewAuthRespMessage creates a TypeAuthResp message.
func NewAuthRespMessage(success bool, errMsg string) (*Message, error) {
	return marshalMessage(TypeAuthResp, &AuthRespMessage{
		Success: success,
		Error:   errMsg,
	})
}

// NewNewProxyMessage creates a TypeNewProxy message.
func NewNewProxyMessage(name, proxyType, localAddr string, remotePort uint16) (*Message, error) {
	return marshalMessage(TypeNewProxy, &NewProxyMessage{
		ProxyName:  name,
		ProxyType:  proxyType,
		LocalAddr:  localAddr,
		RemotePort: remotePort,
	})
}

// NewHTTPProxyMessage creates a TypeNewProxy message with HTTP-specific fields.
func NewHTTPProxyMessage(name, localAddr string, remotePort uint16, domain, hostHeader string, useHTTPS bool) (*Message, error) {
	return marshalMessage(TypeNewProxy, &NewProxyMessage{
		ProxyName:  name,
		ProxyType:  "http",
		LocalAddr:  localAddr,
		RemotePort: remotePort,
		Domain:     domain,
		HostHeader: hostHeader,
		UseHTTPS:   useHTTPS,
	})
}

// NewNewProxyRespMessage creates a TypeNewProxyResp message.
func NewNewProxyRespMessage(name string, success bool, remotePort uint16, errMsg string) (*Message, error) {
	return marshalMessage(TypeNewProxyResp, &NewProxyRespMessage{
		ProxyName:  name,
		Success:    success,
		RemotePort: remotePort,
		Error:      errMsg,
	})
}

// NewCloseProxyMessage creates a TypeCloseProxy message.
func NewCloseProxyMessage(name string) (*Message, error) {
	return marshalMessage(TypeCloseProxy, &CloseProxyMessage{
		ProxyName: name,
	})
}

// NewStartWorkConnMessage creates a TypeStartWorkConn message.
func NewStartWorkConnMessage(proxyName, sessionID string) (*Message, error) {
	return marshalMessage(TypeStartWorkConn, &StartWorkConnMessage{
		ProxyName: proxyName,
		SessionID: sessionID,
	})
}

// NewWorkConnMessage creates a TypeWorkConn message.
func NewWorkConnMessage(proxyName, sessionID string) (*Message, error) {
	return marshalMessage(TypeWorkConn, &WorkConnMessage{
		ProxyName: proxyName,
		SessionID: sessionID,
	})
}

// NewHeartbeat creates a TypeHeartbeat message with empty payload.
func NewHeartbeat() *Message {
	return &Message{Type: TypeHeartbeat}
}

// NewHeartbeatResp creates a TypeHeartbeatResp message with empty payload.
func NewHeartbeatResp() *Message {
	return &Message{Type: TypeHeartbeatResp}
}

// DecodePayload decodes the message payload into the appropriate typed struct.
func (m *Message) DecodePayload() (any, error) {
	switch m.Type {
	case TypeAuth:
		var v AuthMessage
		return &v, json.Unmarshal(m.Payload, &v)
	case TypeAuthResp:
		var v AuthRespMessage
		return &v, json.Unmarshal(m.Payload, &v)
	case TypeNewProxy:
		var v NewProxyMessage
		return &v, json.Unmarshal(m.Payload, &v)
	case TypeNewProxyResp:
		var v NewProxyRespMessage
		return &v, json.Unmarshal(m.Payload, &v)
	case TypeCloseProxy:
		var v CloseProxyMessage
		return &v, json.Unmarshal(m.Payload, &v)
	case TypeStartWorkConn:
		var v StartWorkConnMessage
		return &v, json.Unmarshal(m.Payload, &v)
	case TypeWorkConn:
		var v WorkConnMessage
		return &v, json.Unmarshal(m.Payload, &v)
	case TypeHeartbeat, TypeHeartbeatResp:
		return nil, nil
	default:
		return nil, ErrUnknownMsgType
	}
}

func marshalMessage(t MsgType, v any) (*Message, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, err
	}
	return &Message{Type: t, Payload: data}, nil
}
