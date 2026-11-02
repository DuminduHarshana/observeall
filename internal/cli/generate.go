package cli

import (
	"fmt"
	"os"

	"github.com/observeall/observeall/internal/config"
	"github.com/spf13/cobra"
)

var generateCmd = &cobra.Command{
	Use:   "generate",
	Short: "Regenerate collector-config.yaml from observeall.yaml",
	Long:  `Re-generates the OTel Collector pipeline config from the current observeall.yaml project file.`,
	RunE:  runGenerate,
}

func runGenerate(cmd *cobra.Command, args []string) error {
	proj, err := loadProjectConfig(cmd)
	if err != nil {
		return err
	}

	collectorCfg, err := config.GenerateCollectorConfig(proj)
	if err != nil {
		return fmt.Errorf("failed to generate collector config: %w", err)
	}

	if err := os.WriteFile("collector-config.yaml", []byte(collectorCfg), 0644); err != nil {
		return fmt.Errorf("failed to write collector-config.yaml: %w", err)
	}

	successf("Regenerated collector-config.yaml")
	infof("Backends: %v", proj.Backends)
	infof("Signals : %v", proj.Signals)
	return nil
}

