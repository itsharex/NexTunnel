package dashboard

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"sync"
	"time"
)

// ServerConfig configures the dashboard HTTP server.
type ServerConfig struct {
	ListenAddr     string
	AllowedOrigins []string
	Auth           AuthConfig
	Logger         *slog.Logger
}

// DefaultServerConfig returns sensible defaults.
func DefaultServerConfig() ServerConfig {
	return ServerConfig{
		ListenAddr:     "0.0.0.0:8080",
		AllowedOrigins: []string{"http://127.0.0.1:5173", "http://localhost:5173"},
		Auth:           DefaultAuthConfig(),
		Logger:         slog.Default(),
	}
}

// Server is the dashboard HTTP server.
type Server struct {
	config      ServerConfig
	auth        *AuthManager
	mux         *http.ServeMux
	store       *DataStore
	alertEngine *AlertEngine
	server      *http.Server
}

// DataStore holds dashboard data in-memory.
type DataStore struct {
	mu     sync.RWMutex
	nodes  map[string]*NodeStatus
	stats  map[string]*TrafficStats
	acls   map[string]*ACLRuleView
	alerts map[string]*Alert
}

// NewDataStore creates a new in-memory data store.
func NewDataStore() *DataStore {
	return &DataStore{
		nodes:  make(map[string]*NodeStatus),
		stats:  make(map[string]*TrafficStats),
		acls:   make(map[string]*ACLRuleView),
		alerts: make(map[string]*Alert),
	}
}

// NewServer creates a new dashboard server.
func NewServer(cfg ServerConfig) *Server {
	s := &Server{
		config:      cfg,
		auth:        NewAuthManager(cfg.Auth),
		mux:         http.NewServeMux(),
		store:       NewDataStore(),
		alertEngine: NewAlertEngine(cfg.Logger),
	}
	s.registerRoutes()
	return s
}

// AlertEngine returns the alert engine for external metric injection.
func (s *Server) AlertEngine() *AlertEngine {
	return s.alertEngine
}

// Start begins listening for HTTP requests.
func (s *Server) Start() error {
	if err := s.validateRuntimeConfig(); err != nil {
		return err
	}
	s.server = &http.Server{
		Addr:    s.config.ListenAddr,
		Handler: s.auth.AuthMiddleware(corsMiddleware(s.config.AllowedOrigins, s.mux)),
	}
	s.config.Logger.Info("dashboard server starting", "addr", s.config.ListenAddr)
	return s.server.ListenAndServe()
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() error {
	if s.server != nil {
		return s.server.Close()
	}
	return nil
}

// Handler returns the HTTP handler for testing.
func (s *Server) Handler() http.Handler {
	return s.auth.AuthMiddleware(corsMiddleware(s.config.AllowedOrigins, s.mux))
}

func (s *Server) validateRuntimeConfig() error {
	if s.config.Auth.SecretKey == "" {
		return fmt.Errorf("dashboard auth secret key must be configured")
	}
	if s.config.Auth.DefaultAdmin != "" && s.config.Auth.DefaultPass == "" {
		return fmt.Errorf("dashboard default admin password must not be empty")
	}
	if s.config.Auth.DefaultPass == "admin" {
		return fmt.Errorf("dashboard default admin password is insecure")
	}
	return nil
}

func (s *Server) registerRoutes() {
	// Auth
	s.mux.HandleFunc("POST /api/v1/auth/login", s.handleLogin)

	// Nodes
	s.mux.HandleFunc("GET /api/v1/nodes", s.handleListNodes)
	s.mux.HandleFunc("GET /api/v1/nodes/{id}", s.handleGetNode)
	s.mux.HandleFunc("DELETE /api/v1/nodes/{id}", s.handleDeleteNode)

	// Traffic stats
	s.mux.HandleFunc("GET /api/v1/stats", s.handleGetStats)
	s.mux.HandleFunc("GET /api/v1/stats/{node_id}", s.handleGetNodeStats)

	// ACL
	s.mux.HandleFunc("GET /api/v1/acl", s.handleListACL)
	s.mux.HandleFunc("POST /api/v1/acl", s.handleCreateACL)
	s.mux.HandleFunc("DELETE /api/v1/acl/{id}", s.handleDeleteACL)

	// Alerts
	s.mux.HandleFunc("GET /api/v1/alerts", s.handleListAlerts)
	s.mux.HandleFunc("POST /api/v1/alerts/{id}/ack", s.handleAckAlert)
	s.mux.HandleFunc("GET /api/v1/alerts/unacked", s.handleListUnackedAlerts)

	// Alert Rules
	s.mux.HandleFunc("GET /api/v1/alert-rules", s.handleListAlertRules)
	s.mux.HandleFunc("POST /api/v1/alert-rules", s.handleCreateAlertRule)
	s.mux.HandleFunc("PUT /api/v1/alert-rules/{id}", s.handleUpdateAlertRule)
	s.mux.HandleFunc("DELETE /api/v1/alert-rules/{id}", s.handleDeleteAlertRule)

	// Metrics ingestion (for external systems to push metrics and trigger alerts)
	s.mux.HandleFunc("POST /api/v1/metrics", s.handleIngestMetrics)

	// Users
	s.mux.HandleFunc("GET /api/v1/users", s.handleListUsers)

	// Health
	s.mux.HandleFunc("GET /api/v1/health", s.handleHealth)
}

func corsMiddleware(allowedOrigins []string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin != "" && slices.Contains(allowedOrigins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
		}
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(v)
}

func writeSuccess(w http.ResponseWriter, data interface{}) {
	writeJSON(w, http.StatusOK, APIResponse{Success: true, Data: data})
}

func writeError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, APIResponse{Success: false, Error: msg})
}

