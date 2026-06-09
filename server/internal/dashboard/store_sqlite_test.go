package dashboard

import (
	"testing"
	"time"
)

func newTestDashStore(t *testing.T) *SQLiteDashboardStore {
	t.Helper()
	store, err := NewSQLiteDashboardStore("") // in-memory
	if err != nil {
		t.Fatalf("NewSQLiteDashboardStore: %v", err)
	}
	t.Cleanup(func() { store.Close() })
	return store
}

func TestDashStore_NodeCRUD(t *testing.T) {
	store := newTestDashStore(t)

	node := &NodeStatus{
		NodeID:      "node-1",
		Region:      "us-east",
		NATType:     "full_cone",
		Online:      true,
		ConnectedAt: time.Now().Truncate(time.Second),
		LastSeen:    time.Now().Truncate(time.Second),
		RxBytes:     1024,
		TxBytes:     2048,
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
	if got.NodeID != "node-1" || got.Region != "us-east" || !got.Online {
		t.Errorf("GetNode: %+v", got)
	}
	if got.RxBytes != 1024 || got.TxBytes != 2048 {
		t.Errorf("bytes: rx=%d tx=%d", got.RxBytes, got.TxBytes)
	}

	// List
	list, err := store.ListNodes()
	if err != nil {
		t.Fatalf("ListNodes: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("ListNodes count = %d, want 1", len(list))
	}

	// Upsert
	node.Online = false
	node.RxBytes = 9999
	if err := store.SaveNode(node); err != nil {
		t.Fatalf("SaveNode upsert: %v", err)
	}
	got, _ = store.GetNode("node-1")
	if got.Online || got.RxBytes != 9999 {
		t.Errorf("after upsert: online=%v rx=%d", got.Online, got.RxBytes)
	}

	// Delete
	if err := store.DeleteNode("node-1"); err != nil {
		t.Fatalf("DeleteNode: %v", err)
	}
	_, err = store.GetNode("node-1")
	if err == nil {
		t.Error("expected error after delete")
	}

	t.Log("DashStore NodeCRUD: save/get/list/upsert/delete all passed")
}

func TestDashStore_ACLCRUD(t *testing.T) {
	store := newTestDashStore(t)

	rule := &ACLRuleView{
		ID:        "rule-1",
		Source:    "node-A",
		Target:    "node-B",
		Action:    "allow",
		Protocol:  "tcp",
		Priority:  10,
		Enabled:   true,
		CreatedAt: time.Now().Truncate(time.Second),
	}

	if err := store.SaveACL(rule); err != nil {
		t.Fatalf("SaveACL: %v", err)
	}

	got, err := store.GetACL("rule-1")
	if err != nil {
		t.Fatalf("GetACL: %v", err)
	}
	if got.Source != "node-A" || got.Action != "allow" || !got.Enabled {
		t.Errorf("GetACL: %+v", got)
	}

	list, err := store.ListACLs()
	if err != nil {
		t.Fatalf("ListACLs: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("ListACLs count = %d, want 1", len(list))
	}

	// Update
	rule.Action = "deny"
	if err := store.SaveACL(rule); err != nil {
		t.Fatalf("SaveACL update: %v", err)
	}
	got, _ = store.GetACL("rule-1")
	if got.Action != "deny" {
		t.Errorf("after update: action = %q, want deny", got.Action)
	}

	if err := store.DeleteACL("rule-1"); err != nil {
		t.Fatalf("DeleteACL: %v", err)
	}
	_, err = store.GetACL("rule-1")
	if err == nil {
		t.Error("expected error after delete")
	}

	t.Log("DashStore ACLCRUD: save/get/list/update/delete all passed")
}

func TestDashStore_AlertCRUD(t *testing.T) {
	store := newTestDashStore(t)

	alert := &Alert{
		ID:        "alert-1",
		Level:     "warning",
		Message:   "high latency detected",
		NodeID:    "node-1",
		CreatedAt: time.Now().Truncate(time.Second),
		Acked:     false,
	}

	if err := store.SaveAlert(alert); err != nil {
		t.Fatalf("SaveAlert: %v", err)
	}

	got, err := store.GetAlert("alert-1")
	if err != nil {
		t.Fatalf("GetAlert: %v", err)
	}
	if got.Level != "warning" || got.Acked {
		t.Errorf("GetAlert: %+v", got)
	}

	// Ack
	if err := store.AckAlert("alert-1"); err != nil {
		t.Fatalf("AckAlert: %v", err)
	}
	got, _ = store.GetAlert("alert-1")
	if !got.Acked {
		t.Error("expected acked = true after AckAlert")
	}

	// List
	list, err := store.ListAlerts()
	if err != nil {
		t.Fatalf("ListAlerts: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("ListAlerts count = %d, want 1", len(list))
	}

	// Ack non-existent
	if err := store.AckAlert("nonexistent"); err == nil {
		t.Error("expected error for non-existent alert")
	}

	if err := store.DeleteAlert("alert-1"); err != nil {
		t.Fatalf("DeleteAlert: %v", err)
	}

	t.Log("DashStore AlertCRUD: save/get/ack/list/delete all passed")
}

func TestDashStore_UserCRUD(t *testing.T) {
	store := newTestDashStore(t)

	user := &User{
		ID:           "user-1",
		Username:     "admin",
		Role:         "admin",
		Email:        "admin@example.com",
		PasswordHash: "$2a$10$somehash",
	}

	if err := store.SaveUser(user); err != nil {
		t.Fatalf("SaveUser: %v", err)
	}

	got, err := store.GetUser("admin")
	if err != nil {
		t.Fatalf("GetUser: %v", err)
	}
	if got.Role != "admin" || got.PasswordHash != "$2a$10$somehash" {
		t.Errorf("GetUser: %+v", got)
	}

	// List
	list, err := store.ListUsers()
	if err != nil {
		t.Fatalf("ListUsers: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("ListUsers count = %d, want 1", len(list))
	}

	// Update password hash
	user.PasswordHash = "$2a$10$newhash"
	if err := store.SaveUser(user); err != nil {
		t.Fatalf("SaveUser update: %v", err)
	}
	got, _ = store.GetUser("admin")
	if got.PasswordHash != "$2a$10$newhash" {
		t.Errorf("after update: hash = %q", got.PasswordHash)
	}

	if err := store.DeleteUser("admin"); err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	_, err = store.GetUser("admin")
	if err == nil {
		t.Error("expected error after delete")
	}

	t.Log("DashStore UserCRUD: save/get/list/update/delete all passed")
}

func TestDashStore_InterfaceCompliance(t *testing.T) {
	store := newTestDashStore(t)
	var _ DashboardStore = store
	t.Log("SQLiteDashboardStore implements DashboardStore interface")
}
