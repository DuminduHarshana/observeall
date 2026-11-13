package analytics

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTripperFunc func(*http.Request) (*http.Response, error)

func (f roundTripperFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

// samplePrometheusPayload is a minimal Prometheus text-format response that
// mirrors what the OTel collector exposes.
const samplePrometheusPayload = `# HELP otelcol_receiver_accepted_spans Number of spans successfully pushed into the pipeline.
# TYPE otelcol_receiver_accepted_spans counter
otelcol_receiver_accepted_spans{receiver="otlp",transport="grpc"} 1200
otelcol_receiver_accepted_spans{receiver="otlp",transport="http"} 300
# HELP otelcol_receiver_accepted_metric_points Number of metric points successfully pushed into the pipeline.
# TYPE otelcol_receiver_accepted_metric_points counter
otelcol_receiver_accepted_metric_points{receiver="otlp",transport="grpc"} 5000
# HELP otelcol_receiver_refused_spans Number of spans that could not be pushed into the pipeline.
# TYPE otelcol_receiver_refused_spans counter
otelcol_receiver_refused_spans{receiver="otlp",transport="grpc"} 500
# HELP otelcol_exporter_sent_metric_points Number of metric points successfully sent to destination.
# TYPE otelcol_exporter_sent_metric_points counter
otelcol_exporter_sent_metric_points{exporter="datadog"} 4800
# HELP otelcol_exporter_send_failed_metric_points Number of metric points that failed to be sent to destination.
# TYPE otelcol_exporter_send_failed_metric_points counter
otelcol_exporter_send_failed_metric_points{exporter="datadog"} 200
`

func TestFetchStats_ParsesPrometheusResponse(t *testing.T) {
	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(samplePrometheusPayload)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	mc := NewMetricsCollectorWithClient("http://mock-collector/metrics", client)
	stats, err := mc.FetchStats()
	if err != nil {
		t.Fatalf("FetchStats returned unexpected error: %v", err)
	}

	// AcceptedSpans = 1200 + 300
	if stats.AcceptedSpans != 1500 {
		t.Errorf("AcceptedSpans: got %d, want 1500", stats.AcceptedSpans)
	}

	if stats.AcceptedMetricPoints != 5000 {
		t.Errorf("AcceptedMetricPoints: got %d, want 5000", stats.AcceptedMetricPoints)
	}

	if stats.RefusedSpans != 500 {
		t.Errorf("RefusedSpans: got %d, want 500", stats.RefusedSpans)
	}

	if stats.ExportedPoints != 4800 {
		t.Errorf("ExportedPoints: got %d, want 4800", stats.ExportedPoints)
	}

	if stats.FailedExports != 200 {
		t.Errorf("FailedExports: got %d, want 200", stats.FailedExports)
	}

	// Dropped% = 500/(1500+500)*100 = 25%
	wantDropped := 25.0
	if stats.SampledDroppedPercent != wantDropped {
		t.Errorf("SampledDroppedPercent: got %.2f, want %.2f", stats.SampledDroppedPercent, wantDropped)
	}

	if stats.CollectedAt.IsZero() {
		t.Error("CollectedAt should be populated")
	}

	if stats.CollectedAt.After(time.Now().Add(time.Second)) {
		t.Error("CollectedAt should not be in the future")
	}
}

func TestFetchStats_CollectorUnreachable(t *testing.T) {
	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return nil, fmt.Errorf("connection refused")
		}),
	}
	mc := NewMetricsCollectorWithClient("http://unreachable:19999/metrics", client)
	_, err := mc.FetchStats()
	if err == nil {
		t.Fatal("expected an error when collector is unreachable, got nil")
	}
}

func TestFetchStats_Non200Response(t *testing.T) {
	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: 404,
				Body:       io.NopCloser(strings.NewReader("not found")),
				Header:     make(http.Header),
			}, nil
		}),
	}

	mc := NewMetricsCollectorWithClient("http://mock-collector/metrics", client)
	_, err := mc.FetchStats()
	if err == nil {
		t.Fatal("expected an error for non-200 response, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("error message should mention HTTP status 404, got: %v", err)
	}
}

