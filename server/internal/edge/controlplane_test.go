package edge

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestControlPlaneClient_RegisterNode(t *testing.T) {
	var received nodeRegistration
	var gotAuth string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/nodes" || r.Method != http.MethodPost {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		gotAuth = r.Header.Get("Authorization")
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &received)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	cfg := DefaultControlPlaneConfig()
	cfg.BaseURL = server.URL
	cfg.BearerToken = "test-token"
	client := NewControlPlaneClient(cfg)

	node, err := NewEdgeNode("edge-1", "10.0.0.1:4433", "us-east", RoleFull, 1000)
	if err != nil {
		t.Fatal(err)
	}

	ctx := context.Background()
	if err := client.RegisterNode(ctx, node); err != nil {
		t.Fatal(err)
	}

	if received.NodeID != "edge-1" {
		t.Fatalf("expected node ID edge-1, got %s", received.NodeID)
	}
	if received.Region != "us-east" {
		t.Fatalf("expected region us-east, got %s", received.Region)
	}
	if received.Metadata["addr"] != "10.0.0.1:4433" {
		t.Fatalf("expected metadata addr, got %q", received.Metadata["addr"])
	}
	if gotAuth != "Bearer test-token" {
		t.Fatalf("expected Bearer auth, got %q", gotAuth)
	}
}

func TestControlPlaneClient_RegisterNodeFails(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
	}))
	defer server.Close()

	cfg := DefaultControlPlaneConfig()
	cfg.BaseURL = server.URL
	client := NewControlPlaneClient(cfg)

	node, _ := NewEdgeNode("edge-1", "10.0.0.1:4433", "us-east", RoleFull, 1000)
	err := client.RegisterNode(context.Background(), node)
	if err == nil {
		t.Fatal("expected error for 409 response")
	}
}

func TestControlPlaneClient_SendHeartbeat(t *testing.T) {
	var gotPath string
	var gotPayload heartbeatPayload

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		json.Unmarshal(body, &gotPayload)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := DefaultControlPlaneConfig()
	cfg.BaseURL = server.URL
	client := NewControlPlaneClient(cfg)

	node, _ := NewEdgeNode("edge-1", "10.0.0.1:4433", "us-east", RoleFull, 1000)
	node.Latency = 15 * time.Millisecond

	if err := client.SendHeartbeat(context.Background(), node); err != nil {
		t.Fatal(err)
	}

	if gotPath != "/api/v1/nodes/edge-1/heartbeat" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotPayload.Status != "healthy" {
		t.Fatalf("expected healthy status, got %s", gotPayload.Status)
	}
}

func TestControlPlaneClient_HeartbeatLoop(t *testing.T) {
	var mu sync.Mutex
	count := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		count++
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := DefaultControlPlaneConfig()
	cfg.BaseURL = server.URL
	cfg.HeartbeatInterval = 50 * time.Millisecond
	client := NewControlPlaneClient(cfg)

	node, _ := NewEdgeNode("edge-hb", "10.0.0.1:4433", "eu-west", RoleRelay, 500)
	client.StartHeartbeatLoop(node)

	// Let 3 heartbeats pass
	time.Sleep(180 * time.Millisecond)
	client.StopHeartbeatLoop("edge-hb")

	mu.Lock()
	if count < 2 {
		t.Fatalf("expected at least 2 heartbeats, got %d", count)
	}
	mu.Unlock()
}

func TestControlPlaneClient_ConnectRegistry(t *testing.T) {
	var registered bool
	var mu sync.Mutex

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/v1/nodes" {
			mu.Lock()
			registered = true
			mu.Unlock()
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := DefaultControlPlaneConfig()
	cfg.BaseURL = server.URL
	cfg.HeartbeatInterval = 1 * time.Hour // don't fire heartbeats during test
	client := NewControlPlaneClient(cfg)

	reg := NewRegistry()
	client.ConnectRegistry(reg)

	node, _ := NewEdgeNode("edge-reg", "10.0.0.1:4433", "ap-southeast", RoleFull, 100)
	if err := reg.Register(node); err != nil {
		t.Fatal(err)
	}

	// Wait for async registration
	time.Sleep(100 * time.Millisecond)

	mu.Lock()
	if !registered {
		t.Fatal("expected node to be registered with Control Plane")
	}
	mu.Unlock()

	// Deregister should stop heartbeats
	reg.Deregister("edge-reg")
	client.StopAll()
}

func TestControlPlaneClient_StopAll(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	cfg := DefaultControlPlaneConfig()
	cfg.BaseURL = server.URL
	cfg.HeartbeatInterval = 50 * time.Millisecond
	client := NewControlPlaneClient(cfg)

	for i := 0; i < 3; i++ {
		node, _ := NewEdgeNode(
			"node-"+string(rune('a'+i)),
			"10.0.0.1:4433",
			"us-east",
			RoleRelay,
			100,
		)
		client.StartHeartbeatLoop(node)
	}

	client.StopAll()

	client.mu.Lock()
	remaining := len(client.heartbeats)
	client.mu.Unlock()

	if remaining != 0 {
		t.Fatalf("expected 0 heartbeats after StopAll, got %d", remaining)
	}
}

func TestDefaultControlPlaneConfig(t *testing.T) {
	cfg := DefaultControlPlaneConfig()
	if cfg.BaseURL == "" {
		t.Fatal("expected non-empty BaseURL")
	}
	if cfg.HeartbeatInterval == 0 {
		t.Fatal("expected non-zero HeartbeatInterval")
	}
	if cfg.Logger == nil {
		t.Fatal("expected non-nil Logger")
	}
	_ = slog.Default()
}
