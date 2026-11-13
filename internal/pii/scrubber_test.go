package pii

import (
	"fmt"
	"strings"
	"testing"
)

func TestProcessorName(t *testing.T) {
	cfg := PIIConfig{Enabled: true, Replacement: "[REDACTED]"}
	if got := cfg.ProcessorName(); got != "transform/pii_scrubber" {
		t.Errorf("ProcessorName() = %q, want %q", got, "transform/pii_scrubber")
	}
}

func TestGeneratePIIScrubberYAML_ContainsProcessorName(t *testing.T) {
	cfg := PIIConfig{Enabled: true, Replacement: "[REDACTED]"}
	out := GeneratePIIScrubberYAML(cfg)

	if !strings.Contains(out, "transform/pii_scrubber") {
		t.Errorf("expected output to contain processor name 'transform/pii_scrubber', got:\n%s", out)
	}
}

func TestGeneratePIIScrubberYAML_ContainsReplacePattern(t *testing.T) {
	cfg := PIIConfig{Enabled: true, Replacement: "[REDACTED]"}
	out := GeneratePIIScrubberYAML(cfg)

	if !strings.Contains(out, "replace_pattern") {
		t.Errorf("expected output to contain 'replace_pattern', got:\n%s", out)
	}
}

func TestGeneratePIIScrubberYAML_ContainsAllDefaultPatterns(t *testing.T) {
	cfg := PIIConfig{Enabled: true, Replacement: "[REDACTED]"}
	out := GeneratePIIScrubberYAML(cfg)

	for _, dp := range DefaultPatterns {
		if !strings.Contains(out, fmt.Sprintf("%q", dp.Pattern)) {
			t.Errorf("expected output to contain pattern for %s (%q), not found in:\n%s", dp.Name, dp.Pattern, out)
		}
	}
}

func TestGeneratePIIScrubberYAML_CreditCard(t *testing.T) {
	cfg := PIIConfig{Enabled: true, Replacement: "[REDACTED]"}
	out := GeneratePIIScrubberYAML(cfg)
	ccPattern := `\b(?:4[0-9]{12}(?:[0-9]{3})?|5[1-5][0-9]{14}|3[47][0-9]{13})\b`
	if !strings.Contains(out, fmt.Sprintf("%q", ccPattern)) {
		t.Errorf("expected output to contain credit card pattern, got:\n%s", out)
	}
}

func TestGeneratePIIScrubberYAML_Email(t *testing.T) {
	cfg := PIIConfig{Enabled: true, Replacement: "[REDACTED]"}
	out := GeneratePIIScrubberYAML(cfg)
	emailPattern := `[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`
	if !strings.Contains(out, fmt.Sprintf("%q", emailPattern)) {
		t.Errorf("expected output to contain email pattern, got:\n%s", out)
	}
}

func TestGeneratePIIScrubberYAML_Password(t *testing.T) {
	cfg := PIIConfig{Enabled: true, Replacement: "[REDACTED]"}
	out := GeneratePIIScrubberYAML(cfg)
	pwdPattern := `(?i)(password|passwd|pwd|secret|token|api_key)\s*[=:]\s*\S+`
	if !strings.Contains(out, fmt.Sprintf("%q", pwdPattern)) {
		t.Errorf("expected output to contain password pattern, got:\n%s", out)
	}
}

func TestGeneratePIIScrubberYAML_SSN(t *testing.T) {
	cfg := PIIConfig{Enabled: true, Replacement: "[REDACTED]"}
	out := GeneratePIIScrubberYAML(cfg)
	ssnPattern := `\b\d{3}-\d{2}-\d{4}\b`
	if !strings.Contains(out, fmt.Sprintf("%q", ssnPattern)) {
		t.Errorf("expected output to contain SSN pattern, got:\n%s", out)
	}
}

func TestGeneratePIIScrubberYAML_CustomReplacement(t *testing.T) {
	cfg := PIIConfig{Enabled: true, Replacement: "***"}
	out := GeneratePIIScrubberYAML(cfg)
	if !strings.Contains(out, "***") {
		t.Errorf("expected output to contain custom replacement '***', got:\n%s", out)
	}
}

func TestGeneratePIIScrubberYAML_DefaultReplacement(t *testing.T) {
	cfg := PIIConfig{Enabled: true}
	out := GeneratePIIScrubberYAML(cfg)
	if !strings.Contains(out, "[REDACTED]") {
		t.Errorf("expected default replacement '[REDACTED]' in output, got:\n%s", out)
	}
}

func TestGeneratePIIScrubberYAML_CustomPatterns(t *testing.T) {
	cfg := PIIConfig{
		Enabled:        true,
		Replacement:    "[REDACTED]",
		CustomPatterns: []string{`\bACCT-\d{8}\b`},
	}
	out := GeneratePIIScrubberYAML(cfg)
	if !strings.Contains(out, fmt.Sprintf("%q", `\bACCT-\d{8}\b`)) {
		t.Errorf("expected output to contain custom pattern, got:\n%s", out)
	}
}

func TestGeneratePIIScrubberYAML_LogStatements(t *testing.T) {
	cfg := PIIConfig{Enabled: true, Replacement: "[REDACTED]"}
	out := GeneratePIIScrubberYAML(cfg)
	if !strings.Contains(out, "log_statements") {
		t.Errorf("expected output to contain 'log_statements', got:\n%s", out)
	}
	if !strings.Contains(out, "replace_pattern(body,") {
		t.Errorf("expected output to contain body replace_pattern, got:\n%s", out)
	}
}

func TestGeneratePIIScrubberYAML_TraceStatements(t *testing.T) {
	cfg := PIIConfig{Enabled: true, Replacement: "[REDACTED]"}
	out := GeneratePIIScrubberYAML(cfg)
	if !strings.Contains(out, "trace_statements") {
		t.Errorf("expected output to contain 'trace_statements', got:\n%s", out)
	}
}

func TestGeneratePIIScrubberYAML_MetricStatements(t *testing.T) {
	cfg := PIIConfig{Enabled: true, Replacement: "[REDACTED]"}
	out := GeneratePIIScrubberYAML(cfg)
	if !strings.Contains(out, "metric_statements") {
		t.Errorf("expected output to contain 'metric_statements', got:\n%s", out)
	}
}
