package dashboard

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func newTestServer() *Server {
	cfg := DefaultServerConfig()
	cfg.Auth.SecretKey = "dashboard-test-secret"
	cfg.Auth.DefaultPass = "admin-test-password"
	return NewServer(cfg)
}

func doLogin(t *testing.T, s *Server) string {
	t.Helper()
	body, _ := json.Marshal(LoginRequest{Username: "admin", Password: "admin-test-password"})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", w.Code, w.Body.String())
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	data, _ := json.Marshal(resp.Data)
	var login LoginResponse
	json.Unmarshal(data, &login)
	return login.Token
}

func authReq(method, path, token string, body interface{}) *http.Request {
	var req *http.Request
	if body != nil {
		b, _ := json.Marshal(body)
		req = httptest.NewRequest(method, path, bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	return req
}

func TestDashboard_Login(t *testing.T) {
	s := newTestServer()
	token := doLogin(t, s)
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	t.Logf("Login token: %s...", token[:16])
}

func TestDashboard_LoginInvalid(t *testing.T) {
	s := newTestServer()
	body, _ := json.Marshal(LoginRequest{Username: "admin", Password: "wrong"})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestDashboard_RejectsInsecureRuntimeConfig(t *testing.T) {
	cfg := DefaultServerConfig()
	cfg.Auth.SecretKey = ""
	s := NewServer(cfg)
	if err := s.Start(); err == nil {
		t.Fatal("expected missing secret key error")
	}

	cfg.Auth.SecretKey = "dashboard-test-secret"
	cfg.Auth.DefaultPass = "admin"
	s = NewServer(cfg)
	if err := s.Start(); err == nil {
		t.Fatal("expected insecure default password error")
	}
}

func TestDashboard_AuthRequired(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest("GET", "/api/v1/nodes", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 without token, got %d", w.Code)
	}
}

func TestDashboard_NodeCRUD(t *testing.T) {
	s := newTestServer()
	token := doLogin(t, s)

	// Add node to store
	s.store.AddNode(&NodeStatus{
		NodeID: "node-1", Region: "us-east", NATType: "full_cone",
		Online: true, ConnectedAt: time.Now(), LastSeen: time.Now(),
	})

	// List nodes
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, authReq("GET", "/api/v1/nodes", token, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("list nodes: %d %s", w.Code, w.Body.String())
	}

	var resp APIResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if !resp.Success {
		t.Error("expected success response")
	}

	// Get specific node
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, authReq("GET", "/api/v1/nodes/node-1", token, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("get node: %d", w.Code)
	}

	// Get non-existent node
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, authReq("GET", "/api/v1/nodes/nonexistent", token, nil))
	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}

	// Delete node
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, authReq("DELETE", "/api/v1/nodes/node-1", token, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("delete node: %d", w.Code)
	}
}

func TestDashboard_ACLCRUD(t *testing.T) {
	s := newTestServer()
	token := doLogin(t, s)

	// Create ACL rule
	rule := ACLRuleView{
		ID: "rule-1", Source: "*", Target: "node-A",
		Action: "allow", Protocol: "tcp", Priority: 1, Enabled: true,
	}
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, authReq("POST", "/api/v1/acl", token, rule))
	if w.Code != http.StatusCreated {
		t.Fatalf("create ACL: %d %s", w.Code, w.Body.String())
	}

	// List ACL rules
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, authReq("GET", "/api/v1/acl", token, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("list ACL: %d", w.Code)
	}

	// Delete ACL rule
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, authReq("DELETE", "/api/v1/acl/rule-1", token, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("delete ACL: %d", w.Code)
	}
}

func TestDashboard_Alerts(t *testing.T) {
	s := newTestServer()
	token := doLogin(t, s)

	s.store.AddAlert(&Alert{
		ID: "alert-1", Level: "warning", Message: "Node high latency",
		NodeID: "node-1", CreatedAt: time.Now(),
	})

	// List alerts
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, authReq("GET", "/api/v1/alerts", token, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("list alerts: %d", w.Code)
	}

	// Ack alert
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, authReq("POST", "/api/v1/alerts/alert-1/ack", token, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("ack alert: %d", w.Code)
	}
}

func TestDashboard_Stats(t *testing.T) {
	s := newTestServer()
	token := doLogin(t, s)

	s.store.AddStats(&TrafficStats{
		NodeID: "node-1", RxBytes: 1024000, TxBytes: 2048000,
		RxBandwidth: 102400, TxBandwidth: 204800, Connections: 5,
		Timestamp: time.Now(),
	})

	// Get global stats
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, authReq("GET", "/api/v1/stats", token, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("get stats: %d", w.Code)
	}

	// Get node stats
	w = httptest.NewRecorder()
	s.Handler().ServeHTTP(w, authReq("GET", "/api/v1/stats/node-1", token, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("get node stats: %d", w.Code)
	}
}

func TestDashboard_Health(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("health: %d", w.Code)
	}
}

func TestDashboard_Users(t *testing.T) {
	s := newTestServer()
	token := doLogin(t, s)

	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, authReq("GET", "/api/v1/users", token, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("list users: %d", w.Code)
	}
}
