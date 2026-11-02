package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/manifoldco/promptui"
	"github.com/observeall/observeall/internal/config"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var connectCmd = &cobra.Command{
	Use:   "connect",
	Short: "Connect an observability backend to your pipeline",
	Long: `Add or update a backend exporter in your OTel Collector pipeline.

Supported backends:
  prometheus, grafana, datadog, dynatrace, newrelic, jaeger, elastic, splunk, custom-otlp

Examples:
  observeall connect --backend datadog
  observeall connect --backend prometheus
  observeall connect --backend custom-otlp --endpoint http://my-collector:4317`,
	RunE: runConnect,
}

var (
	connectBackend  string
	connectEndpoint string
)

func init() {
	connectCmd.Flags().StringVar(&connectBackend, "backend", "", "Backend to connect (required)")
	connectCmd.Flags().StringVar(&connectEndpoint, "endpoint", "", "Custom OTLP endpoint URL")
	_ = connectCmd.MarkFlagRequired("backend")
}

func runConnect(cmd *cobra.Command, args []string) error {
	// Load existing project config
	proj, err := loadProjectConfig(cmd)
	if err != nil {
		return err
	}

	backend := strings.ToLower(connectBackend)

	// Collect backend credentials interactively
	backendCfg, err := promptBackendConfig(backend, connectEndpoint)
	if err != nil {
		return err
	}

	// Add backend to project config
	if !contains(proj.Backends, backend) {
		proj.Backends = append(proj.Backends, backend)
	}
	if proj.BackendConfigs == nil {
		proj.BackendConfigs = make(map[string]config.BackendConfig)
	}
	proj.BackendConfigs[backend] = backendCfg

	// Write updated project config
	projBytes, _ := yaml.Marshal(proj)
	cfgFile, _ := cmd.Flags().GetString("config")
	if err := os.WriteFile(cfgFile, projBytes, 0644); err != nil {
		return fmt.Errorf("failed to update %s: %w", cfgFile, err)
	}
	successf("Updated %s", cfgFile)

	// Regenerate collector config
	collectorCfg, err := config.GenerateCollectorConfig(proj)
	if err != nil {
		return fmt.Errorf("failed to regenerate collector config: %w", err)
	}
	if err := os.WriteFile("collector-config.yaml", []byte(collectorCfg), 0644); err != nil {
		return fmt.Errorf("failed to write collector-config.yaml: %w", err)
	}
	successf("Regenerated collector-config.yaml with %s backend", backend)

	fmt.Println()
	infof("Backend '%s' connected. Restart your collector to apply changes:", backend)
	fmt.Println("  docker compose restart otel-collector")
	fmt.Println("  # or: kubectl rollout restart deployment/observeall-collector")
	fmt.Println()

	return nil
}

func promptBackendConfig(backend, endpoint string) (config.BackendConfig, error) {
	cfg := config.BackendConfig{Type: backend}

	switch backend {
	case "datadog":
		prompt := promptui.Prompt{Label: "Datadog API Key", Mask: '*'}
		key, err := prompt.Run()
		if err != nil {
			return cfg, err
		}
		cfg.APIKey = key

		sitePrompt := promptui.Select{
			Label: "Datadog Site",
			Items: []string{"datadoghq.com", "datadoghq.eu", "us3.datadoghq.com", "us5.datadoghq.com"},
		}
		_, site, err := sitePrompt.Run()
		if err != nil {
			return cfg, err
		}
		cfg.Site = site

	case "dynatrace":
		if endpoint != "" {
			cfg.Endpoint = endpoint
		} else {
			epPrompt := promptui.Prompt{Label: "Dynatrace OTLP Endpoint (e.g. https://{env-id}.live.dynatrace.com/api/v2/otlp)"}
			ep, err := epPrompt.Run()
			if err != nil {
				return cfg, err
			}
			cfg.Endpoint = ep
		}
		tokenPrompt := promptui.Prompt{Label: "Dynatrace API Token", Mask: '*'}
		token, err := tokenPrompt.Run()
		if err != nil {
			return cfg, err
		}
		cfg.APIKey = token

	case "grafana":
		epPrompt := promptui.Prompt{
			Label:   "Grafana Cloud OTLP endpoint",
			Default: "https://otlp-gateway-prod-us-central-0.grafana.net/otlp",
		}
		ep, err := epPrompt.Run()
		if err != nil {
			return cfg, err
		}
		cfg.Endpoint = ep
		tokenPrompt := promptui.Prompt{Label: "Grafana Cloud API token (Base64 instanceId:token)", Mask: '*'}
		token, err := tokenPrompt.Run()
		if err != nil {
			return cfg, err
		}
		cfg.APIKey = token

	case "prometheus":
		epPrompt := promptui.Prompt{
			Label:   "Prometheus Remote Write endpoint",
			Default: "http://prometheus:9090/api/v1/write",
		}
		ep, err := epPrompt.Run()
		if err != nil {
			return cfg, err
		}
		cfg.Endpoint = ep

	case "newrelic":
		tokenPrompt := promptui.Prompt{Label: "New Relic License Key (INGEST)", Mask: '*'}
		token, err := tokenPrompt.Run()
		if err != nil {
			return cfg, err
		}
		cfg.APIKey = token
		cfg.Endpoint = "https://otlp.nr-data.net"

	case "jaeger":
		epPrompt := promptui.Prompt{
			Label:   "Jaeger OTLP gRPC endpoint",
			Default: "http://jaeger:4317",
		}
		ep, err := epPrompt.Run()
		if err != nil {
			return cfg, err
		}
		cfg.Endpoint = ep

	case "custom-otlp":
		if endpoint != "" {
			cfg.Endpoint = endpoint
		} else {
			epPrompt := promptui.Prompt{Label: "Custom OTLP gRPC/HTTP endpoint"}
			ep, err := epPrompt.Run()
			if err != nil {
				return cfg, err
			}
			cfg.Endpoint = ep
		}
	}

	return cfg, nil
}

func loadProjectConfig(cmd *cobra.Command) (config.Project, error) {
	cfgFile, _ := cmd.Flags().GetString("config")
	data, err := os.ReadFile(cfgFile)
	if err != nil {
		return config.Project{}, fmt.Errorf("no observeall.yaml found — run `observeall init` first: %w", err)
	}
	var proj config.Project
	if err := yaml.Unmarshal(data, &proj); err != nil {
		return config.Project{}, fmt.Errorf("failed to parse %s: %w", cfgFile, err)
	}
	return proj, nil
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