// --- Auth handler ---

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	resp, err := s.auth.Login(req)
	if err != nil {
		writeError(w, http.StatusUnauthorized, err.Error())
		return
	}
	writeSuccess(w, resp)
}

// --- Node handlers ---

func (s *Server) handleListNodes(w http.ResponseWriter, r *http.Request) {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()
	nodes := make([]*NodeStatus, 0, len(s.store.nodes))
	for _, n := range s.store.nodes {
		nodes = append(nodes, n)
	}
	writeSuccess(w, nodes)
}

func (s *Server) handleGetNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.store.mu.RLock()
	node, ok := s.store.nodes[id]
	s.store.mu.RUnlock()
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("node %q not found", id))
		return
	}
	writeSuccess(w, node)
}

func (s *Server) handleDeleteNode(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.store.mu.Lock()
	_, ok := s.store.nodes[id]
	if ok {
		delete(s.store.nodes, id)
	}
	s.store.mu.Unlock()
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("node %q not found", id))
		return
	}
	writeSuccess(w, map[string]string{"deleted": id})
}

// --- Stats handlers ---

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()
	stats := make([]*TrafficStats, 0, len(s.store.stats))
	for _, st := range s.store.stats {
		stats = append(stats, st)
	}
	writeSuccess(w, stats)
}

func (s *Server) handleGetNodeStats(w http.ResponseWriter, r *http.Request) {
	nodeID := r.PathValue("node_id")
	s.store.mu.RLock()
	st, ok := s.store.stats[nodeID]
	s.store.mu.RUnlock()
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("stats for %q not found", nodeID))
		return
	}
	writeSuccess(w, st)
}

// --- ACL handlers ---

func (s *Server) handleListACL(w http.ResponseWriter, r *http.Request) {
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()
	rules := make([]*ACLRuleView, 0, len(s.store.acls))
	for _, r := range s.store.acls {
		rules = append(rules, r)
	}
	writeSuccess(w, rules)
}

func (s *Server) handleCreateACL(w http.ResponseWriter, r *http.Request) {
	var rule ACLRuleView
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	rule.CreatedAt = time.Now()
	s.store.mu.Lock()
	s.store.acls[rule.ID] = &rule
	s.store.mu.Unlock()
	writeJSON(w, http.StatusCreated, APIResponse{Success: true, Data: &rule})
}