func TestFetchStats_EmptyMetrics(t *testing.T) {
	client := &http.Client{
		Transport: roundTripperFunc(func(r *http.Request) (*http.Response, error) {
			emptyPayload := "# HELP nothing_here no metric\n# TYPE nothing_here counter\n"
			return &http.Response{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(emptyPayload)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	mc := NewMetricsCollectorWithClient("http://mock-collector/metrics", client)
	stats, err := mc.FetchStats()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stats.AcceptedSpans != 0 || stats.AcceptedMetricPoints != 0 {
		t.Errorf("expected zeroed stats for empty payload, got: %+v", stats)
	}
	if stats.SampledDroppedPercent != 0 {
		t.Errorf("SampledDroppedPercent should be 0 when no spans, got %.2f", stats.SampledDroppedPercent)
	}
}

// ---- EstimateCost tests ----

func makeStats(spans, metricPoints int64) IngestStats {
	return IngestStats{
		AcceptedSpans:        spans,
		AcceptedMetricPoints: metricPoints,
		CollectedAt:          time.Now(),
	}
}

func TestEstimateCost_Datadog(t *testing.T) {
	mc := NewMetricsCollector("http://localhost:8888/metrics")
	// 2 million spans → $1.70 * 2 = $3.40
	stats := makeStats(2_000_000, 0)
	est := mc.EstimateCost(stats, []string{"datadog"})

	wantSpans := 3.40
	if abs(est.SpansCostUSD-wantSpans) > 0.001 {
		t.Errorf("SpansCostUSD: got %.4f, want %.4f", est.SpansCostUSD, wantSpans)
	}
	if est.TotalCostUSD != est.SpansCostUSD {
		t.Errorf("TotalCostUSD should equal SpansCostUSD when only spans are costed")
	}
	if !strings.Contains(est.PricingNotes, "Datadog") {
		t.Errorf("PricingNotes should mention Datadog, got: %s", est.PricingNotes)
	}
}

func TestEstimateCost_NewRelic(t *testing.T) {
	mc := NewMetricsCollector("http://localhost:8888/metrics")
	// 1 GB worth of points: 1024*1024 points (each 1 KB) = 1 GB → $0.30
	points := int64(1024 * 1024)
	stats := makeStats(0, points)
	est := mc.EstimateCost(stats, []string{"newrelic"})

	wantMetrics := 0.30
	if abs(est.MetricsCostUSD-wantMetrics) > 0.001 {
		t.Errorf("MetricsCostUSD: got %.6f, want %.4f", est.MetricsCostUSD, wantMetrics)
	}
	if !strings.Contains(est.PricingNotes, "New Relic") {
		t.Errorf("PricingNotes should mention New Relic, got: %s", est.PricingNotes)
	}
}

func TestEstimateCost_GrafanaFreeWithinTier(t *testing.T) {
	mc := NewMetricsCollector("http://localhost:8888/metrics")
	// 30k spans — within 50k free tier → $0 cost
	stats := makeStats(30_000, 0)
	est := mc.EstimateCost(stats, []string{"grafana"})

	if est.SpansCostUSD != 0 {
		t.Errorf("SpansCostUSD should be 0 within free tier, got %.4f", est.SpansCostUSD)
	}
	if est.TotalCostUSD != 0 {
		t.Errorf("TotalCostUSD should be 0 within free tier, got %.4f", est.TotalCostUSD)
	}
}

func TestEstimateCost_GrafanaBeyondFreeTier(t *testing.T) {
	mc := NewMetricsCollector("http://localhost:8888/metrics")
	// 1,050,000 spans → 1,000,000 billable → $0.50
	stats := makeStats(1_050_000, 0)
	est := mc.EstimateCost(stats, []string{"grafana"})

	wantSpans := 0.50
	if abs(est.SpansCostUSD-wantSpans) > 0.001 {
		t.Errorf("SpansCostUSD: got %.4f, want %.4f", est.SpansCostUSD, wantSpans)
	}
}

func TestEstimateCost_MultiBackend(t *testing.T) {
	mc := NewMetricsCollector("http://localhost:8888/metrics")
	stats := makeStats(1_000_000, 0)
	// datadog: 1M spans → $1.70, grafana: 950k billable → $0.475
	est := mc.EstimateCost(stats, []string{"datadog", "grafana"})

	wantTotal := 1.70 + 0.475
	if abs(est.TotalCostUSD-wantTotal) > 0.001 {
		t.Errorf("TotalCostUSD: got %.4f, want %.4f", est.TotalCostUSD, wantTotal)
	}
}

func TestEstimateCost_UnknownBackend(t *testing.T) {
	mc := NewMetricsCollector("http://localhost:8888/metrics")
	stats := makeStats(0, 0)
	est := mc.EstimateCost(stats, []string{"splunk"})

	if est.TotalCostUSD != 0 {
		t.Errorf("TotalCostUSD should be 0 for unknown backend, got %.4f", est.TotalCostUSD)
	}
	if !strings.Contains(est.PricingNotes, "splunk") {
		t.Errorf("PricingNotes should mention unknown backend name, got: %s", est.PricingNotes)
	}
}

func TestEstimateCost_NilBackends(t *testing.T) {
	mc := NewMetricsCollector("http://localhost:8888/metrics")
	stats := makeStats(999_999, 999_999)
	est := mc.EstimateCost(stats, nil)

	if est.TotalCostUSD != 0 {
		t.Errorf("TotalCostUSD should be 0 when no backends given, got %.4f", est.TotalCostUSD)
	}
	if est.PricingNotes != "" {
		t.Errorf("PricingNotes should be empty with no backends, got: %s", est.PricingNotes)
	}
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
