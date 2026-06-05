package dashboard

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"
)

// mockNotifier captures sent alerts for testing.
type mockNotifier struct {
	mu     sync.Mutex
	events []*AlertEvent
}

func (n *mockNotifier) Name() string { return "mock" }
func (n *mockNotifier) Send(_ context.Context, event *AlertEvent) error {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.events = append(n.events, event)
	return nil
}

func TestAlertEngine_AddRemoveRule(t *testing.T) {
	engine := NewAlertEngine(slog.Default())

	rule := &AlertRule{
		ID:        "rule-1",
		Name:      "High Latency",
		Condition: ConditionHighLatency,
		Threshold: 200,
		Level:     AlertWarning,
		Enabled:   true,
	}
	if err := engine.AddRule(rule); err != nil {
		t.Fatal(err)
	}
	if engine.RuleCount() != 1 {
		t.Fatalf("expected 1 rule, got %d", engine.RuleCount())
	}

	if err := engine.RemoveRule("rule-1"); err != nil {
		t.Fatal(err)
	}
	if engine.RuleCount() != 0 {
		t.Fatalf("expected 0 rules, got %d", engine.RuleCount())
	}
}

func TestAlertEngine_AddRuleRequiresID(t *testing.T) {
	engine := NewAlertEngine(slog.Default())
	err := engine.AddRule(&AlertRule{Name: "no-id"})
	if err == nil {
		t.Fatal("expected error for empty ID")
	}
}

func TestAlertEngine_EvaluateFiresAlert(t *testing.T) {
	engine := NewAlertEngine(slog.Default())
	mock := &mockNotifier{}
	engine.AddNotifier(mock)

	engine.AddRule(&AlertRule{
		ID:        "rule-latency",
		Name:      "High Latency",
		Condition: ConditionHighLatency,
		Threshold: 100,
		Level:     AlertWarning,
		Enabled:   true,
	})

	fired := engine.Evaluate([]MetricSample{
		{NodeID: "node-1", Metric: ConditionHighLatency, Value: 150},
	})

	if len(fired) != 1 {
		t.Fatalf("expected 1 fired alert, got %d", len(fired))
	}
	if fired[0].Level != AlertWarning {
		t.Fatalf("expected warning level, got %s", fired[0].Level)
	}
	if fired[0].NodeID != "node-1" {
		t.Fatalf("expected node-1, got %s", fired[0].NodeID)
	}

	mock.mu.Lock()
	if len(mock.events) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(mock.events))
	}
	mock.mu.Unlock()
}

func TestAlertEngine_EvaluateNoFire(t *testing.T) {
	engine := NewAlertEngine(slog.Default())

	engine.AddRule(&AlertRule{
		ID:        "rule-latency",
		Name:      "High Latency",
		Condition: ConditionHighLatency,
		Threshold: 200,
		Level:     AlertWarning,
		Enabled:   true,
	})

	fired := engine.Evaluate([]MetricSample{
		{NodeID: "node-1", Metric: ConditionHighLatency, Value: 50},
	})

	if len(fired) != 0 {
		t.Fatalf("expected 0 fired alerts, got %d", len(fired))
	}
}

func TestAlertEngine_DisabledRule(t *testing.T) {
	engine := NewAlertEngine(slog.Default())

	engine.AddRule(&AlertRule{
		ID:        "rule-disabled",
		Name:      "Disabled Rule",
		Condition: ConditionHighLatency,
		Threshold: 10,
		Level:     AlertCritical,
		Enabled:   false,
	})

	fired := engine.Evaluate([]MetricSample{
		{NodeID: "node-1", Metric: ConditionHighLatency, Value: 999},
	})
	if len(fired) != 0 {
		t.Fatal("disabled rule should not fire")
	}
}

func TestAlertEngine_Cooldown(t *testing.T) {
	engine := NewAlertEngine(slog.Default())

	engine.AddRule(&AlertRule{
		ID:        "rule-cooldown",
		Name:      "Cooldown Test",
		Condition: ConditionHighLatency,
		Threshold: 100,
		Level:     AlertWarning,
		Enabled:   true,
		Cooldown:  1 * time.Hour,
	})

	// First evaluation should fire
	fired1 := engine.Evaluate([]MetricSample{
		{NodeID: "node-1", Metric: ConditionHighLatency, Value: 200},
	})
	if len(fired1) != 1 {
		t.Fatal("expected first alert")
	}

	// Second evaluation within cooldown should NOT fire
	fired2 := engine.Evaluate([]MetricSample{
		{NodeID: "node-1", Metric: ConditionHighLatency, Value: 300},
	})
	if len(fired2) != 0 {
		t.Fatal("expected no alert during cooldown")
	}
}

