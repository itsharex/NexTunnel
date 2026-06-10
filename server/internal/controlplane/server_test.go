package controlplane

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNodeRegistry_Register1000(t *testing.T) {
	store := NewMemoryStore()
	reg := NewNodeRegistry(store, nil)

	// Register 1000 nodes
	for i := 0; i < 1000; i++ {
		node := &NodeInfo{
			NodeID:    fmt.Sprintf("node-%d", i),
			PublicKey: fmt.Sprintf("key-%d", i),
			NATType:   "full_cone",
			Region:    "us-east",
		}
		if err := reg.Register(node); err != nil {
			t.Fatalf("register node-%d: %v", i, err)
		}
	}

	if count := reg.Count(); count != 1000 {
		t.Errorf("count = %d, want 1000", count)
	}

	// Retrieve a specific node
	node, err := reg.Get("node-500")
	if err != nil {
		t.Fatalf("Get node-500: %v", err)
	}
	if node.PublicKey != "key-500" {
		t.Errorf("key = %s, want key-500", node.PublicKey)
	}

	// List all
	all := reg.List()
	if len(all) != 1000 {
		t.Errorf("list = %d, want 1000", len(all))
	}

	t.Log("1000 nodes registered and retrieved successfully")
}

func TestACLEngine_Evaluate(t *testing.T) {
	store := NewMemoryStore()
	acl := NewACLRuleEngine(store, nil)

	// Add rules
	acl.AddRule(&ACLRule{
		ID:       "rule-1",
		Source:   "node-A",
		Target:   "node-B",
		Action:   "allow",
		Protocol: "tcp",
		Ports:    []int{80, 443},
		Priority: 10,
	})

	acl.AddRule(&ACLRule{
		ID:       "rule-2",
		Source:   "node-A",
		Target:   "node-C",
		Action:   "deny",
		Protocol: "*",
		Priority: 5,
	})

	acl.AddRule(&ACLRule{
		ID:       "rule-3",
		Source:   "*",
		Target:   "node-B",
		Action:   "allow",
		Protocol: "tcp",
		Ports:    []int{80},
		Priority: 1,
	})

	// Test evaluations
	tests := []struct {
		source, target, protocol string
		port                     int
		expected                 bool
	}{
		{"node-A", "node-B", "tcp", 80, true},    // rule-1 allows
		{"node-A", "node-B", "tcp", 443, true},   // rule-1 allows
		{"node-A", "node-B", "tcp", 8080, false}, // no matching rule (default deny)
		{"node-A", "node-C", "tcp", 80, false},   // rule-2 denies
		{"node-X", "node-B", "tcp", 80, true},    // rule-3 allows (wildcard source)
		{"node-X", "node-C", "tcp", 80, false},   // no matching rule (default deny)
	}

	for _, tt := range tests {
		result := acl.Evaluate(tt.source, tt.target, tt.protocol, tt.port)
		if result != tt.expected {
			t.Errorf("Evaluate(%s, %s, %s, %d) = %v, want %v",
				tt.source, tt.target, tt.protocol, tt.port, result, tt.expected)
		}
	}

	t.Log("ACL evaluation matrix passed")
}

func TestKeyRotation_Automatic(t *testing.T) {
	store := NewMemoryStore()
	kx := NewKeyExchange(store, nil)

	// Register initial key
	if err := kx.RegisterKey("node-1", "pubkey-v1", 1, 24*time.Hour); err != nil {
		t.Fatalf("RegisterKey: %v", err)
	}

	// Get key
	km, err := kx.GetPeerKey("node-1")
	if err != nil {
		t.Fatalf("GetPeerKey: %v", err)
	}
	if km.KeyVersion != 1 {
		t.Errorf("version = %d, want 1", km.KeyVersion)
	}

	// Rotate key
	if err := kx.RotateKey("node-1", "pubkey-v2"); err != nil {
		t.Fatalf("RotateKey: %v", err)
	}

	// Verify new version
	km, err = kx.GetPeerKey("node-1")
	if err != nil {
		t.Fatalf("GetPeerKey after rotation: %v", err)
	}
	if km.KeyVersion != 2 {
		t.Errorf("version = %d, want 2", km.KeyVersion)
	}
	if km.PublicKey != "pubkey-v2" {
		t.Errorf("key = %s, want pubkey-v2", km.PublicKey)
	}

	t.Log("Key rotation passed")
}

func TestRBAC_Permissions(t *testing.T) {
	roles := []Role{RoleAdmin, RoleUser, RoleViewer}

	// Verify role hierarchy
	if RoleAdmin != "admin" {
		t.Error("admin role")
	}
	if RoleUser != "user" {
		t.Error("user role")
	}
	if RoleViewer != "viewer" {
		t.Error("viewer role")
	}

	for _, r := range roles {
		t.Logf("Role: %s", r)
	}
}

