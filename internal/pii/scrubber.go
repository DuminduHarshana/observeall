package pii

import (
	"fmt"
	"strings"
)

// PIIConfig holds configuration for the PII scrubbing processor.
type PIIConfig struct {
	Enabled        bool
	ScrubPatterns  []string
	CustomPatterns []string
	Replacement    string
}

// DefaultPatterns lists built-in regex patterns for common PII types,
// expressed as OTTL replace_pattern targets.
var DefaultPatterns = []struct {
	Name    string
	Pattern string
}{
	{
		Name:    "credit_card",
		Pattern: `\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13})\b`,
	},
	{
		Name:    "email",
		Pattern: `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`,
	},
	{
		Name:    "password_kv",
		Pattern: `(?i)(password|passwd|pwd|secret|token|api_key)\s*[=:]\s*\S+`,
	},
	{
		Name:    "us_ssn",
		Pattern: `\b\d{3}-\d{2}-\d{4}\b`,
	},
}

// ProcessorName returns the OTel processor name for the PII scrubber.
func (cfg PIIConfig) ProcessorName() string {
	return "transform/pii_scrubber"
}

// allPatterns returns the combined list of built-in and custom patterns.
func (cfg PIIConfig) allPatterns() []string {
	var patterns []string

	// Use ScrubPatterns if explicitly provided, else use DefaultPatterns.
	if len(cfg.ScrubPatterns) > 0 {
		patterns = append(patterns, cfg.ScrubPatterns...)
	} else {
		for _, dp := range DefaultPatterns {
			patterns = append(patterns, dp.Pattern)
		}
	}

	patterns = append(patterns, cfg.CustomPatterns...)
	return patterns
}

// replacement returns the configured replacement string or a sensible default.
func (cfg PIIConfig) replacement() string {
	if cfg.Replacement != "" {
		return cfg.Replacement
	}
	return "[REDACTED]"
}

// GeneratePIIScrubberYAML returns a YAML fragment for the OTel transform
// processor that applies replace_pattern to span attributes, log body, and
// metric data points.
func GeneratePIIScrubberYAML(cfg PIIConfig) string {
	patterns := cfg.allPatterns()
	replacement := cfg.replacement()

	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("  # PII scrubbing — redacts sensitive data in spans, logs, and metrics\n  %s:\n", cfg.ProcessorName()))
	sb.WriteString("    error_mode: ignore\n")
	sb.WriteString("    trace_statements:\n")
	sb.WriteString("      - context: span\n")
	sb.WriteString("        statements:\n")
	for _, pat := range patterns {
		sb.WriteString(fmt.Sprintf("          - replace_pattern(attributes[\"http.url\"], %q, %q)\n", pat, replacement))
		sb.WriteString(fmt.Sprintf("          - replace_pattern(attributes[\"db.statement\"], %q, %q)\n", pat, replacement))
		sb.WriteString(fmt.Sprintf("          - replace_pattern(attributes[\"rpc.request.metadata.authorization\"], %q, %q)\n", pat, replacement))
	}

	sb.WriteString("    log_statements:\n")
	sb.WriteString("      - context: log\n")
	sb.WriteString("        statements:\n")
	for _, pat := range patterns {
		sb.WriteString(fmt.Sprintf("          - replace_pattern(body, %q, %q)\n", pat, replacement))
	}

	sb.WriteString("    metric_statements:\n")
	sb.WriteString("      - context: datapoint\n")
	sb.WriteString("        statements:\n")
	for _, pat := range patterns {
		sb.WriteString(fmt.Sprintf("          - replace_pattern(attributes[\"label\"], %q, %q)\n", pat, replacement))
	}

	sb.WriteString("\n")
	return sb.String()
}
