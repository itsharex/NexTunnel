package controlplane

import (
	"fmt"
	"testing"
	"time"
)

func newTestSQLiteStore(t *testing.T) *SQLiteStore {
	t.Helper()
	store, err := NewSQLiteStore("") // in-memory
	if err != nil {
		t.Fatalf("NewSQLiteStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestSQLiteStore_NodeCRUD(t *testing.T) {
	store := newTestSQLiteStore(t)

	node := &NodeInfo{
		NodeID:      "node-1",
		PublicKey:   "pubkey-abc",
		NATType:     "full_cone",
		Region:      "us-east",
		Subnet:      "10.7.0.0/24",
		VirtualIP:   "10.7.0.2",
		Metadata:    map[string]string{"version": "1.0", "os": "linux"},
		ConnectedAt: time.Now().Truncate(time.Second),
		LastSeen:    time.Now().Truncate(time.Second),
	}

	// Save
	if err := store.SaveNode(node); err != nil {
		t.Fatalf("SaveNode: %v", err)
	}

	// Get
	got, err := store.GetNode("node-1")
	if err != nil {
		t.Fatalf("GetNode: %v", err)
	}
	if got.NodeID != node.NodeID {
		t.Errorf("NodeID = %q, want %q", got.NodeID, node.NodeID)
	}
	if got.PublicKey != node.PublicKey {
		t.Errorf("PublicKey = %q, want %q", got.PublicKey, node.PublicKey)
	}
	if got.Region != node.Region {
		t.Errorf("Region = %q, want %q", got.Region, node.Region)
	}
	if got.VirtualIP != node.VirtualIP {
		t.Errorf("VirtualIP = %q, want %q", got.VirtualIP, node.VirtualIP)
	}
	if got.Metadata["version"] != "1.0" {
		t.Errorf("Metadata[version] = %q, want %q", got.Metadata["version"], "1.0")
	}

	// List
	list, err := store.ListNodes()
	if err != nil {
		t.Fatalf("ListNodes: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("ListNodes count = %d, want 1", len(list))
	}

	// Upsert (update existing)
	node.NATType = "symmetric"
	if err := store.SaveNode(node); err != nil {
		t.Fatalf("SaveNode (upsert): %v", err)
	}
	got, _ = store.GetNode("node-1")
	if got.NATType != "symmetric" {
		t.Errorf("after upsert: NATType = %q, want %q", got.NATType, "symmetric")
	}

	// Delete
	if err := store.DeleteNode("node-1"); err != nil {
		t.Fatalf("DeleteNode: %v", err)
	}
	_, err = store.GetNode("node-1")
	if err == nil {
		t.Error("expected error after delete")
	}

	// Get non-existent
	_, err = store.GetNode("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent node")
	}

	t.Logf("SQLiteStore NodeCRUD: save/get/list/upsert/delete all passed")
}

func TestSQLiteStore_ACLCRUD(t *testing.T) {
	store := newTestSQLiteStore(t)

	rule := &ACLRule{
		ID:        "rule-1",
		Source:    "node-A",
		Target:    "node-B",
		Action:    "allow",
		Protocol:  "tcp",
		Ports:     []int{80, 443, 8080},
		Priority:  10,
		CreatedAt: time.Now().Truncate(time.Second),
	}

	// Save
	if err := store.SaveACLRule(rule); err != nil {
		t.Fatalf("SaveACLRule: %v", err)
	}

	// Get
	got, err := store.GetACLRule("rule-1")
	if err != nil {
		t.Fatalf("GetACLRule: %v", err)
	}
	if got.ID != rule.ID {
		t.Errorf("ID = %q, want %q", got.ID, rule.ID)
	}
	if got.Action != "allow" {
		t.Errorf("Action = %q, want %q", got.Action, "allow")
	}
	if len(got.Ports) != 3 || got.Ports[0] != 80 {
		t.Errorf("Ports = %v, want [80 443 8080]", got.Ports)
	}
	if got.Priority != 10 {
		t.Errorf("Priority = %d, want 10", got.Priority)
	}

	// List
	list, err := store.ListACLRules()
	if err != nil {
		t.Fatalf("ListACLRules: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("ListACLRules count = %d, want 1", len(list))
	}

	// Upsert
	rule.Action = "deny"
	if err := store.SaveACLRule(rule); err != nil {
		t.Fatalf("SaveACLRule (upsert): %v", err)
	}
	got, _ = store.GetACLRule("rule-1")
	if got.Action != "deny" {
		t.Errorf("after upsert: Action = %q, want %q", got.Action, "deny")
	}

	// Delete
	if err := store.DeleteACLRule("rule-1"); err != nil {
		t.Fatalf("DeleteACLRule: %v", err)
	}
	_, err = store.GetACLRule("rule-1")
	if err == nil {
		t.Error("expected error after delete")
	}

	t.Logf("SQLiteStore ACLCRUD: save/get/list/upsert/delete all passed")
}

func TestSQLiteStore_KeyMaterialCRUD(t *testing.T) {
	store := newTestSQLiteStore(t)

	km := &KeyMaterial{
		NodeID:     "node-1",
		PublicKey:  "wg-pub-key-abc",
		KeyVersion: 1,
		RotatedAt:  time.Now().Truncate(time.Second),
		ExpiresAt:  time.Now().Add(24 * time.Hour).Truncate(time.Second),
	}

	// Save
	if err := store.SaveKeyMaterial(km); err != nil {
		t.Fatalf("SaveKeyMaterial: %v", err)
	}

	// Get
	got, err := store.GetKeyMaterial("node-1")
	if err != nil {
		t.Fatalf("GetKeyMaterial: %v", err)
	}
	if got.NodeID != km.NodeID {
		t.Errorf("NodeID = %q, want %q", got.NodeID, km.NodeID)
	}
	if got.PublicKey != km.PublicKey {
		t.Errorf("PublicKey = %q, want %q", got.PublicKey, km.PublicKey)
	}
	if got.KeyVersion != 1 {
		t.Errorf("KeyVersion = %d, want 1", got.KeyVersion)
	}

	// Upsert (rotate key)
	km.PublicKey = "wg-pub-key-v2"
	km.KeyVersion = 2
	if err := store.SaveKeyMaterial(km); err != nil {
		t.Fatalf("SaveKeyMaterial (upsert): %v", err)
	}
	got, _ = store.GetKeyMaterial("node-1")
	if got.KeyVersion != 2 {
		t.Errorf("after rotation: KeyVersion = %d, want 2", got.KeyVersion)
	}
	if got.PublicKey != "wg-pub-key-v2" {
		t.Errorf("after rotation: PublicKey = %q, want %q", got.PublicKey, "wg-pub-key-v2")
	}

	// Get non-existent
	_, err = store.GetKeyMaterial("nonexistent")
	if err == nil {
		t.Error("expected error for non-existent key")
	}

	t.Logf("SQLiteStore KeyMaterialCRUD: save/get/upsert all passed")
}

func TestSQLiteStore_InterfaceCompliance(t *testing.T) {
	store := newTestSQLiteStore(t)
	// Verify SQLiteStore satisfies Store interface at compile time
	var _ Store = store

	t.Logf("SQLiteStore implements Store interface")
}

func TestSQLiteStore_MultipleNodes(t *testing.T) {
	store := newTestSQLiteStore(t)

	// Insert 100 nodes
	for i := 0; i < 100; i++ {
		node := &NodeInfo{
			NodeID:      fmt.Sprintf("node-%d", i),
			PublicKey:   fmt.Sprintf("key-%d", i),
			NATType:     "full_cone",
			Region:      "us-east",
			ConnectedAt: time.Now(),
			LastSeen:    time.Now(),
			Metadata:    map[string]string{},
		}
		if err := store.SaveNode(node); err != nil {
			t.Fatalf("SaveNode %d: %v", i, err)
		}
	}

	list, err := store.ListNodes()
	if err != nil {
		t.Fatalf("ListNodes: %v", err)
	}
	if len(list) != 100 {
		t.Errorf("ListNodes count = %d, want 100", len(list))
	}

	// Delete half
	for i := 0; i < 50; i++ {
		if err := store.DeleteNode(fmt.Sprintf("node-%d", i)); err != nil {
			t.Fatalf("DeleteNode %d: %v", i, err)
		}
	}

	list, _ = store.ListNodes()
	if len(list) != 50 {
		t.Errorf("after delete: ListNodes count = %d, want 50", len(list))
	}

	t.Logf("SQLiteStore: 100 nodes insert/delete batch test passed")
}
