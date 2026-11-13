package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/observeall/observeall/internal/analytics"
	"github.com/spf13/cobra"
)

var analyticsCmd = &cobra.Command{
	Use:   "analytics",
	Short: "Show real-time ingestion rates and cost estimates for your backends",
	Long: `Fetch live metrics from the OTel collector and display:
  - Current ingestion rates (spans/min, metric-points/min)
  - Estimated monthly cloud cost per backend

Supported backends: datadog, newrelic, grafana

Examples:
  observeall analytics
  observeall analytics --backend datadog
  observeall analytics --backend datadog,newrelic`,
	RunE: runAnalytics,
}

var (
	analyticsBackend      string
	analyticsCollectorURL string
)

func init() {
	analyticsCmd.Flags().StringVar(&analyticsBackend, "backend", "datadog,newrelic,grafana",
		"Comma-separated list of backends to estimate costs for (datadog|newrelic|grafana)")
	analyticsCmd.Flags().StringVar(&analyticsCollectorURL, "collector-url", "http://localhost:8888/metrics",
		"Prometheus metrics endpoint of the OTel collector")
}

func runAnalytics(cmd *cobra.Command, args []string) error {
	backends := parseBackends(analyticsBackend)
	if len(backends) == 0 {
		return fmt.Errorf("no valid backends specified; choose from: datadog, newrelic, grafana")
	}

	mc := analytics.NewMetricsCollector(analyticsCollectorURL)

	color.Cyan("\n  ObserveAll — Ingestion Rate & Cost Breakdown")
	fmt.Println("  " + strings.Repeat("─", 52))

	infof("Fetching live stats from %s …", analyticsCollectorURL)

	// Fetch two samples 10 seconds apart to compute rates
	firstStats, err := mc.FetchStats()
	if err != nil {
		errorf("Could not reach collector: %v", err)
		errorf("Is your OTel collector running? (docker compose up otel-collector)")
		return err
	}

	infof("Waiting 10 s for rate calculation …")
	time.Sleep(10 * time.Second)

	secondStats, err := mc.FetchStats()
	if err != nil {
		errorf("Second fetch failed: %v", err)
		return err
	}

	// Compute per-minute rates from the two snapshots (10s apart → ×6 to get /min)
	deltaSpans := secondStats.AcceptedSpans - firstStats.AcceptedSpans
	deltaMetrics := secondStats.AcceptedMetricPoints - firstStats.AcceptedMetricPoints
	spansPerMin := deltaSpans * 6
	metricsPerMin := deltaMetrics * 6

	// Use the second (more recent) snapshot for the cost estimate
	estimate := mc.EstimateCost(secondStats, backends)

	// ---- Print stats table ----
	fmt.Println()
	bold := color.New(color.Bold)
	bold.Println("  Ingestion Rates (last 10s window, extrapolated to /min)")
	fmt.Println("  " + strings.Repeat("─", 52))
	printKV("  Spans / min", fmt.Sprintf("%d", spansPerMin))
	printKV("  Metric points / min", fmt.Sprintf("%d", metricsPerMin))
	printKV("  Refused spans (total)", fmt.Sprintf("%d", secondStats.RefusedSpans))
	printKV("  Dropped %", fmt.Sprintf("%.2f%%", secondStats.SampledDroppedPercent))

	fmt.Println()
	bold.Println("  Cumulative Totals (since collector start)")
	fmt.Println("  " + strings.Repeat("─", 52))
	printKV("  Accepted spans", fmt.Sprintf("%d", secondStats.AcceptedSpans))
	printKV("  Accepted metric points", fmt.Sprintf("%d", secondStats.AcceptedMetricPoints))
	printKV("  Exported points", fmt.Sprintf("%d", secondStats.ExportedPoints))
	printKV("  Failed exports", fmt.Sprintf("%d", secondStats.FailedExports))

	fmt.Println()
	bold.Println("  Estimated Monthly Cost Breakdown")
	fmt.Println("  " + strings.Repeat("─", 52))
	printKVColored("  Spans cost", fmt.Sprintf("$%.4f USD", estimate.SpansCostUSD))
	printKVColored("  Metrics cost", fmt.Sprintf("$%.4f USD", estimate.MetricsCostUSD))
	printKVColored("  Total estimated cost", fmt.Sprintf("$%.4f USD / month", estimate.TotalCostUSD))

	fmt.Println()
	bold.Println("  Pricing Notes")
	fmt.Println("  " + strings.Repeat("─", 52))
	for _, note := range strings.Split(estimate.PricingNotes, "; ") {
		if note != "" {
			fmt.Printf("  • %s\n", note)
		}
	}

	fmt.Println()
	color.HiBlack("  Collected at: %s", secondStats.CollectedAt.Format(time.RFC3339))
	color.HiBlack("  Note: cost estimates are based on cumulative counters since collector start.")
	fmt.Println()

	return nil
}

// parseBackends splits a comma-separated list of backend names and trims whitespace.
func parseBackends(raw string) []string {
	parts := strings.Split(raw, ",")
	var result []string
	for _, p := range parts {
		p = strings.ToLower(strings.TrimSpace(p))
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

// printKV prints a simple key: value row.
func printKV(key, value string) {
	fmt.Printf("%-30s %s\n", key+":", value)
}

// printKVColored prints a key: value row with the value highlighted in yellow.
func printKVColored(key, value string) {
	fmt.Printf("%-30s %s\n", key+":", color.YellowString(value))
}
