package relay

import (
	"context"
	"io"
	"log/slog"
	"net"
	"testing"

	"github.com/nextunnel/pkg/protocol"
)

func TestRelay_RejectsInvalidAuthToken(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BindAddr = "127.0.0.1"
	cfg.ControlPort = 0
	cfg.QUICPort = 0
	cfg.AuthToken = "expected-token"

	srv := NewServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := srv.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	defer srv.Shutdown(context.Background())

	conn, err := net.Dial("tcp", srv.Addr().String())
	if err != nil {
		t.Fatalf("dial relay: %v", err)
	}
	defer conn.Close()

	pconn := protocol.NewConn(conn)
	authMsg, err := protocol.NewAuthMessageWithToken("client-1", "wrong-token")
	if err != nil {
		t.Fatalf("auth message: %v", err)
	}
	if err := pconn.Write(authMsg); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	resp, err := pconn.Read()
	if err != nil {
		t.Fatalf("read response: %v", err)
	}
	payload, err := resp.DecodePayload()
	if err != nil {
		t.Fatalf("decode response: %v", err)
	}
	authResp := payload.(*protocol.AuthRespMessage)
	if authResp.Success {
		t.Fatal("expected auth rejection")
	}
}
