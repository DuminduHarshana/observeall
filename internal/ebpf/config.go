package ebpf

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// SupportedTools lists the supported eBPF instrumentation engines.
var SupportedTools = []string{"beyla", "pixie", "coroot"}

// EBPFConfig holds settings for zero-code eBPF network instrumentation.
type EBPFConfig struct {
	Enabled           bool              `yaml:"enabled" json:"enabled"`
	Tool              string            `yaml:"tool" json:"tool"`
	TargetPort        int               `yaml:"target_port" json:"target_port"`
	OpenPorts         []int             `yaml:"open_ports,omitempty" json:"open_ports,omitempty"`
	PodLabels         map[string]string `yaml:"pod_labels,omitempty" json:"pod_labels,omitempty"`
	CollectorEndpoint string            `yaml:"collector_endpoint,omitempty" json:"collector_endpoint,omitempty"`
}

// Validate checks whether the eBPF configuration is valid.
func (cfg EBPFConfig) Validate() error {
	if cfg.Tool == "" {
		return fmt.Errorf("tool cannot be empty")
	}

	toolValid := false
	for _, t := range SupportedTools {
		if strings.EqualFold(cfg.Tool, t) {
			toolValid = true
			break
		}
	}
	if !toolValid {
		return fmt.Errorf("unsupported eBPF tool %q; supported tools: %s", cfg.Tool, strings.Join(SupportedTools, ", "))
	}

	if cfg.TargetPort <= 0 {
		return fmt.Errorf("target port must be greater than 0")
	}

	return nil
}

// DefaultCollectorEndpoint returns the OTLP gRPC endpoint for the collector.
func (cfg EBPFConfig) DefaultCollectorEndpoint(env string) string {
	if cfg.CollectorEndpoint != "" {
		return cfg.CollectorEndpoint
	}
	if env == "k8s" {
		return "http://observeall.observeall.svc.cluster.local:4317"
	}
	return "http://otel-collector:4317"
}

// GenerateBeylaDockerCompose returns a docker-compose service definition snippet
// for Grafana Beyla auto-instrumentation.
func GenerateBeylaDockerCompose(cfg EBPFConfig) string {
	endpoint := cfg.DefaultCollectorEndpoint("docker")
	tmpl := `# =============================================================
# ObserveAll — Grafana Beyla eBPF Auto-Instrumentation
# Captures HTTP/gRPC traffic without app changes or language agents
# =============================================================
version: "3.9"

services:
  beyla:
    image: grafana/beyla:latest
    container_name: beyla
    restart: unless-stopped
    pid: "host"
    network_mode: "host"
    cap_add:
      - SYS_ADMIN
      - NET_ADMIN
      - SYS_PTRACE
    environment:
      - BEYLA_OPEN_PORT={{.TargetPort}}
      - OTEL_EXPORTER_OTLP_ENDPOINT={{.Endpoint}}
      - BEYLA_PRINT_TRACES=false
    volumes:
      - /sys/kernel/debug:/sys/kernel/debug:ro
      - /proc:/proc:ro
`
	t := template.Must(template.New("compose").Parse(tmpl))
	var buf bytes.Buffer
	_ = t.Execute(&buf, map[string]any{
		"TargetPort": cfg.TargetPort,
		"Endpoint":   endpoint,
	})
	return buf.String()
}

// GenerateBeylaK8sManifest generates a Kubernetes DaemonSet YAML manifest for Beyla.
func GenerateBeylaK8sManifest(cfg EBPFConfig) string {
	endpoint := cfg.DefaultCollectorEndpoint("k8s")
	tmpl := `apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: beyla-ebpf
  namespace: observeall
  labels:
    app.kubernetes.io/name: beyla
    app.kubernetes.io/part-of: observeall
spec:
  selector:
    matchLabels:
      app: beyla
  template:
    metadata:
      labels:
        app: beyla
    spec:
      hostPID: true
      hostNetwork: true
      tolerations:
        - operator: Exists
      containers:
        - name: beyla
          image: grafana/beyla:latest
          securityContext:
            privileged: false
            capabilities:
              add:
                - SYS_ADMIN
                - NET_ADMIN
                - SYS_PTRACE
          env:
            - name: BEYLA_OPEN_PORT
              value: "{{.TargetPort}}"
            - name: OTEL_EXPORTER_OTLP_ENDPOINT
              value: "{{.Endpoint}}"
          volumeMounts:
            - mountPath: /sys/kernel/debug
              name: debugfs
              readOnly: true
            - mountPath: /proc
              name: proc
              readOnly: true
      volumes:
        - name: debugfs
          hostPath:
            path: /sys/kernel/debug
        - name: proc
          hostPath:
            path: /proc
`
	t := template.Must(template.New("k8s").Parse(tmpl))
	var buf bytes.Buffer
	_ = t.Execute(&buf, map[string]any{
		"TargetPort": cfg.TargetPort,
		"Endpoint":   endpoint,
	})
	return buf.String()
}

