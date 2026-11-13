package ebpf

import (
	"strings"
	"testing"
)

func TestValidate_Valid(t *testing.T) {
	cfg := EBPFConfig{
		Enabled:    true,
		Tool:       "beyla",
		TargetPort: 8080,
	}
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected valid config, got error: %v", err)
	}
}

func TestValidate_UnsupportedTool(t *testing.T) {
	cfg := EBPFConfig{
		Tool:       "unknown-tool",
		TargetPort: 8080,
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for unsupported tool, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported eBPF tool") {
		t.Errorf("expected 'unsupported eBPF tool' in error, got: %v", err)
	}
}

func TestValidate_ZeroPort(t *testing.T) {
	cfg := EBPFConfig{
		Tool:       "beyla",
		TargetPort: 0,
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatal("expected error for zero port, got nil")
	}
	if !strings.Contains(err.Error(), "target port") {
		t.Errorf("expected 'target port' in error, got: %v", err)
	}
}

func TestGenerateBeylaDockerCompose(t *testing.T) {
	cfg := EBPFConfig{
		Tool:       "beyla",
		TargetPort: 5050,
	}
	out := GenerateBeylaDockerCompose(cfg)

	expectedSubstrings := []string{
		"grafana/beyla:latest",
		"BEYLA_OPEN_PORT=5050",
		"OTEL_EXPORTER_OTLP_ENDPOINT",
		"NET_ADMIN",
		"SYS_ADMIN",
		"/sys/kernel/debug",
	}

	for _, s := range expectedSubstrings {
		if !strings.Contains(out, s) {
			t.Errorf("expected docker-compose to contain %q", s)
		}
	}
}

func TestGenerateBeylaK8sManifest(t *testing.T) {
	cfg := EBPFConfig{
		Tool:       "beyla",
		TargetPort: 8080,
	}
	out := GenerateBeylaK8sManifest(cfg)

	expectedSubstrings := []string{
		"kind: DaemonSet",
		"name: beyla-ebpf",
		"hostPID: true",
		"hostNetwork: true",
		"BEYLA_OPEN_PORT",
		"SYS_ADMIN",
	}

	for _, s := range expectedSubstrings {
		if !strings.Contains(out, s) {
			t.Errorf("expected K8s manifest to contain %q", s)
		}
	}
}

