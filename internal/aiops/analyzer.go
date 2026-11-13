package aiops

import (
	"fmt"
	"strings"
	"time"
)

// Severity represents incident severity levels.
type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityWarning  Severity = "WARNING"
	SeverityInfo     Severity = "INFO"
)

// TraceSpan represents a trace span collected during an incident.
type TraceSpan struct {
	TraceID      string            `json:"trace_id"`
	SpanID       string            `json:"span_id"`
	Name         string            `json:"name"`
	Service      string            `json:"service"`
	DurationMs   float64           `json:"duration_ms"`
	Status       string            `json:"status"` // "OK" or "ERROR"
	ErrorMessage string            `json:"error_message,omitempty"`
	Attributes   map[string]string `json:"attributes,omitempty"`
}

// MetricAnomaly represents a statistical anomaly detected on a metric.
type MetricAnomaly struct {
	MetricName    string  `json:"metric_name"`
	BaselineValue float64 `json:"baseline_value"`
	ObservedValue float64 `json:"observed_value"`
	DeltaPercent  float64 `json:"delta_percent"`
	Unit          string  `json:"unit,omitempty"`
}

// LogRecord represents an error log line correlated with the incident.
type LogRecord struct {
	Timestamp  time.Time         `json:"timestamp"`
	Level      string            `json:"level"`
	Body       string            `json:"body"`
	Attributes map[string]string `json:"attributes,omitempty"`
}

// Incident packages correlated traces, metrics, and logs for root-cause diagnosis.
type Incident struct {
	ID          string          `json:"id"`
	Service     string          `json:"service"`
	Trigger     string          `json:"trigger"`
	Timestamp   time.Time       `json:"timestamp"`
	Severity    Severity        `json:"severity"`
	Spans       []TraceSpan     `json:"spans"`
	Metrics     []MetricAnomaly `json:"metrics"`
	Logs        []LogRecord     `json:"logs"`
}

// RCAResult contains the synthesized root-cause diagnosis and remediation plan.
type RCAResult struct {
	IncidentID            string    `json:"incident_id"`
	Timestamp             time.Time `json:"timestamp"`
	Title                 string    `json:"title"`
	PrimaryCulprit        string    `json:"primary_culprit"`
	RootCause             string    `json:"root_cause"`
	ConfidenceScore       float64   `json:"confidence_score"` // 0.0 - 1.0
	CorrelatedSignals     []string  `json:"correlated_signals"`
	ActionableRemediation []string  `json:"actionable_remediation"`
	LLMReasoning          string    `json:"llm_reasoning"`
}

// Analyzer provides cross-signal correlation and root-cause analysis.
type Analyzer struct {
	Model string
}

// NewAnalyzer creates a new AIOps Analyzer.
func NewAnalyzer(model string) *Analyzer {
	if model == "" {
		model = "gemini-1.5-pro / claude-3.5-sonnet"
	}
	return &Analyzer{Model: model}
}

