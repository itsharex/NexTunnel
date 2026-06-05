package dashboard

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"sync"
	"time"
)

// AlertLevel defines alert severity levels.
type AlertLevel string

const (
	AlertInfo     AlertLevel = "info"
	AlertWarning  AlertLevel = "warning"
	AlertCritical AlertLevel = "critical"
)

// AlertCondition defines the metric condition that triggers an alert.
type AlertCondition string

const (
	ConditionNodeOffline     AlertCondition = "node_offline"
	ConditionHighLatency     AlertCondition = "high_latency"
	ConditionHighBandwidth   AlertCondition = "high_bandwidth"
	ConditionNodeCount       AlertCondition = "node_count_low"
	ConditionErrorRate       AlertCondition = "error_rate"
	ConditionDiskUsage       AlertCondition = "disk_usage"
)

// AlertRule defines a single alerting rule.
type AlertRule struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Condition   AlertCondition `json:"condition"`
	Threshold   float64        `json:"threshold"`      // condition-specific threshold
	Level       AlertLevel     `json:"level"`
	Enabled     bool           `json:"enabled"`
	Cooldown    time.Duration  `json:"cooldown"`        // minimum time between alerts for this rule
	NodeFilter  string         `json:"node_filter"`     // empty = all nodes
	CreatedAt   time.Time      `json:"created_at"`
	LastFired   time.Time      `json:"last_fired"`
}

// AlertEvent is a fired alert instance.
type AlertEvent struct {
	ID        string     `json:"id"`
	RuleID    string     `json:"rule_id"`
	RuleName  string     `json:"rule_name"`
	Level     AlertLevel `json:"level"`
	Message   string     `json:"message"`
	NodeID    string     `json:"node_id,omitempty"`
	Value     float64    `json:"value"`       // the metric value that triggered
	Threshold float64    `json:"threshold"`
	CreatedAt time.Time  `json:"created_at"`
	Acked     bool       `json:"acked"`
	AckedBy   string     `json:"acked_by,omitempty"`
	AckedAt   time.Time  `json:"acked_at,omitempty"`
}

// Notifier is the interface for alert notification channels.
type Notifier interface {
	// Name returns the notifier channel name.
	Name() string
	// Send delivers an alert notification.
	Send(ctx context.Context, event *AlertEvent) error
}

// LogNotifier logs alerts to the structured logger.
type LogNotifier struct {
	Logger *slog.Logger
}

func (n *LogNotifier) Name() string { return "log" }

func (n *LogNotifier) Send(_ context.Context, event *AlertEvent) error {
	n.Logger.Warn("alert fired",
		"rule", event.RuleName,
		"level", event.Level,
		"message", event.Message,
		"node", event.NodeID,
		"value", event.Value,
		"threshold", event.Threshold)
	return nil
}

// WebhookNotifier sends alerts via HTTP POST to a configured URL.
type WebhookNotifier struct {
	URL     string
	Client  *http.Client
	Headers map[string]string
}

func (n *WebhookNotifier) Name() string { return "webhook" }

func (n *WebhookNotifier) Send(ctx context.Context, event *AlertEvent) error {
	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal alert: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, n.URL, jsonReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range n.Headers {
		req.Header.Set(k, v)
	}

	client := n.Client
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("webhook request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("webhook returned status %d", resp.StatusCode)
	}
	return nil
}

// jsonReader wraps a byte slice as an io.Reader for HTTP requests.
func jsonReader(data []byte) *bytes.Reader {
	return bytes.NewReader(data)
}

// AlertEngine evaluates metrics against alert rules and fires notifications.
type AlertEngine struct {
	mu        sync.RWMutex
	rules     map[string]*AlertRule
	events    map[string]*AlertEvent
	notifiers []Notifier
	logger    *slog.Logger
	nextID    int
}

// NewAlertEngine creates a new alert engine.
func NewAlertEngine(logger *slog.Logger) *AlertEngine {
	if logger == nil {
		logger = slog.Default()
	}
	return &AlertEngine{
		rules:  make(map[string]*AlertRule),
		events: make(map[string]*AlertEvent),
		notifiers: []Notifier{
			&LogNotifier{Logger: logger},
		},
		logger: logger,
	}
}

// AddNotifier registers a notification channel.
func (e *AlertEngine) AddNotifier(n Notifier) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.notifiers = append(e.notifiers, n)
}

// RemoveNotifier removes a notifier by name.
func (e *AlertEngine) RemoveNotifier(name string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	for i, n := range e.notifiers {
		if n.Name() == name {
			e.notifiers = append(e.notifiers[:i], e.notifiers[i+1:]...)
			return
		}
	}
}

// AddRule adds an alert rule.
func (e *AlertEngine) AddRule(rule *AlertRule) error {
	if rule.ID == "" {
		return fmt.Errorf("rule ID is required")
	}
	e.mu.Lock()
	defer e.mu.Unlock()
	rule.CreatedAt = time.Now()
	e.rules[rule.ID] = rule
	return nil
}

