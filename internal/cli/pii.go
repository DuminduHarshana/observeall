package cli

import (
	"fmt"
	"os"

	"github.com/observeall/observeall/internal/config"
	"github.com/observeall/observeall/internal/pii"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

var (
	piiEnable      bool
	piiDisable     bool
	piiReplacement string
)

var piiCmd = &cobra.Command{
	Use:   "pii",
	Short: "Configure automatic PII scrubbing for traces, metrics, and logs",
	Long: `Enable or disable automatic PII scrubbing in the ObserveAll collector pipeline.
When enabled, sensitive data (credit cards, emails, passwords, SSNs) is automatically
redacted before leaving the collector.`,
	RunE: runPII,
}

func init() {
	piiCmd.Flags().BoolVar(&piiEnable, "enable", false, "Enable PII scrubbing")
	piiCmd.Flags().BoolVar(&piiDisable, "disable", false, "Disable PII scrubbing")
	piiCmd.Flags().StringVar(&piiReplacement, "replacement", "[REDACTED]", "Replacement string for redacted values")
}

func runPII(cmd *cobra.Command, args []string) error {
	if !piiEnable && !piiDisable {
		return fmt.Errorf("specify either --enable or --disable")
	}
	if piiEnable && piiDisable {
		return fmt.Errorf("cannot specify both --enable and --disable")
	}

	configFile, _ := cmd.Flags().GetString("config")
	data, err := os.ReadFile(configFile)
	if err != nil {
		return fmt.Errorf("reading %s: %w", configFile, err)
	}

	var proj config.Project
	if err := yaml.Unmarshal(data, &proj); err != nil {
		return fmt.Errorf("parsing %s: %w", configFile, err)
	}

	if piiEnable {
		proj.PIIScrubbing = true
		proj.PIIReplacement = piiReplacement
	} else {
		proj.PIIScrubbing = false
		proj.PIIReplacement = ""
	}

	// Write back observeall.yaml
	updatedYAML, err := yaml.Marshal(&proj)
	if err != nil {
		return fmt.Errorf("marshaling project config: %w", err)
	}
	if err := os.WriteFile(configFile, updatedYAML, 0644); err != nil {
		return fmt.Errorf("writing %s: %w", configFile, err)
	}

	// Regenerate collector-config.yaml
	collectorCfg, err := config.GenerateCollectorConfig(proj)
	if err != nil {
		return fmt.Errorf("generating collector config: %w", err)
	}
	if err := os.WriteFile("collector-config.yaml", []byte(collectorCfg), 0644); err != nil {
		return fmt.Errorf("writing collector-config.yaml: %w", err)
	}

	if piiEnable {
		successf("PII scrubbing ENABLED (replacement: %q)", piiReplacement)
		infof("Active scrubbing patterns:")
		for _, p := range pii.DefaultPatterns {
			fmt.Printf("   • %-15s : %s\n", p.Name, p.Pattern)
		}
		successf("Regenerated collector-config.yaml with transform/pii_scrubber")
	} else {
		successf("PII scrubbing DISABLED")
		successf("Regenerated collector-config.yaml without PII filter")
	}

	return nil
}

