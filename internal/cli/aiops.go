package cli

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/observeall/observeall/internal/aiops"
	"github.com/spf13/cobra"
)

var (
	aiopsService  string
	aiopsSeverity string
	aiopsScenario string
)

var aiopsCmd = &cobra.Command{
	Use:   "aiops",
	Short: "AIOps cross-signal automated root-cause analysis (LLM-assisted)",
	Long: `Correlates signals across metrics, distributed traces, and log exceptions
to diagnose incident root causes and generate actionable remediation recommendations.`,
	RunE: runAIOps,
}

func init() {
	aiopsCmd.Flags().StringVarP(&aiopsService, "service", "s", "dummy-shop-service", "Service name to analyze")
	aiopsCmd.Flags().StringVar(&aiopsSeverity, "severity", "CRITICAL", "Incident severity (CRITICAL, WARNING, INFO)")
	aiopsCmd.Flags().StringVar(&aiopsScenario, "scenario", "db_exhaustion", "Incident scenario to simulate/diagnose: db_exhaustion, oom, latency, app_error")
}

func runAIOps(cmd *cobra.Command, args []string) error {
	analyzer := aiops.NewAnalyzer("gemini-1.5-pro / claude-3.5-sonnet")

	var inc aiops.Incident

	switch aiopsScenario {
	case "oom":
		inc = aiops.Incident{
			ID:        fmt.Sprintf("inc-%d", time.Now().Unix()%10000),
			Service:   aiopsService,
			Trigger:   "Pod Evicted / CrashLoopBackOff",
			Timestamp: time.Now(),
			Severity:  aiops.SeverityCritical,
			Metrics: []aiops.MetricAnomaly{
				{
					MetricName:    "container_memory_working_set_bytes",
					BaselineValue: 256.0,
					ObservedValue: 512.0,
					DeltaPercent:  100.0,
					Unit:          "MiB",
				},
			},
			Logs: []aiops.LogRecord{
				{
					Timestamp: time.Now(),
					Level:     "FATAL",
					Body:      "java.lang.OutOfMemoryError: Java heap space exceeded container memory limit",
				},
			},
		}
	case "latency":
		inc = aiops.Incident{
			ID:        fmt.Sprintf("inc-%d", time.Now().Unix()%10000),
			Service:   aiopsService,
			Trigger:   "P99 Latency SLA Breach (> 500ms)",
			Timestamp: time.Now(),
			Severity:  aiops.SeverityWarning,
			Spans: []aiops.TraceSpan{
				{
					TraceID:    "trace-slow-873910ab",
					SpanID:     "span-slow-01",
					Name:       "checkout_payment_gateway",
					Service:    aiopsService,
					DurationMs: 820.0,
					Status:     "OK",
				},
			},
			Metrics: []aiops.MetricAnomaly{
				{
					MetricName:    "http_request_duration_seconds{quantile=\"0.99\"}",
					BaselineValue: 0.12,
					ObservedValue: 0.82,
					DeltaPercent:  583.3,
					Unit:          "s",
				},
			},
		}
	case "app_error":
		inc = aiops.Incident{
			ID:        fmt.Sprintf("inc-%d", time.Now().Unix()%10000),
			Service:   aiopsService,
			Trigger:   "HTTP 500 Spike on /error",
			Timestamp: time.Now(),
			Severity:  aiops.SeverityWarning,
			Spans: []aiops.TraceSpan{
				{
					TraceID:      "trace-err-9921ef4",
					SpanID:       "span-err-01",
					Name:         "failing_operation",
					Service:      aiopsService,
					Status:       "ERROR",
					ErrorMessage: "ValueError: Simulated database connection failure",
				},
			},
			Logs: []aiops.LogRecord{
				{
					Timestamp: time.Now(),
					Level:     "ERROR",
					Body:      "ValueError: Simulated database connection failure at app.py:141",
				},
			},
		}
	default: // db_exhaustion
		inc = aiops.Incident{
			ID:        fmt.Sprintf("inc-%d", time.Now().Unix()%10000),
			Service:   aiopsService,
			Trigger:   "Connection Pool Exhaustion / 500 Spike on /order",
			Timestamp: time.Now(),
			Severity:  aiops.SeverityCritical,
			Spans: []aiops.TraceSpan{
				{
					TraceID:      "trace-db-7102cd34",
					SpanID:       "span-db-01",
					Name:         "charge_payment",
					Service:      aiopsService,
					Status:       "ERROR",
					ErrorMessage: "Timeout waiting for idle database connection from pool after 15000ms",
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
				{
					MetricName:    "db_connection_pool_wait_seconds",
					BaselineValue: 0.02,
					ObservedValue: 15.0,
					DeltaPercent:  74900.0,
					Unit:          "s",
				},
			},
			Logs: []aiops.LogRecord{
				{
					Timestamp: time.Now(),
					Level:     "ERROR",
					Body:      "ConnectionRefusedError: Max connection pool capacity (100) reached; requests queued.",
				},
			},
		}
	}

	result, err := analyzer.Analyze(inc)
	if err != nil {
		return err
	}

	fmt.Println()
	color.Magenta("╔════════════════════════════════════════════════════════════════════════════════╗")
	color.Magenta("║          ObserveAll AIOps — Automated Root-Cause Analysis (LLM-Assisted)       ║")
	color.Magenta("╚════════════════════════════════════════════════════════════════════════════════╝")
	fmt.Println()

	fmt.Printf("  Incident ID : %s\n", color.YellowString(result.IncidentID))
	fmt.Printf("  Title       : %s\n", result.Title)
	fmt.Printf("  Analyzed At : %s\n", result.Timestamp.Format(time.RFC3339))
	fmt.Printf("  Confidence  : %s\n", color.GreenString("%.1f%%", result.ConfidenceScore*100))
	fmt.Println()

	color.Cyan("▶ Primary Culprit:")
	fmt.Printf("  %s\n\n", color.RedString(result.PrimaryCulprit))

	color.Cyan("▶ Root Cause Summary:")
	fmt.Printf("  %s\n\n", result.RootCause)

	color.Cyan("▶ Correlated Signals (Traces + Metrics + Logs):")
	for _, sig := range result.CorrelatedSignals {
		fmt.Printf("  • %s\n", sig)
	}
	fmt.Println()

	color.Cyan("▶ LLM Synthesis & Diagnosis:")
	fmt.Printf("  %s\n\n", result.LLMReasoning)

	color.Green("▶ Actionable Remediation Plan:")
	for i, rem := range result.ActionableRemediation {
		fmt.Printf("  %d. %s\n", i+1, rem)
	}
	fmt.Println()

	return nil
}