// RemoveRule removes an alert rule.
func (e *AlertEngine) RemoveRule(ruleID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.rules[ruleID]; !ok {
		return fmt.Errorf("rule %q not found", ruleID)
	}
	delete(e.rules, ruleID)
	return nil
}

// UpdateRule replaces an existing rule.
func (e *AlertEngine) UpdateRule(rule *AlertRule) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if _, ok := e.rules[rule.ID]; !ok {
		return fmt.Errorf("rule %q not found", rule.ID)
	}
	e.rules[rule.ID] = rule
	return nil
}

// GetRule returns a rule by ID.
func (e *AlertEngine) GetRule(ruleID string) (*AlertRule, bool) {
	e.mu.RLock()
	defer e.mu.RUnlock()
	r, ok := e.rules[ruleID]
	return r, ok
}

// ListRules returns all rules.
func (e *AlertEngine) ListRules() []*AlertRule {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]*AlertRule, 0, len(e.rules))
	for _, r := range e.rules {
		result = append(result, r)
	}
	return result
}

// RuleCount returns the number of rules.
func (e *AlertEngine) RuleCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.rules)
}

// MetricSample is a single metric observation for evaluation.
type MetricSample struct {
	NodeID string
	Metric AlertCondition
	Value  float64
}

// Evaluate checks all enabled rules against a batch of metric samples.
// Returns the list of newly fired alert events.
func (e *AlertEngine) Evaluate(samples []MetricSample) []*AlertEvent {
	e.mu.Lock()
	defer e.mu.Unlock()

	var fired []*AlertEvent
	now := time.Now()

	for _, sample := range samples {
		for _, rule := range e.rules {
			if !rule.Enabled {
				continue
			}
			if rule.Condition != sample.Metric {
				continue
			}
			if rule.NodeFilter != "" && rule.NodeFilter != sample.NodeID {
				continue
			}
			// Check cooldown
			if !rule.LastFired.IsZero() && now.Sub(rule.LastFired) < rule.Cooldown {
				continue
			}
			// Evaluate condition
			if !e.checkCondition(rule, sample.Value) {
				continue
			}

			// Fire alert
			e.nextID++
			event := &AlertEvent{
				ID:        fmt.Sprintf("alert-%d", e.nextID),
				RuleID:    rule.ID,
				RuleName:  rule.Name,
				Level:     rule.Level,
				Message:   fmt.Sprintf("Rule %q: %s threshold %.2f exceeded (value=%.2f)", rule.Name, rule.Condition, rule.Threshold, sample.Value),
				NodeID:    sample.NodeID,
				Value:     sample.Value,
				Threshold: rule.Threshold,
				CreatedAt: now,
			}
			rule.LastFired = now
			e.events[event.ID] = event
			fired = append(fired, event)

			// Notify asynchronously-safe (synchronous here, caller can wrap)
			for _, notifier := range e.notifiers {
				if err := notifier.Send(context.Background(), event); err != nil {
					e.logger.Error("notification failed", "notifier", notifier.Name(), "error", err)
				}
			}
		}
	}
	return fired
}

func (e *AlertEngine) checkCondition(rule *AlertRule, value float64) bool {
	switch rule.Condition {
	case ConditionNodeOffline:
		// value = 1 means offline
		return value >= 1
	case ConditionNodeCount:
		// value < threshold triggers (too few nodes)
		return value < rule.Threshold
	default:
		// value > threshold triggers (latency, bandwidth, error rate, etc.)
		return value > rule.Threshold
	}
}

// AckEvent acknowledges an alert event.
func (e *AlertEngine) AckEvent(eventID, ackedBy string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	event, ok := e.events[eventID]
	if !ok {
		return fmt.Errorf("event %q not found", eventID)
	}
	event.Acked = true
	event.AckedBy = ackedBy
	event.AckedAt = time.Now()
	return nil
}

// ListEvents returns all alert events, newest first.
func (e *AlertEngine) ListEvents() []*AlertEvent {
	e.mu.RLock()
	defer e.mu.RUnlock()
	result := make([]*AlertEvent, 0, len(e.events))
	for _, ev := range e.events {
		result = append(result, ev)
	}
	// Sort newest first
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].CreatedAt.After(result[i].CreatedAt) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}
	return result
}

// ListUnackedEvents returns only unacknowledged events.
func (e *AlertEngine) ListUnackedEvents() []*AlertEvent {
	all := e.ListEvents()
	result := make([]*AlertEvent, 0)
	for _, ev := range all {
		if !ev.Acked {
			result = append(result, ev)
		}
	}
	return result
}

// EventCount returns the total number of alert events.
func (e *AlertEngine) EventCount() int {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return len(e.events)
}
