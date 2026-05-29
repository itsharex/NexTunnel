package protocol

import (
	"bytes"
	"io"
	"net"
	"sync"
	"testing"
)

func TestReadWriteMessage_RoundTrip(t *testing.T) {
	tests := []struct {
		name string
		msg  *Message
	}{
		{
			name: "auth",
			msg:  must(NewAuthMessage("client-123")),
		},
		{
			name: "auth_resp_success",
			msg:  must(NewAuthRespMessage(true, "")),
		},
		{
			name: "auth_resp_error",
			msg:  must(NewAuthRespMessage(false, "invalid token")),
		},
		{
			name: "new_proxy",
			msg:  must(NewNewProxyMessage("web", "tcp", "127.0.0.1:3000", 8080)),
		},
		{
			name: "new_proxy_resp",
			msg:  must(NewNewProxyRespMessage("web", true, 8080, "")),
		},
		{
			name: "close_proxy",
			msg:  must(NewCloseProxyMessage("web")),
		},
		{
			name: "start_work_conn",
			msg:  must(NewStartWorkConnMessage("web", "sess-456")),
		},
		{
			name: "work_conn",
			msg:  must(NewWorkConnMessage("web", "sess-456")),
		},
		{
			name: "heartbeat",
			msg:  NewHeartbeat(),
		},
		{
			name: "heartbeat_resp",
			msg:  NewHeartbeatResp(),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := WriteMessage(&buf, tt.msg); err != nil {
				t.Fatalf("WriteMessage: %v", err)
			}

			got, err := ReadMessage(&buf)
			if err != nil {
				t.Fatalf("ReadMessage: %v", err)
			}

			if got.Type != tt.msg.Type {
				t.Errorf("Type = %v, want %v", got.Type, tt.msg.Type)
			}
			if !bytes.Equal(got.Payload, tt.msg.Payload) {
				t.Errorf("Payload = %s, want %s", got.Payload, tt.msg.Payload)
			}
		})
	}
}

func TestDecodePayload(t *testing.T) {
	t.Run("auth", func(t *testing.T) {
		msg := must(NewAuthMessage("c1"))
		v, err := msg.DecodePayload()
		if err != nil {
			t.Fatal(err)
		}
		auth := v.(*AuthMessage)
		if auth.ClientID != "c1" || auth.Version != ProtocolVersion {
			t.Errorf("got %+v", auth)
		}
	})

	t.Run("new_proxy", func(t *testing.T) {
		msg := must(NewNewProxyMessage("web", "tcp", "127.0.0.1:3000", 8080))
		v, err := msg.DecodePayload()
		if err != nil {
			t.Fatal(err)
		}
		np := v.(*NewProxyMessage)
		if np.ProxyName != "web" || np.ProxyType != "tcp" || np.LocalAddr != "127.0.0.1:3000" || np.RemotePort != 8080 {
			t.Errorf("got %+v", np)
		}
	})

	t.Run("heartbeat_empty", func(t *testing.T) {
		msg := NewHeartbeat()
		v, err := msg.DecodePayload()
		if err != nil {
			t.Fatal(err)
		}
		if v != nil {
			t.Errorf("expected nil payload, got %v", v)
		}
	})
}

func TestReadMessage_TruncatedHeader(t *testing.T) {
	buf := bytes.NewBuffer([]byte{0x01, 0x00}) // only 2 bytes of header
	_, err := ReadMessage(buf)
	if err == nil {
		t.Fatal("expected error for truncated header")
	}
}

func TestReadMessage_PayloadTooLarge(t *testing.T) {
	// Build a header claiming 17 MB payload
	var header [5]byte
	header[0] = byte(TypeHeartbeat)
	header[1] = 0x01 // 17 MB = 0x01100000
	header[2] = 0x10
	header[3] = 0x00
	header[4] = 0x00
	buf := bytes.NewBuffer(header[:])
	_, err := ReadMessage(buf)
	if err != ErrPayloadTooLarge {
		t.Fatalf("expected ErrPayloadTooLarge, got %v", err)
	}
}

func TestReadMessage_EmptyPayload(t *testing.T) {
	msg := NewHeartbeat()
	var buf bytes.Buffer
	if err := WriteMessage(&buf, msg); err != nil {
		t.Fatal(err)
	}
	got, err := ReadMessage(&buf)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Payload) != 0 {
		t.Errorf("expected empty payload, got %d bytes", len(got.Payload))
	}
}

func TestWriteMessage_PayloadTooLarge(t *testing.T) {
	msg := &Message{
		Type:    TypeHeartbeat,
		Payload: make([]byte, maxPayloadSize+1),
	}
	var buf bytes.Buffer
	err := WriteMessage(&buf, msg)
	if err != ErrPayloadTooLarge {
		t.Fatalf("expected ErrPayloadTooLarge, got %v", err)
	}
}

func TestReadMessage_TruncatedPayload(t *testing.T) {
	// Header says 100 bytes but we only provide 10
	var header [5]byte
	header[0] = byte(TypeAuth)
	header[1] = 0x00
	header[2] = 0x00
	header[3] = 0x00
	header[4] = 100
	buf := bytes.NewBuffer(header[:])
	buf.Write(make([]byte, 10)) // only 10 bytes of the claimed 100
	_, err := ReadMessage(buf)
	if err == nil {
		t.Fatal("expected error for truncated payload")
	}
}

func TestReadMessage_EmptyReader(t *testing.T) {
	buf := bytes.NewBuffer(nil)
	_, err := ReadMessage(buf)
	if err != io.EOF {
		t.Fatalf("expected io.EOF, got %v", err)
	}
}

func TestConn_ConcurrentReadWrite(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()
	defer client.Close()

	serverConn := NewConn(server)
	clientConn := NewConn(client)

	const numMessages = 100
	var wg sync.WaitGroup

	// Writer goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numMessages; i++ {
			msg := NewHeartbeat()
			if err := clientConn.Write(msg); err != nil {
				t.Errorf("Write %d: %v", i, err)
				return
			}
		}
	}()

	// Reader goroutine
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < numMessages; i++ {
			msg, err := serverConn.Read()
			if err != nil {
				t.Errorf("Read %d: %v", i, err)
				return
			}
			if msg.Type != TypeHeartbeat {
				t.Errorf("Read %d: type = %v, want Heartbeat", i, msg.Type)
			}
		}
	}()

	wg.Wait()
}

func TestConn_Close(t *testing.T) {
	server, client := net.Pipe()
	defer server.Close()

	conn := NewConn(client)
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}

	// Double close should not panic
	if err := conn.Close(); err != nil {
		t.Fatal(err)
	}

	// Read after close should return error
	_, err := conn.Read()
	if err != ErrConnClosed {
		t.Errorf("expected ErrConnClosed, got %v", err)
	}

	// Write after close should return error
	err = conn.Write(NewHeartbeat())
	if err != ErrConnClosed {
		t.Errorf("expected ErrConnClosed, got %v", err)
	}
}

func must(msg *Message, err error) *Message {
	if err != nil {
		panic(err)
	}
	return msg
}