func (s *Server) handleDeleteACL(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	s.store.mu.Lock()
	_, ok := s.store.acls[id]
	if ok {
		delete(s.store.acls, id)
	}
	s.store.mu.Unlock()
	if !ok {
		writeError(w, http.StatusNotFound, fmt.Sprintf("acl rule %q not found", id))
		return
	}
	writeSuccess(w, map[string]string{"deleted": id})
}

// --- Alert handlers ---

func (s *Server) handleListAlerts(w http.ResponseWriter, r *http.Request) {
	// Primary source: AlertEngine events
	engineEvents := s.alertEngine.ListEvents()
	if len(engineEvents) > 0 {
		writeSuccess(w, engineEvents)
		return
	}
	// Fallback: legacy DataStore alerts
	s.store.mu.RLock()
	defer s.store.mu.RUnlock()
	alerts := make([]*Alert, 0, len(s.store.alerts))
	for _, a := range s.store.alerts {
		alerts = append(alerts, a)
	}
	writeSuccess(w, alerts)
}

func (s *Server) handleAckAlert(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var req struct {
		AckedBy string `json:"acked_by"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.AckedBy == "" {
		req.AckedBy = r.Header.Get("X-User-ID")
	}
	if err := s.alertEngine.AckEvent(id, req.AckedBy); err != nil {
		// Fallback to legacy DataStore alerts
		s.store.mu.Lock()
		alert, ok := s.store.alerts[id]
		if ok {
			alert.Acked = true
		}
		s.store.mu.Unlock()
		if !ok {
			writeError(w, http.StatusNotFound, fmt.Sprintf("alert %q not found", id))
			return
		}
		writeSuccess(w, alert)
		return
	}
	writeSuccess(w, map[string]string{"acked": id})
}

// --- Alert Rule handlers ---

func (s *Server) handleListAlertRules(w http.ResponseWriter, _ *http.Request) {
	writeSuccess(w, s.alertEngine.ListRules())
}

func (s *Server) handleCreateAlertRule(w http.ResponseWriter, r *http.Request) {
	var rule AlertRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if err := s.alertEngine.AddRule(&rule); err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusCreated, APIResponse{Success: true, Data: &rule})
}

func (s *Server) handleUpdateAlertRule(w http.ResponseWriter, r *http.Request) {
	var rule AlertRule
	if err := json.NewDecoder(r.Body).Decode(&rule); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	rule.ID = r.PathValue("id")
	if err := s.alertEngine.UpdateRule(&rule); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeSuccess(w, &rule)
}

func (s *Server) handleDeleteAlertRule(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := s.alertEngine.RemoveRule(id); err != nil {
		writeError(w, http.StatusNotFound, err.Error())
		return
	}
	writeSuccess(w, map[string]string{"deleted": id})
}

func (s *Server) handleListUnackedAlerts(w http.ResponseWriter, _ *http.Request) {
	writeSuccess(w, s.alertEngine.ListUnackedEvents())
}

// metricsRequest is the payload for POST /api/v1/metrics.
type metricsRequest struct {
	Samples []MetricSample `json:"samples"`
}

func (s *Server) handleIngestMetrics(w http.ResponseWriter, r *http.Request) {
	var req metricsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid request body")
		return
	}
	fired := s.alertEngine.Evaluate(req.Samples)
	writeSuccess(w, map[string]interface{}{
		"ingested": len(req.Samples),
		"fired":    len(fired),
		"alerts":   fired,
	})
}

// --- User handlers ---

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w, s.auth.ListUsers())
}

// --- Health handler ---

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	writeSuccess(w, map[string]string{"status": "ok"})
}

// --- DataStore mutation methods for external integration ---

// AddNode adds a node to the dashboard store.
func (ds *DataStore) AddNode(node *NodeStatus) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.nodes[node.NodeID] = node
}

// AddStats adds traffic stats for a node.
func (ds *DataStore) AddStats(stats *TrafficStats) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.stats[stats.NodeID] = stats
}

// AddAlert adds an alert.
func (ds *DataStore) AddAlert(alert *Alert) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.alerts[alert.ID] = alert
}