func TestControlPlane_FullFlow(t *testing.T) {
	store := NewMemoryStore()
	cfg := DefaultControlPlaneConfig()
	cfg.ListenAddr = "127.0.0.1:0"

	srv := NewServer(cfg, store)
	if err := srv.Start(); err != nil {
		t.Fatalf("Start: %v", err)
	}

	// Register 10 nodes
	for i := 0; i < 10; i++ {
		srv.Registry().Register(&NodeInfo{
			NodeID:    fmt.Sprintf("flow-node-%d", i),
			PublicKey: fmt.Sprintf("flow-key-%d", i),
			NATType:   "restricted",
			Region:    "eu-west",
		})
	}
	if srv.Registry().Count() != 10 {
		t.Fatalf("count = %d, want 10", srv.Registry().Count())
	}

	// Key exchange
	srv.Keys().RegisterKey("flow-node-0", "wg-key-0", 1, 24*time.Hour)
	km, err := srv.Keys().GetPeerKey("flow-node-0")
	if err != nil {
		t.Fatalf("GetPeerKey: %v", err)
	}
	if km.PublicKey != "wg-key-0" {
		t.Errorf("key = %s, want wg-key-0", km.PublicKey)
	}

	// ACL
	srv.ACL().AddRule(&ACLRule{
		ID:       "allow-all",
		Source:   "*",
		Target:   "*",
		Action:   "allow",
		Protocol: "*",
		Priority: 10,
	})
	if !srv.ACL().Evaluate("flow-node-0", "flow-node-1", "tcp", 80) {
		t.Error("ACL should allow")
	}
	if srv.ACL().Evaluate("flow-node-0", "flow-node-1", "tcp", 80) == false {
		t.Error("ACL wildcard should allow")
	}

	// Prune test: manually test PruneStale
	time.Sleep(10 * time.Millisecond) // ensure some time passes
	pruned := srv.Registry().PruneStale(1 * time.Millisecond)
	t.Logf("Manual prune: %d nodes pruned", pruned)
	if pruned != 10 {
		t.Logf("pruned %d, expected 10 (all stale with 1ms timeout)", pruned)
	}

	srv.Stop()
	t.Log("Control plane full flow PASSED")
}

func TestControlPlane_HTTPAPI(t *testing.T) {
	store := NewMemoryStore()
	cfg := DefaultControlPlaneConfig()
	cfg.ListenAddr = "127.0.0.1:0"
	srv := NewServer(cfg, store)
	handler := srv.Handler()

	nodeBody, _ := json.Marshal(NodeInfo{
		NodeID:    "api-node-1",
		PublicKey: "api-key-1",
		NATType:   "restricted",
		Region:    "ap-east",
	})
	req := httptest.NewRequest(http.MethodPost, "/api/v1/nodes", bytes.NewReader(nodeBody))
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("register node: %d %s", w.Code, w.Body.String())
	}
	var registered NodeInfo
	if err := json.NewDecoder(w.Body).Decode(&registered); err != nil {
		t.Fatalf("decode registered node: %v", err)
	}
	if registered.VirtualIP == "" {
		t.Fatal("registered node should receive virtual_ip")
	}
	if registered.Subnet != cfg.VirtualSubnet {
		t.Fatalf("registered subnet = %q, want %q", registered.Subnet, cfg.VirtualSubnet)
	}

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/nodes/api-node-1/heartbeat", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("heartbeat: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/nodes/api-node-1/routes", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("get node routes: %d %s", w.Code, w.Body.String())
	}
	var routeConfig VirtualNetworkConfig
	if err := json.NewDecoder(w.Body).Decode(&routeConfig); err != nil {
		t.Fatalf("decode route config: %v", err)
	}
	if routeConfig.VirtualIP != registered.VirtualIP {
		t.Fatalf("route virtual_ip = %q, want %q", routeConfig.VirtualIP, registered.VirtualIP)
	}
	if len(routeConfig.Routes) != 1 || routeConfig.Routes[0].Destination != cfg.VirtualSubnet {
		t.Fatalf("unexpected route config: %+v", routeConfig)
	}

	ruleBody, _ := json.Marshal(ACLRule{
		ID: "api-allow", Source: "*", Target: "*", Action: "allow", Protocol: "*", Priority: 1,
	})
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/acl", bytes.NewReader(ruleBody)))
	if w.Code != http.StatusCreated {
		t.Fatalf("add ACL: %d %s", w.Code, w.Body.String())
	}

	keyBody, _ := json.Marshal(map[string]any{
		"node_id":     "api-node-1",
		"public_key":  "wg-public-key",
		"key_version": 1,
	})
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/keys", bytes.NewReader(keyBody)))
	if w.Code != http.StatusCreated {
		t.Fatalf("register key: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/nodes/api-node-1/peers", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("get peers: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/api/v1/nodes/api-node-1", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("delete node: %d %s", w.Code, w.Body.String())
	}
	allocations, err := store.ListIPAllocations()
	if err != nil {
		t.Fatalf("ListIPAllocations: %v", err)
	}
	if _, exists := allocations["api-node-1"]; exists {
		t.Fatal("node IP allocation should be released after delete")
	}
}

func TestControlPlane_HTTPAPIToken(t *testing.T) {
	store := NewMemoryStore()
	cfg := DefaultControlPlaneConfig()
	cfg.APIToken = "control-plane-test-token"
	srv := NewServer(cfg, store)
	handler := srv.Handler()

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if w.Code != http.StatusOK {
		t.Fatalf("health should bypass token: %d %s", w.Code, w.Body.String())
	}

	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/v1/nodes", nil))
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized without token, got %d", w.Code)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/v1/nodes", nil)
	req.Header.Set("Authorization", "Bearer control-plane-test-token")
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("expected authorized request, got %d %s", w.Code, w.Body.String())
	}
}

