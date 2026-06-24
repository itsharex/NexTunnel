package relay

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net"
	"net/http"
	"testing"
	"time"

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

func TestRelay_AdminAPIListsAndDisconnectsClient(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BindAddr = "127.0.0.1"
	cfg.ControlPort = 0
	cfg.QUICPort = 0
	cfg.AdminListenAddr = "127.0.0.1:0"
	cfg.AdminToken = "admin-token"

	srv := NewServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := srv.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	defer srv.Shutdown(context.Background())

	conn, err := net.Dial("tcp", srv.Addr().String())
	if err != nil {
		t.Fatalf("dial relay: %v", err)
	}
	pconn := protocol.NewConn(conn)
	authMsg, err := protocol.NewAuthMessageWithToken("client-1", "")
	if err != nil {
		t.Fatalf("auth message: %v", err)
	}
	if err := pconn.Write(authMsg); err != nil {
		t.Fatalf("write auth: %v", err)
	}
	resp, err := pconn.Read()
	if err != nil {
		t.Fatalf("read auth response: %v", err)
	}
	payload, err := resp.DecodePayload()
	if err != nil {
		t.Fatalf("decode auth response: %v", err)
	}
	if authResp := payload.(*protocol.AuthRespMessage); !authResp.Success {
		t.Fatalf("auth rejected: %s", authResp.Error)
	}

	adminURL := "http://" + srv.AdminAddr().String() + "/api/v1/admin/clients"
	req, err := http.NewRequest(http.MethodGet, adminURL, nil)
	if err != nil {
		t.Fatalf("create admin request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer admin-token")
	httpResp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("list clients: %v", err)
	}
	defer httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		t.Fatalf("list status = %d", httpResp.StatusCode)
	}
	var clients []ClientSnapshot
	if err := json.NewDecoder(httpResp.Body).Decode(&clients); err != nil {
		t.Fatalf("decode clients: %v", err)
	}
	if len(clients) != 1 || clients[0].ClientID != "client-1" || clients[0].RemoteAddr == "" {
		t.Fatalf("unexpected clients: %+v", clients)
	}

	req, err = http.NewRequest(http.MethodDelete, adminURL+"/client-1", nil)
	if err != nil {
		t.Fatalf("create delete request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer admin-token")
	httpResp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("disconnect client: %v", err)
	}
	_ = httpResp.Body.Close()
	if httpResp.StatusCode != http.StatusOK {
		t.Fatalf("disconnect status = %d", httpResp.StatusCode)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if srv.GetClientCount() == 0 {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("client still connected after disconnect")
}

func TestRelay_AdminAPIRejectsInvalidToken(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BindAddr = "127.0.0.1"
	cfg.ControlPort = 0
	cfg.QUICPort = 0
	cfg.AdminListenAddr = "127.0.0.1:0"
	cfg.AdminToken = "admin-token"

	srv := NewServer(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := srv.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	defer srv.Shutdown(context.Background())

	resp, err := http.Get("http://" + srv.AdminAddr().String() + "/api/v1/admin/clients")
	if err != nil {
		t.Fatalf("request admin API: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", resp.StatusCode)
	}
}

func TestConfig_Validate_LocalhostNoToken(t *testing.T) {
	// Localhost bind without token should be allowed (development mode)
	cfg := DefaultConfig()
	cfg.BindAddr = "127.0.0.1"
	cfg.AuthToken = ""
	if err := cfg.Validate(); err != nil {
		t.Errorf("localhost without token should be allowed, got: %v", err)
	}
}

func TestConfig_Validate_NonLocalhostRequiresToken(t *testing.T) {
	// Non-localhost bind without token should fail
	cfg := DefaultConfig()
	cfg.BindAddr = "0.0.0.0"
	cfg.AuthToken = ""
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for non-localhost bind without auth-token")
	}
}

func TestConfig_Validate_NonLocalhostWithToken(t *testing.T) {
	// Non-localhost bind with token should pass
	cfg := DefaultConfig()
	cfg.BindAddr = "0.0.0.0"
	cfg.AuthToken = "my-secure-token"
	if err := cfg.Validate(); err != nil {
		t.Errorf("non-localhost with token should pass, got: %v", err)
	}
}

func TestConfig_Validate_RequireAuthFlag(t *testing.T) {
	// Explicit RequireAuth with empty token should fail even on localhost
	cfg := DefaultConfig()
	cfg.BindAddr = "127.0.0.1"
	cfg.AuthToken = ""
	cfg.RequireAuth = true
	if err := cfg.Validate(); err == nil {
		t.Error("expected error when RequireAuth=true but no token")
	}
}

func TestConfig_Validate_AdminTokenRequired(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BindAddr = "127.0.0.1"
	cfg.AdminListenAddr = "127.0.0.1:17001"
	if err := cfg.Validate(); err == nil {
		t.Error("expected error when admin-listen is configured without admin-token")
	}
}

func TestConfig_Validate_InvalidPort(t *testing.T) {
	cfg := DefaultConfig()
	cfg.BindAddr = "127.0.0.1"
	cfg.ControlPort = -1
	if err := cfg.Validate(); err == nil {
		t.Error("expected error for invalid port")
	}
}
