package config

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestGenerateCollectorConfig_Default(t *testing.T) {
	proj := Project{
		Name:        "test-project",
		Environment: "production",
		Language:    "go",
		Backends:    []string{"prometheus"},
		Signals:     []string{"traces", "metrics"},
		Sampling:    "tail",
	}

	cfg, err := GenerateCollectorConfig(proj)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if !strings.Contains(cfg, "Project: test-project") {
		t.Errorf("expected project header to contain project name")
	}

	if !strings.Contains(cfg, "prometheusremotewrite:") {
		t.Errorf("expected prometheus exporter configuration")
	}

	if !strings.Contains(cfg, "tail_sampling:") {
		t.Errorf("expected tail_sampling processor when sampling is 'tail'")
	}

	// Verify valid YAML
	var parsed map[string]interface{}
	if err := yaml.Unmarshal([]byte(cfg), &parsed); err != nil {
		t.Fatalf("generated config is invalid YAML: %v", err)
	}

	service, ok := parsed["service"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected service section in config")
	}
	pipelines, ok := service["pipelines"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected pipelines section in service")
	}

	if _, exists := pipelines["traces"]; !exists {
		t.Errorf("expected traces pipeline to exist")
	}
	if _, exists := pipelines["metrics"]; !exists {
		t.Errorf("expected metrics pipeline to exist")
	}
	if _, exists := pipelines["logs"]; exists {
		t.Errorf("did not expect logs pipeline to exist")
	}
}

func TestGenerateCollectorConfig_MultiBackend(t *testing.T) {
	proj := Project{
		Name:        "enterprise-app",
		Environment: "staging",
		Language:    "java",
		Backends:    []string{"datadog", "dynatrace", "grafana", "jaeger"},
		Signals:     []string{"traces", "metrics", "logs"},
		Sampling:    "head",
		BackendConfigs: map[string]BackendConfig{
			"datadog": {
				APIKey: "mock-dd-key",
				Site:   "datadoghq.eu",
			},
			"dynatrace": {
				Endpoint: "https://mock.live.dynatrace.com/api/v2/otlp",
				APIKey:   "dt-token",
			},
		},
	}

	cfg, err := GenerateCollectorConfig(proj)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expectedSubstrings := []string{
		"datadog:",
		"mock-dd-key",
		"datadoghq.eu",
		"otlp/dynatrace:",
		"https://mock.live.dynatrace.com/api/v2/otlp",
		"otlp/grafana:",
		"otlp/jaeger:",
		"probabilistic_sampler:",
		"traces:",
		"metrics:",
		"logs:",
	}

	for _, sub := range expectedSubstrings {
		if !strings.Contains(cfg, sub) {
			t.Errorf("expected config to contain '%s'", sub)
		}
	}
}

func TestGenerateDockerCompose(t *testing.T) {
	proj := Project{
		Name: "local-app",
	}

	dc, err := GenerateDockerCompose(proj)
	if err != nil {
		t.Fatalf("expected no error generating docker-compose, got %v", err)
	}

	if !strings.Contains(dc, "otel-collector:") || !strings.Contains(dc, "prometheus:") {
		t.Errorf("docker-compose missing core services")
	}
}
