package webui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"time"

	"github.com/observeall/observeall/internal/aiops"
	"github.com/observeall/observeall/internal/analytics"
	"github.com/observeall/observeall/internal/config"
	"gopkg.in/yaml.v3"
)

// Server provides the HTTP endpoints for the ObserveAll Web UI.
type Server struct {
	configPath          string
	collectorConfigPath string
	collectorMetricsURL string
	httpClient          *http.Client
	mux                 *http.ServeMux
}

// NewServer constructs a new Web UI Server.
func NewServer(configPath, collectorConfigPath, collectorMetricsURL string) *Server {
	return NewServerWithClient(configPath, collectorConfigPath, collectorMetricsURL, http.DefaultClient)
}

// NewServerWithClient allows injecting a custom *http.Client for testing.
func NewServerWithClient(configPath, collectorConfigPath, collectorMetricsURL string, client *http.Client) *Server {
	if client == nil {
		client = http.DefaultClient
	}
	s := &Server{
		configPath:          configPath,
		collectorConfigPath: collectorConfigPath,
		collectorMetricsURL: collectorMetricsURL,
		httpClient:          client,
		mux:                 http.NewServeMux(),
	}
	s.registerRoutes()
	return s
}

func (s *Server) registerRoutes() {
	s.mux.HandleFunc("/", s.handleIndex)
	s.mux.HandleFunc("/api/pipeline", s.handlePipeline)
	s.mux.HandleFunc("/api/pipeline/route", s.handleRoute)
	s.mux.HandleFunc("/api/status", s.handleStatus)
	s.mux.HandleFunc("/api/aiops/analyze", s.handleAIOps)
}

// Handler returns the underlying http.Handler (useful for httptest).
func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(IndexHTML))
}

func (s *Server) handlePipeline(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	data, err := os.ReadFile(s.configPath)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	var proj config.Project
	if err := yaml.Unmarshal(data, &proj); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(proj)
}

type routeRequest struct {
	Signal   string   `json:"signal,omitempty"`
	Backends []string `json:"backends"`
}

func (s *Server) handleRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req routeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}

	data, err := os.ReadFile(s.configPath)
	if err != nil {
		http.Error(w, fmt.Sprintf("reading config: %v", err), http.StatusInternalServerError)
		return
	}

	var proj config.Project
	if err := yaml.Unmarshal(data, &proj); err != nil {
		http.Error(w, fmt.Sprintf("parsing config: %v", err), http.StatusInternalServerError)
		return
	}

	// Update backends
	if len(req.Backends) > 0 {
		proj.Backends = req.Backends
	}

	// Write back observeall.yaml
	updated, err := yaml.Marshal(&proj)
	if err != nil {
		http.Error(w, fmt.Sprintf("marshaling config: %v", err), http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(s.configPath, updated, 0644); err != nil {
		http.Error(w, fmt.Sprintf("writing config: %v", err), http.StatusInternalServerError)
		return
	}

	// Regenerate collector-config.yaml
	collectorCfg, err := config.GenerateCollectorConfig(proj)
	if err != nil {
		http.Error(w, fmt.Sprintf("generating collector config: %v", err), http.StatusInternalServerError)
		return
	}
	if err := os.WriteFile(s.collectorConfigPath, []byte(collectorCfg), 0644); err != nil {
		http.Error(w, fmt.Sprintf("writing collector config: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"success":  true,
		"backends": proj.Backends,
		"signals":  proj.Signals,
	})
}

type statusResponse struct {
	Healthy              bool  `json:"healthy"`
	AcceptedSpans        int64 `json:"accepted_spans"`
	AcceptedMetricPoints int64 `json:"accepted_metric_points"`
	RefusedSpans         int64 `json:"refused_spans"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	mc := analytics.NewMetricsCollectorWithClient(s.collectorMetricsURL, s.httpClient)
	stats, err := mc.FetchStats()

	resp := statusResponse{
		Healthy:              err == nil,
		AcceptedSpans:        stats.AcceptedSpans,
		AcceptedMetricPoints: stats.AcceptedMetricPoints,
		RefusedSpans:         stats.RefusedSpans,
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

func (s *Server) handleAIOps(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	scenario := r.URL.Query().Get("scenario")
	if scenario == "" {
		scenario = "db_exhaustion"
	}

	analyzer := aiops.NewAnalyzer("gemini-1.5-pro / claude-3.5-sonnet")
	inc := aiops.Incident{
		ID:        fmt.Sprintf("inc-%d", time.Now().Unix()%10000),
		Service:   "dummy-shop-service",
		Trigger:   "Automated Anomaly Detection",
		Timestamp: time.Now(),
		Severity:  aiops.SeverityCritical,
		Spans: []aiops.TraceSpan{
			{
				TraceID:      "trace-db-7102cd34",
				SpanID:       "span-db-01",
				Name:         "charge_payment",
				Service:      "dummy-shop-service",
				Status:       "ERROR",
				ErrorMessage: "Timeout waiting for idle database connection from pool",
			},
		},
		Metrics: []aiops.MetricAnomaly{
			{
				MetricName:    "dummy_requests_total{status=\"500\"}",
				BaselineValue: 0.0,
				ObservedValue: 42.0,
				DeltaPercent:  4200.0,
				Unit:          "requests",
			},
		},
		Logs: []aiops.LogRecord{
			{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Body:      "ConnectionRefusedError: Max connection pool capacity (100) reached",
			},
		},
	}

	result, err := analyzer.Analyze(inc)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

// Start runs the HTTP server listening on the provided address.
func (s *Server) Start(addr string) error {
	return http.ListenAndServe(addr, s.mux)
}