func TestAlertEngine_NodeFilter(t *testing.T) {
	engine := NewAlertEngine(slog.Default())

	engine.AddRule(&AlertRule{
		ID:         "rule-filter",
		Name:       "Node Specific",
		Condition:  ConditionHighLatency,
		Threshold:  100,
		Level:      AlertWarning,
		Enabled:    true,
		NodeFilter: "node-a",
	})

	// Should fire for node-a
	fired1 := engine.Evaluate([]MetricSample{
		{NodeID: "node-a", Metric: ConditionHighLatency, Value: 200},
	})
	if len(fired1) != 1 {
		t.Fatal("expected alert for node-a")
	}

	// Should NOT fire for node-b
	fired2 := engine.Evaluate([]MetricSample{
		{NodeID: "node-b", Metric: ConditionHighLatency, Value: 200},
	})
	if len(fired2) != 0 {
		t.Fatal("expected no alert for node-b")
	}
}

func TestAlertEngine_NodeOffline(t *testing.T) {
	engine := NewAlertEngine(slog.Default())

	engine.AddRule(&AlertRule{
		ID:        "rule-offline",
		Name:      "Node Offline",
		Condition: ConditionNodeOffline,
		Level:     AlertCritical,
		Enabled:   true,
	})

	fired := engine.Evaluate([]MetricSample{
		{NodeID: "node-1", Metric: ConditionNodeOffline, Value: 1},
	})
	if len(fired) != 1 {
		t.Fatal("expected offline alert")
	}
	if fired[0].Level != AlertCritical {
		t.Fatalf("expected critical, got %s", fired[0].Level)
	}
}

func TestAlertEngine_NodeCountLow(t *testing.T) {
	engine := NewAlertEngine(slog.Default())

	engine.AddRule(&AlertRule{
		ID:        "rule-count",
		Name:      "Low Node Count",
		Condition: ConditionNodeCount,
		Threshold: 3,
		Level:     AlertWarning,
		Enabled:   true,
	})

	// Value 2 < threshold 3 → should fire
	fired := engine.Evaluate([]MetricSample{
		{NodeID: "", Metric: ConditionNodeCount, Value: 2},
	})
	if len(fired) != 1 {
		t.Fatal("expected low count alert")
	}
}

func TestAlertEngine_AckEvent(t *testing.T) {
	engine := NewAlertEngine(slog.Default())

	engine.AddRule(&AlertRule{
		ID:        "rule-ack",
		Name:      "Ack Test",
		Condition: ConditionHighLatency,
		Threshold: 100,
		Level:     AlertWarning,
		Enabled:   true,
	})

	fired := engine.Evaluate([]MetricSample{
		{NodeID: "node-1", Metric: ConditionHighLatency, Value: 200},
	})
	if len(fired) != 1 {
		t.Fatal("expected alert")
	}

	eventID := fired[0].ID
	if err := engine.AckEvent(eventID, "admin"); err != nil {
		t.Fatal(err)
	}

	events := engine.ListEvents()
	if !events[0].Acked {
		t.Fatal("expected acked event")
	}
	if events[0].AckedBy != "admin" {
		t.Fatalf("expected acked_by admin, got %s", events[0].AckedBy)
	}
}

func TestAlertEngine_ListUnacked(t *testing.T) {
	engine := NewAlertEngine(slog.Default())

	engine.AddRule(&AlertRule{
		ID:        "rule-unacked",
		Name:      "Unacked Test",
		Condition: ConditionHighLatency,
		Threshold: 100,
		Level:     AlertWarning,
		Enabled:   true,
	})

	engine.Evaluate([]MetricSample{
		{NodeID: "n1", Metric: ConditionHighLatency, Value: 200},
		{NodeID: "n2", Metric: ConditionHighLatency, Value: 300},
	})

	unacked := engine.ListUnackedEvents()
	if len(unacked) != 2 {
		t.Fatalf("expected 2 unacked, got %d", len(unacked))
	}

	engine.AckEvent(unacked[0].ID, "admin")
	unacked = engine.ListUnackedEvents()
	if len(unacked) != 1 {
		t.Fatalf("expected 1 unacked after ack, got %d", len(unacked))
	}
}

func TestAlertEngine_RemoveNotifier(t *testing.T) {
	engine := NewAlertEngine(slog.Default())
	mock := &mockNotifier{}
	engine.AddNotifier(mock)
	engine.RemoveNotifier("mock")

	engine.AddRule(&AlertRule{
		ID:        "rule-nn",
		Name:      "No Notifier",
		Condition: ConditionHighLatency,
		Threshold: 100,
		Level:     AlertWarning,
		Enabled:   true,
	})

	engine.Evaluate([]MetricSample{
		{NodeID: "n1", Metric: ConditionHighLatency, Value: 200},
	})

	mock.mu.Lock()
	if len(mock.events) != 0 {
		t.Fatalf("expected 0 notifications after removing notifier, got %d", len(mock.events))
	}
	mock.mu.Unlock()
}

func TestLogNotifier_Name(t *testing.T) {
	n := &LogNotifier{Logger: slog.Default()}
	if n.Name() != "log" {
		t.Fatalf("expected 'log', got %s", n.Name())
	}
}