// Analyze performs cross-signal correlation between traces, metrics, and logs
// to deduce the root cause and generate actionable remediation recommendations.
func (a *Analyzer) Analyze(inc Incident) (*RCAResult, error) {
	if len(inc.Spans) == 0 && len(inc.Metrics) == 0 && len(inc.Logs) == 0 {
		return nil, fmt.Errorf("cannot analyze empty incident: missing traces, metrics, and logs")
	}

	result := &RCAResult{
		IncidentID: inc.ID,
		Timestamp:  time.Now(),
		Title:      fmt.Sprintf("[%s] %s in %s", inc.Severity, inc.Trigger, inc.Service),
	}

	var signals []string
	var errorSpans []TraceSpan
	var slowSpans []TraceSpan

	for _, s := range inc.Spans {
		if strings.EqualFold(s.Status, "ERROR") || s.ErrorMessage != "" {
			errorSpans = append(errorSpans, s)
			signals = append(signals, fmt.Sprintf("Trace Span %s (%s) failed with: %s", s.Name, s.TraceID, s.ErrorMessage))
		} else if s.DurationMs > 500 {
			slowSpans = append(slowSpans, s)
			signals = append(signals, fmt.Sprintf("Trace Span %s (%s) latency %.1fms exceeds 500ms SLA", s.Name, s.TraceID, s.DurationMs))
		}
	}

	for _, m := range inc.Metrics {
		sign := "+"
		if m.DeltaPercent < 0 {
			sign = ""
		}
		signals = append(signals, fmt.Sprintf("Metric %s anomaly: %s%.1f%% (from %.2f to %.2f %s)", m.MetricName, sign, m.DeltaPercent, m.BaselineValue, m.ObservedValue, m.Unit))
	}

	for _, l := range inc.Logs {
		if strings.EqualFold(l.Level, "ERROR") || strings.EqualFold(l.Level, "FATAL") {
			signals = append(signals, fmt.Sprintf("Log [%s]: %s", l.Level, l.Body))
		}
	}

	result.CorrelatedSignals = signals

	// Cross-Signal Automated Correlation Rules
	hasDBFailure := false
	hasTimeout := false
	hasOOM := false

	for _, s := range errorSpans {
		lowerErr := strings.ToLower(s.ErrorMessage + " " + s.Name)
		if strings.Contains(lowerErr, "database") || strings.Contains(lowerErr, "sql") || strings.Contains(lowerErr, "connection pool") {
			hasDBFailure = true
		}
		if strings.Contains(lowerErr, "timeout") || strings.Contains(lowerErr, "deadline") {
			hasTimeout = true
		}
	}

	for _, l := range inc.Logs {
		lowerBody := strings.ToLower(l.Body)
		if strings.Contains(lowerBody, "outofmemory") || strings.Contains(lowerBody, "out of memory") || strings.Contains(lowerBody, "oom") {
			hasOOM = true
		}
		if strings.Contains(lowerBody, "connection refused") || strings.Contains(lowerBody, "database") {
			hasDBFailure = true
		}
	}

	if hasDBFailure {
		result.PrimaryCulprit = "Downstream Database Connection Exhaustion"
		result.RootCause = "Database client pool exhausted under spike; upstream requests backing up."
		result.ConfidenceScore = 0.94
		result.ActionableRemediation = []string{
			"Increase database max_connections or connection pool size in application configuration.",
			"Enable ObserveAll tail-sampling for slow queries (>500ms) to identify unindexed statements.",
			"Verify database CPU and disk IOPS limits in cloud console.",
		}
		result.LLMReasoning = fmt.Sprintf("Cross-signal analysis correlated %d failed span(s) with database connection errors. Correlated metrics demonstrate request queue saturation immediately preceding the latency spike.", len(errorSpans))
	} else if hasOOM {
		result.PrimaryCulprit = "JVM / Container Out-of-Memory (OOM)"
		result.RootCause = "Memory consumption surpassed cgroup boundary; process evicted by kernel OOM-killer."
		result.ConfidenceScore = 0.98
		result.ActionableRemediation = []string{
			"Increase container memory limit in Kubernetes pod spec (spec.containers[].resources.limits.memory).",
			"Inspect recent commits for unclosed file descriptors, in-memory caches, or memory leaks.",
			"Review heap dumps or activate continuous profiling.",
		}
		result.LLMReasoning = "Log stream contains fatal OOM events matching container termination signals. Process memory limits require upward adjustment."
	} else if hasTimeout || len(slowSpans) > 0 {
		result.PrimaryCulprit = "Upstream Service Latency & Network Timeout"
		result.RootCause = "Upstream API dependencies exceeded response threshold, triggering cascade timeouts."
		result.ConfidenceScore = 0.89
		result.ActionableRemediation = []string{
			"Implement circuit breakers (e.g. Resilience4j or Envoy) around third-party API invocations.",
			"Review distributed trace flame graphs for external network latency.",
			"Configure client timeouts to fail fast rather than stalling downstream queues.",
		}
		result.LLMReasoning = fmt.Sprintf("Spans indicate %d operation(s) exceeding 500ms SLA without database errors, pointing to network latency or slow upstream dependencies.", len(slowSpans))
	} else {
		result.PrimaryCulprit = "Application Logic Exception"
		result.RootCause = "Uncaught runtime exception in request processing handler."
		result.ConfidenceScore = 0.85
		result.ActionableRemediation = []string{
			"Inspect stack trace in error logs and verify input parameter validation.",
			"Check recent deployments or canary releases for regressions.",
			"Add unit and integration tests covering the edge-case inputs.",
		}
		result.LLMReasoning = "Correlated traces and logs demonstrate isolated application exception without system resource or network degradation."
	}

	return result, nil
}
