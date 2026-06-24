package dashboard

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func newRBACTestServer(t *testing.T) *Server {
	t.Helper()
	cfg := DefaultServerConfig()
	cfg.Auth.SecretKey = "test-secret"
	cfg.Auth.DefaultAdmin = "admin"
	cfg.Auth.DefaultPass = "secure-password"
	return NewServer(cfg)
}

func loginAs(t *testing.T, srv *Server, username, password string) string {
	t.Helper()
	body := `{"username":"` + username + `","password":"` + password + `"}`
	req := httptest.NewRequest("POST", "/api/v1/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("login failed: %d %s", rr.Code, rr.Body.String())
	}
	var resp APIResponse
	json.Unmarshal(rr.Body.Bytes(), &resp)
	data, _ := json.Marshal(resp.Data)
	var login LoginResponse
	json.Unmarshal(data, &login)
	if login.Token == "" {
		t.Fatalf("login returned empty token: %s", rr.Body.String())
	}
	return login.Token
}

func TestRBAC_AdminHasFullAccess(t *testing.T) {
	srv := newRBACTestServer(t)
	token := loginAs(t, srv, "admin", "secure-password")

	// Admin can GET nodes
	req := httptest.NewRequest("GET", "/api/v1/nodes", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("admin GET /nodes: %d, want 200", rr.Code)
	}

	// Admin can POST acl
	req = httptest.NewRequest("POST", "/api/v1/acl", strings.NewReader(`{"id":"r1","source":"*","target":"*","action":"allow","protocol":"tcp","priority":1}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("admin POST /acl: %d, want 201", rr.Code)
	}
}

func TestRBAC_ViewerReadOnly(t *testing.T) {
	srv := newRBACTestServer(t)

	// Create a viewer user
	srv.auth.AddUserWithPassword(&User{ID: "viewer-1", Username: "viewer", Role: "viewer"}, "viewer-pass")
	token := loginAs(t, srv, "viewer", "viewer-pass")

	// Viewer can GET nodes
	req := httptest.NewRequest("GET", "/api/v1/nodes", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("viewer GET /nodes: %d, want 200", rr.Code)
	}

	// Viewer cannot POST acl
	req = httptest.NewRequest("POST", "/api/v1/acl", strings.NewReader(`{"id":"r2","source":"*","target":"*","action":"allow","protocol":"tcp","priority":1}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("viewer POST /acl: %d, want 403", rr.Code)
	}

	// Viewer cannot DELETE nodes
	req = httptest.NewRequest("DELETE", "/api/v1/nodes/some-node", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("viewer DELETE /nodes: %d, want 403", rr.Code)
	}

	// Viewer can inspect clients but cannot disconnect them
	req = httptest.NewRequest("GET", "/api/v1/clients", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("viewer GET /clients: %d, want 200", rr.Code)
	}

	req = httptest.NewRequest("GET", "/api/v1/config/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("viewer GET /config/status: %d, want 200", rr.Code)
	}

	req = httptest.NewRequest("DELETE", "/api/v1/clients/client-1", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("viewer DELETE /clients: %d, want 403", rr.Code)
	}
}

func TestRBAC_OperatorAccess(t *testing.T) {
	srv := newRBACTestServer(t)

	srv.auth.AddUserWithPassword(&User{ID: "op-1", Username: "operator", Role: "operator"}, "op-pass")
	token := loginAs(t, srv, "operator", "op-pass")

	// Operator can read and write nodes
	req := httptest.NewRequest("GET", "/api/v1/nodes", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("operator GET /nodes: %d, want 200", rr.Code)
	}

	// Operator cannot manage users
	req = httptest.NewRequest("GET", "/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("operator GET /users: %d, want 403", rr.Code)
	}

	// Operator cannot manage alert-rules
	req = httptest.NewRequest("POST", "/api/v1/alert-rules", strings.NewReader(`{"id":"ar1"}`))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusForbidden {
		t.Errorf("operator POST /alert-rules: %d, want 403", rr.Code)
	}
}

func TestRBAC_UnauthenticatedDenied(t *testing.T) {
	srv := newRBACTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/nodes", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("no auth GET /nodes: %d, want 401", rr.Code)
	}
}

func TestRBAC_LoginAndHealthOpen(t *testing.T) {
	srv := newRBACTestServer(t)

	// Health check open
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("health: %d, want 200", rr.Code)
	}
}

func TestRole_HasPermission(t *testing.T) {
	if !RoleAdmin.HasPermission("users", "write") {
		t.Error("admin should have users/write")
	}
	if RoleViewer.HasPermission("users", "write") {
		t.Error("viewer should not have users/write")
	}
	if !RoleViewer.HasPermission("nodes", "read") {
		t.Error("viewer should have nodes/read")
	}
	if RoleViewer.HasPermission("acl", "write") {
		t.Error("viewer should not have acl/write")
	}
}

func TestParseRole(t *testing.T) {
	if ParseRole("admin") != RoleAdmin {
		t.Error("admin")
	}
	if ParseRole("operator") != RoleOperator {
		t.Error("operator")
	}
	if ParseRole("viewer") != RoleViewer {
		t.Error("viewer")
	}
	if ParseRole("unknown") != RoleViewer {
		t.Error("unknown should default to viewer")
	}
}

func TestSecurityHeaders(t *testing.T) {
	srv := newRBACTestServer(t)

	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)

	if rr.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing X-Content-Type-Options")
	}
	if rr.Header().Get("X-Frame-Options") != "DENY" {
		t.Error("missing X-Frame-Options")
	}
	if rr.Header().Get("Referrer-Policy") != "strict-origin-when-cross-origin" {
		t.Error("missing Referrer-Policy")
	}
	// HSTS should not be present in non-TLS mode
	if rr.Header().Get("Strict-Transport-Security") != "" {
		t.Error("HSTS should not be set in non-TLS mode")
	}
}

func TestSecurityHeaders_TLS(t *testing.T) {
	handler := securityHeadersMiddleware(true, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	rr := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/", nil)
	handler.ServeHTTP(rr, req)

	if rr.Header().Get("Strict-Transport-Security") == "" {
		t.Error("HSTS should be set in TLS mode")
	}
}

func TestUserManagement_CreateAndDelete(t *testing.T) {
	srv := newRBACTestServer(t)
	adminToken := loginAs(t, srv, "admin", "secure-password")

	// Create user
	body := `{"username":"newuser","password":"newpass123","role":"viewer"}`
	req := httptest.NewRequest("POST", "/api/v1/users", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+adminToken)
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusCreated {
		t.Errorf("create user: %d, want 201: %s", rr.Code, rr.Body.String())
	}

	// Delete user
	req = httptest.NewRequest("DELETE", "/api/v1/users/newuser", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	rr = httptest.NewRecorder()
	srv.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("delete user: %d, want 200", rr.Code)
	}
}