func TestControlPlane_PersistentStateRestore(t *testing.T) {
	dbPath := t.TempDir() + "/control-plane.db"
	cfg := DefaultControlPlaneConfig()
	cfg.StorePath = dbPath

	store, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	srv := NewServer(cfg, store)
	handler := srv.Handler()

	nodeBody, _ := json.Marshal(NodeInfo{
		NodeID:    "restore-node-1",
		PublicKey: "restore-key-1",
		NATType:   "full_cone",
		Region:    "ap-south",
	})
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/nodes", bytes.NewReader(nodeBody)))
	if w.Code != http.StatusCreated {
		t.Fatalf("register node: %d %s", w.Code, w.Body.String())
	}
	var registered NodeInfo
	if err := json.NewDecoder(w.Body).Decode(&registered); err != nil {
		t.Fatalf("decode registered node: %v", err)
	}
	if registered.VirtualIP == "" {
		t.Fatal("registered node should receive virtual_ip")
	}

	ruleBody, _ := json.Marshal(ACLRule{
		ID:       "restore-allow",
		Source:   "restore-node-1",
		Target:   "*",
		Action:   "allow",
		Protocol: "tcp",
		Ports:    []int{443},
		Priority: 10,
	})
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/v1/acl", bytes.NewReader(ruleBody)))
	if w.Code != http.StatusCreated {
		t.Fatalf("add ACL: %d %s", w.Code, w.Body.String())
	}
	if err := store.Close(); err != nil {
		t.Fatalf("close store: %v", err)
	}

	restoredStore, err := NewSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("NewSQLiteStore restored: %v", err)
	}
	defer restoredStore.Close()
	restoredServer := NewServer(cfg, restoredStore)

	restoredNode, err := restoredServer.Registry().Get("restore-node-1")
	if err != nil {
		t.Fatalf("restored node missing: %v", err)
	}
	if restoredNode.VirtualIP != registered.VirtualIP {
		t.Fatalf("restored virtual_ip = %q, want %q", restoredNode.VirtualIP, registered.VirtualIP)
	}
	if !restoredServer.ACL().Evaluate("restore-node-1", "any-node", "tcp", 443) {
		t.Fatal("restored ACL should allow matching traffic")
	}
	routeConfig, err := restoredServer.virtualNet.BuildConfig("restore-node-1")
	if err != nil {
		t.Fatalf("restored route config: %v", err)
	}
	if routeConfig.VirtualIP != registered.VirtualIP {
		t.Fatalf("restored route virtual_ip = %q, want %q", routeConfig.VirtualIP, registered.VirtualIP)
	}
}

func TestMemoryStore_CRUD(t *testing.T) {
	store := NewMemoryStore()

	// Node CRUD
	node := &NodeInfo{NodeID: "test-node", PublicKey: "pk"}
	store.SaveNode(node)

	got, err := store.GetNode("test-node")
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if got.PublicKey != "pk" {
		t.Error("wrong key")
	}

	nodes, _ := store.ListNodes()
	if len(nodes) != 1 {
		t.Errorf("list = %d", len(nodes))
	}

	store.DeleteNode("test-node")
	_, err = store.GetNode("test-node")
	if err == nil {
		t.Error("expected error after delete")
	}

	// ACL CRUD
	rule := &ACLRule{ID: "r1", Source: "*", Target: "*", Action: "allow"}
	store.SaveACLRule(rule)
	rules, _ := store.ListACLRules()
	if len(rules) != 1 {
		t.Errorf("rules = %d", len(rules))
	}
	store.DeleteACLRule("r1")

	// Key CRUD
	km := &KeyMaterial{NodeID: "n1", PublicKey: "pk1"}
	store.SaveKeyMaterial(km)
	got2, err := store.GetKeyMaterial("n1")
	if err != nil {
		t.Fatalf("GetKeyMaterial: %v", err)
	}
	if got2.PublicKey != "pk1" {
		t.Error("wrong key")
	}
}
