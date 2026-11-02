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

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Scaffold a new ObserveAll project with OTel collector config",
	Long: `Interactively set up ObserveAll for your environment.
Generates:
  - observeall.yaml       (project config)
  - collector-config.yaml (OTel Collector pipeline config)
  - docker-compose.yaml   (local dev stack, optional)`,
	RunE: runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	fmt.Println()
	infof("Welcome to ObserveAll setup wizard")
	fmt.Println()

	// 1. Project name
	namePrompt := promptui.Prompt{
		Label:   "Project name",
		Default: "my-observeall-project",
	}
	projectName, err := namePrompt.Run()
	if err != nil {
		return err
	}

	// 2. Environment
	envPrompt := promptui.Select{
		Label: "Deployment environment",
		Items: []string{"kubernetes", "docker-compose", "bare-metal", "serverless"},
	}
	_, environment, err := envPrompt.Run()
	if err != nil {
		return err
	}

	// 3. Languages to instrument
	infof("Select languages to auto-instrument (space to toggle, enter to confirm)")
	langs := []string{"java", "python", "nodejs", "go", "dotnet", "ruby"}
	langPrompt := promptui.Select{
		Label: "Primary application language",
		Items: langs,
	}
	_, language, err := langPrompt.Run()
	if err != nil {
		return err
	}

	// 4. Backends
	backendPrompt := promptui.Select{
		Label: "Observability backends (select primary — you can add more with `observeall connect`)",
		Items: []string{
			"prometheus+grafana",
			"datadog",
			"dynatrace",
			"newrelic",
			"jaeger",
			"custom-otlp",
		},
	}
	_, backend, err := backendPrompt.Run()
	if err != nil {
		return err
	}

	// 5. Signal types
	signalPrompt := promptui.Select{
		Label: "Signal types to collect",
		Items: []string{"traces+metrics+logs", "traces+metrics", "metrics+logs", "traces only"},
	}
	_, signals, err := signalPrompt.Run()
	if err != nil {
		return err
	}

	// 6. Sampling strategy
	samplingPrompt := promptui.Select{
		Label: "Sampling strategy",
		Items: []string{"tail-sampling (recommended)", "head-sampling (1%)", "always-on (100%)", "none"},
	}
	_, sampling, err := samplingPrompt.Run()
	if err != nil {
		return err
	}

	// Build project config
	proj := config.Project{
		Name:        projectName,
		Environment: environment,
		Language:    language,
		Backends:    []string{backend},
		Signals:     parseSignals(signals),
		Sampling:    parseSampling(sampling),
	}

	// Write observeall.yaml
	projBytes, _ := yaml.Marshal(proj)
	if err := os.WriteFile("observeall.yaml", projBytes, 0644); err != nil {
		return fmt.Errorf("failed to write observeall.yaml: %w", err)
	}
	successf("Created observeall.yaml")

	// Generate collector config
	collectorCfg, err := config.GenerateCollectorConfig(proj)
	if err != nil {
		return fmt.Errorf("failed to generate collector config: %w", err)
	}
	if err := os.WriteFile("collector-config.yaml", []byte(collectorCfg), 0644); err != nil {
		return fmt.Errorf("failed to write collector-config.yaml: %w", err)
	}
	successf("Created collector-config.yaml")

	// Generate docker-compose if needed
	if environment == "docker-compose" {
		dc, err := config.GenerateDockerCompose(proj)
		if err != nil {
			return fmt.Errorf("failed to generate docker-compose.yaml: %w", err)
		}
		if err := os.WriteFile("docker-compose.yaml", []byte(dc), 0644); err != nil {
			return fmt.Errorf("failed to write docker-compose.yaml: %w", err)
		}
		successf("Created docker-compose.yaml")
	}

	// Print next steps
	fmt.Println()
	infof("Setup complete! Next steps:")
	fmt.Printf("\n  Language instrumentation for %s:\n", language)
	printInstrInstructions(language, projectName)
	fmt.Println()
	fmt.Printf("  Run the collector:\n")
	if environment == "docker-compose" {
		fmt.Println("    docker compose up -d")
	} else if environment == "kubernetes" {
		fmt.Println("    helm install observeall ./helm/observeall -f observeall.yaml")
	} else {
		fmt.Println("    otelcol --config collector-config.yaml")
	}
	fmt.Println()
	fmt.Printf("  Add more backends:\n")
	fmt.Println("    observeall connect --backend datadog")
	fmt.Println("    observeall connect --backend dynatrace")
	fmt.Println()

	return nil
}

func parseSignals(s string) []string {
	switch {
	case strings.HasPrefix(s, "traces+metrics+logs"):
		return []string{"traces", "metrics", "logs"}
	case s == "traces+metrics":
		return []string{"traces", "metrics"}
	case s == "metrics+logs":
		return []string{"metrics", "logs"}
	default:
		return []string{"traces"}
	}
}

func parseSampling(s string) string {
	if strings.HasPrefix(s, "tail") {
		return "tail"
	} else if strings.HasPrefix(s, "head") {
		return "head"
	} else if strings.HasPrefix(s, "always") {
		return "always_on"
	}
	return "none"
}

func printInstrInstructions(lang, project string) {
	switch lang {
	case "java":
		fmt.Printf("    # Download the OTel Java agent\n")
		fmt.Printf("    wget https://github.com/open-telemetry/opentelemetry-java-instrumentation/releases/latest/download/opentelemetry-javaagent.jar\n")
		fmt.Printf("    # Run your app with the agent\n")
		fmt.Printf("    java -javaagent:opentelemetry-javaagent.jar \\\n")
		fmt.Printf("         -Dotel.service.name=%s \\\n", project)
		fmt.Printf("         -Dotel.exporter.otlp.endpoint=http://localhost:4317 \\\n")
		fmt.Printf("         -jar your-app.jar\n")
	case "python":
		fmt.Printf("    pip install opentelemetry-distro opentelemetry-exporter-otlp\n")
		fmt.Printf("    opentelemetry-bootstrap -a install\n")
		fmt.Printf("    OTEL_SERVICE_NAME=%s \\\n", project)
		fmt.Printf("    OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317 \\\n")
		fmt.Printf("    opentelemetry-instrument python your_app.py\n")
	case "nodejs":
		fmt.Printf("    npm install @opentelemetry/auto-instrumentations-node\n")
		fmt.Printf("    OTEL_SERVICE_NAME=%s \\\n", project)
		fmt.Printf("    OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4317 \\\n")
		fmt.Printf("    node --require @opentelemetry/auto-instrumentations-node/register app.js\n")
	case "go":
		fmt.Printf("    # Add OTel SDK to your Go app\n")
		fmt.Printf("    go get go.opentelemetry.io/otel \\\n")
		fmt.Printf("           go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc\n")
		fmt.Printf("    # See examples/go/ for bootstrap code\n")
	default:
		fmt.Printf("    See https://opentelemetry.io/docs/instrumentation/%s/ for setup instructions\n", lang)
	}
}

