// Package analytics provides real-time ingestion rate and cost breakdown
// analytics by polling the OpenTelemetry collector's Prometheus metrics endpoint.
package analytics

import (
	"bufio"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// MetricsCollector polls an OTel collector Prometheus endpoint for ingestion stats.
type MetricsCollector struct {
	collectorURL string
	httpClient   *http.Client
}

// IngestStats holds a snapshot of collector ingestion counters.
type IngestStats struct {
	AcceptedSpans        int64
	AcceptedMetricPoints int64
	RefusedSpans         int64
	ExportedPoints       int64
	FailedExports        int64
	// SampledDroppedPercent is the percentage of spans refused vs accepted+refused.
	SampledDroppedPercent float64
	CollectedAt           time.Time
}

// CostEstimate holds a per-backend monthly cost breakdown in USD.
type CostEstimate struct {
	SpansCostUSD   float64
	MetricsCostUSD float64
	TotalCostUSD   float64
	PricingNotes   string
}

// NewMetricsCollector constructs a MetricsCollector targeting the given URL.
// collectorURL should be the full URL to the Prometheus metrics endpoint,
// e.g. "http://localhost:8888/metrics".
func NewMetricsCollector(collectorURL string) *MetricsCollector {
	return NewMetricsCollectorWithClient(collectorURL, &http.Client{
		Timeout: 10 * time.Second,
	})
}

// NewMetricsCollectorWithClient constructs a MetricsCollector with a custom http.Client.
func NewMetricsCollectorWithClient(collectorURL string, client *http.Client) *MetricsCollector {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	return &MetricsCollector{
		collectorURL: collectorURL,
		httpClient:   client,
	}
}

// FetchStats performs an HTTP GET against the collector's /metrics endpoint,
// parses the Prometheus text format, and returns an IngestStats snapshot.
func (c *MetricsCollector) FetchStats() (IngestStats, error) {
	resp, err := c.httpClient.Get(c.collectorURL)
	if err != nil {
		return IngestStats{}, fmt.Errorf("failed to reach collector at %s: %w", c.collectorURL, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return IngestStats{}, fmt.Errorf("collector returned HTTP %d", resp.StatusCode)
	}

	stats, err := parsePrometheusMetrics(resp.Body)
	if err != nil {
		return IngestStats{}, fmt.Errorf("failed to parse Prometheus metrics: %w", err)
	}
	stats.CollectedAt = time.Now()

	// Compute dropped percentage
	total := stats.AcceptedSpans + stats.RefusedSpans
	if total > 0 {
		stats.SampledDroppedPercent = float64(stats.RefusedSpans) / float64(total) * 100.0
	}

	return stats, nil
}

// parsePrometheusMetrics reads Prometheus text-format lines from r and
// extracts the OTel collector counters we care about.
func parsePrometheusMetrics(r io.Reader) (IngestStats, error) {
	// Metric names we're interested in (without labels).
	const (
		keyAcceptedSpans   = "otelcol_receiver_accepted_spans"
		keyAcceptedMetrics = "otelcol_receiver_accepted_metric_points"
		keyRefusedSpans    = "otelcol_receiver_refused_spans"
		keyExportedPoints  = "otelcol_exporter_sent_metric_points"
		keyFailedExports   = "otelcol_exporter_send_failed_metric_points"
	)

	var stats IngestStats
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		// Skip comments and empty lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Extract metric name (before optional labels or space)
		metricName, rawValue, found := parsePromLine(line)
		if !found {
			continue
		}

		val, err := strconv.ParseFloat(rawValue, 64)
		if err != nil {
			continue
		}
		intVal := int64(val)

		// Accumulate values (there may be multiple label variants per metric)
		switch metricName {
		case keyAcceptedSpans:
			stats.AcceptedSpans += intVal
		case keyAcceptedMetrics:
			stats.AcceptedMetricPoints += intVal
		case keyRefusedSpans:
			stats.RefusedSpans += intVal
		case keyExportedPoints:
			stats.ExportedPoints += intVal
		case keyFailedExports:
			stats.FailedExports += intVal
		}
	}
	if err := scanner.Err(); err != nil {
		return stats, err
	}
	return stats, nil
}

// parsePromLine parses a single non-comment Prometheus text line and returns
// (metricName, value, true) or ("", "", false) if the line cannot be parsed.
//
// Prometheus text format:
//
//	metric_name[{labels}] value [timestamp]
func parsePromLine(line string) (name, value string, ok bool) {
	// Find the last space-separated token that is the value (and optional timestamp)
	// First split off the metric name (and labels) from the value.
	spaceIdx := strings.LastIndex(line, " ")
	if spaceIdx < 0 {
		return "", "", false
	}
	nameAndLabels := strings.TrimSpace(line[:spaceIdx])
	valueField := strings.TrimSpace(line[spaceIdx+1:])

	// If there is a timestamp after the value, skip it
	if idx := strings.Index(valueField, " "); idx >= 0 {
		valueField = valueField[:idx]
	}

	// Strip labels: "metric_name{...}" → "metric_name"
	if idx := strings.Index(nameAndLabels, "{"); idx >= 0 {
		nameAndLabels = nameAndLabels[:idx]
	}

	name = strings.TrimSpace(nameAndLabels)
	if name == "" || valueField == "" {
		return "", "", false
	}
	return name, valueField, true
}

// EstimateCost calculates a monthly cost estimate for the given IngestStats
// across the provided backend list.
//
// Pricing assumptions (as of 2024, illustrative):
//   - datadog:  $1.70 per million spans/month; metrics free tier then $0.10/million
//   - newrelic: $0.30 per GB of metric data/month (assume 1 KB per point)
//   - grafana:  free tier covers 50,000 spans/month; $0.50/million spans beyond
//
// Stats are treated as cumulative counters observed at collection time, so the
// caller is responsible for computing rates (e.g. per-minute) if desired.
// EstimateCost operates on the raw snapshot totals for a "current moment" estimate.
func (c *MetricsCollector) EstimateCost(stats IngestStats, backends []string) CostEstimate {
	var estimate CostEstimate
	var notes []string

	spansMillions := float64(stats.AcceptedSpans) / 1_000_000.0
	// Metric points: assume 1 KB per point → GB
	metricGB := float64(stats.AcceptedMetricPoints) * 1024.0 / (1024.0 * 1024.0 * 1024.0)

	for _, backend := range backends {
		switch strings.ToLower(backend) {
		case "datadog":
			spanCost := spansMillions * 1.70
			estimate.SpansCostUSD += spanCost
			notes = append(notes, fmt.Sprintf("Datadog: $1.70/M spans (%.2f spans → $%.4f)", float64(stats.AcceptedSpans), spanCost))

		case "newrelic":
			metricCost := metricGB * 0.30
			estimate.MetricsCostUSD += metricCost
			notes = append(notes, fmt.Sprintf("New Relic: $0.30/GB metrics (%.3f GB → $%.4f)", metricGB, metricCost))

		case "grafana":
			const freeTierSpans = 50_000
			billableSpans := int64(0)
			if stats.AcceptedSpans > freeTierSpans {
				billableSpans = stats.AcceptedSpans - freeTierSpans
			}
			billableMillions := float64(billableSpans) / 1_000_000.0
			spanCost := billableMillions * 0.50
			estimate.SpansCostUSD += spanCost
			notes = append(notes, fmt.Sprintf("Grafana Cloud: free up to 50k spans, $0.50/M beyond (%d spans → $%.4f)", stats.AcceptedSpans, spanCost))

		default:
			notes = append(notes, fmt.Sprintf("%s: no pricing model available", backend))
		}
	}

	estimate.TotalCostUSD = estimate.SpansCostUSD + estimate.MetricsCostUSD
	estimate.PricingNotes = strings.Join(notes, "; ")
	return estimate
}
