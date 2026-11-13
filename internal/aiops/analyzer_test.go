package aiops

import (
	"strings"
	"testing"
	"time"
)

func TestAnalyze_DatabaseFailure(t *testing.T) {
	analyzer := NewAnalyzer("test-model")

	inc := Incident{
		ID:        "inc-001",
		Service:   "payment-service",
		Trigger:   "High Error Rate on /checkout",
		Timestamp: time.Now(),
		Severity:  SeverityCritical,
		Spans: []TraceSpan{
			{
				TraceID:      "trace-db-1",
				SpanID:       "span-1",
				Name:         "execute_sql",
				Service:      "payment-service",
				Status:       "ERROR",
				ErrorMessage: "database connection pool timeout after 30s",
			},
		},
		Metrics: []MetricAnomaly{
			{
				MetricName:    "db_connection_pool_wait_seconds",
				BaselineValue: 0.05,
				ObservedValue: 30.0,
				DeltaPercent:  59900.0,
				Unit:          "s",
			},
		},
		Logs: []LogRecord{
			{
				Timestamp: time.Now(),
				Level:     "ERROR",
				Body:      "ConnectionRefused: could not acquire connection from pool",
			},
		},
	}

	result, err := analyzer.Analyze(inc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.IncidentID != "inc-001" {
		t.Errorf("expected incident ID inc-001, got %s", result.IncidentID)
	}
	if !strings.Contains(result.PrimaryCulprit, "Database") {
		t.Errorf("expected Database culprit, got %s", result.PrimaryCulprit)
	}
	if result.ConfidenceScore < 0.90 {
		t.Errorf("expected high confidence score, got %.2f", result.ConfidenceScore)
	}
	if len(result.ActionableRemediation) == 0 {
		t.Errorf("expected actionable remediation steps")
	}
}

func TestAnalyze_OOMScenario(t *testing.T) {
	analyzer := NewAnalyzer("")

	inc := Incident{
		ID:        "inc-002",
		Service:   "analytics-worker",
		Trigger:   "Pod CrashLoopBackOff",
		Timestamp: time.Now(),
		Severity:  SeverityCritical,
		Spans:     []TraceSpan{},
		Metrics: []MetricAnomaly{
			{
				MetricName:    "container_memory_working_set_bytes",
				BaselineValue: 256.0,
				ObservedValue: 512.0,
				DeltaPercent:  100.0,
				Unit:          "MiB",
			},
		},
		Logs: []LogRecord{
			{
				Timestamp: time.Now(),
				Level:     "FATAL",
				Body:      "java.lang.OutOfMemoryError: Java heap space",
			},
		},
	}

	result, err := analyzer.Analyze(inc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.PrimaryCulprit, "Memory") {
		t.Errorf("expected Memory culprit, got %s", result.PrimaryCulprit)
	}
	if result.ConfidenceScore < 0.95 {
		t.Errorf("expected confidence score >= 0.95, got %.2f", result.ConfidenceScore)
	}
}

func TestAnalyze_TimeoutScenario(t *testing.T) {
	analyzer := NewAnalyzer("")

	inc := Incident{
		ID:        "inc-003",
		Service:   "frontend-api",
		Trigger:   "P99 Latency SLA Breach",
		Timestamp: time.Now(),
		Severity:  SeverityWarning,
		Spans: []TraceSpan{
			{
				TraceID:    "trace-slow-1",
				SpanID:     "span-slow",
				Name:       "call_external_inventory_api",
				Service:    "frontend-api",
				DurationMs: 850.0,
				Status:     "OK",
			},
		},
	}

	result, err := analyzer.Analyze(inc)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(result.PrimaryCulprit, "Latency") && !strings.Contains(result.PrimaryCulprit, "Timeout") {
		t.Errorf("expected Latency/Timeout culprit, got %s", result.PrimaryCulprit)
	}
}

func TestAnalyze_EmptyIncident(t *testing.T) {
	analyzer := NewAnalyzer("")
	_, err := analyzer.Analyze(Incident{})
	if err == nil {
		t.Fatal("expected error for empty incident, got nil")
	}
}

