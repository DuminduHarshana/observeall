package webui

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func setupTestServer(t *testing.T) (*Server, string, string) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "observeall.yaml")
	collectorConfigPath := filepath.Join(tmpDir, "collector-config.yaml")

	initialYAML := `name: test-app
environment: test
language: go
backends:
  - prometheus
signals:
  - traces
  - metrics
sampling: tail
`
	if err := os.WriteFile(configPath, []byte(initialYAML), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}

	mockClient := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			payload := "otelcol_receiver_accepted_spans 150\notelcol_receiver_accepted_metric_points 800\n"
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(payload)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	server := NewServerWithClient(configPath, collectorConfigPath, "http://mock-collector/metrics", mockClient)
	return server, configPath, collectorConfigPath
}

func TestServer_IndexHTML(t *testing.T) {
	server, _, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "ObserveAll Pipeline Control Plane") {
		t.Errorf("Index HTML missing header")
	}
}

func TestServer_GetPipeline(t *testing.T) {
	server, _, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/pipeline", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("invalid json response: %v", err)
	}

	if res["name"] != "test-app" {
		t.Errorf("expected project name test-app, got %v", res["name"])
	}
}

func TestServer_PostRoute(t *testing.T) {
	server, configPath, collectorConfigPath := setupTestServer(t)

	payload := map[string]any{
		"backends": []string{"datadog", "prometheus"},
	}
	body, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/pipeline/route", bytes.NewReader(body))
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", rec.Code, rec.Body.String())
	}

	// Verify collector-config.yaml was generated
	colData, err := os.ReadFile(collectorConfigPath)
	if err != nil {
		t.Fatalf("collector config was not written: %v", err)
	}
	if !strings.Contains(string(colData), "datadog") {
		t.Errorf("collector config does not contain updated backend datadog")
	}

	// Verify observeall.yaml was updated
	cfgData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("observeall.yaml could not be read: %v", err)
	}
	if !strings.Contains(string(cfgData), "datadog") {
		t.Errorf("observeall.yaml does not contain updated backend datadog")
	}
}

func TestServer_GetStatus(t *testing.T) {
	server, _, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/status", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if res["healthy"] != true {
		t.Errorf("expected healthy = true, got %v", res["healthy"])
	}
	if res["accepted_spans"] != float64(150) {
		t.Errorf("expected accepted_spans = 150, got %v", res["accepted_spans"])
	}
}

func TestServer_GetAIOpsAnalyze(t *testing.T) {
	server, _, _ := setupTestServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/aiops/analyze?scenario=db_exhaustion", nil)
	rec := httptest.NewRecorder()

	server.Handler().ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rec.Code)
	}

	var res map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &res); err != nil {
		t.Fatalf("invalid json: %v", err)
	}

	if res["primary_culprit"] == "" {
		t.Errorf("expected non-empty primary_culprit")
	}
	if res["confidence_score"] == 0 {
		t.Errorf("expected confidence_score > 0")
	}
}

