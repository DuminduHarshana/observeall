package cli

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/observeall/observeall/internal/config"
	"github.com/observeall/observeall/internal/gitops"
	"github.com/spf13/cobra"
)

// ---------- watch command ----------

var watchCmd = &cobra.Command{
	Use:   "watch",
	Short: "Watch observeall.yaml for changes and regenerate collector-config.yaml",
	Long: `Continuously polls observeall.yaml every 2 seconds.
When a change is detected, collector-config.yaml is regenerated.
Pass --gitops to also commit and push the updated file automatically.`,
	RunE: runWatch,
}

var watchGitOps bool

func init() {
	watchCmd.Flags().BoolVar(&watchGitOps, "gitops", false, "Commit and push collector-config.yaml after each regeneration")
}

func runWatch(cmd *cobra.Command, _ []string) error {
	cfgFile, _ := cmd.Flags().GetString("config")

	// Validate that the config file exists before starting the watcher.
	if _, err := os.Stat(cfgFile); err != nil {
		return fmt.Errorf("config file %q not found — run `observeall init` first: %w", cfgFile, err)
	}

	infof("Watching %s for changes (Ctrl-C to stop)…", cfgFile)

	watcher := gitops.NewConfigWatcher(cfgFile, "collector-config.yaml")

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	return watcher.Watch(ctx, func(proj config.Project) {
		fmt.Println("[changed] regenerating collector-config.yaml")

		collectorCfg, err := config.GenerateCollectorConfig(proj)
		if err != nil {
			errorf("Failed to generate collector config: %v", err)
			return
		}
		if err := os.WriteFile("collector-config.yaml", []byte(collectorCfg), 0644); err != nil {
			errorf("Failed to write collector-config.yaml: %v", err)
			return
		}
		successf("Regenerated collector-config.yaml")

		if watchGitOps {
			pub := gitops.NewGitOpsPublisher("")
			msg := fmt.Sprintf("gitops: regenerate collector-config.yaml at %s", time.Now().UTC().Format(time.RFC3339))
			if err := pub.CommitAndPush(msg); err != nil {
				errorf("Git push failed: %v", err)
				return
			}
			successf("Committed and pushed collector-config.yaml")
		}
	})
}

// ---------- sync command ----------

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Regenerate collector-config.yaml and commit+push via GitOps",
	Long: `One-shot: regenerates collector-config.yaml from the current observeall.yaml,
then stages, commits, and pushes the result using the system git binary.`,
	RunE: runSync,
}

func runSync(cmd *cobra.Command, _ []string) error {
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

	pub := gitops.NewGitOpsPublisher("")
	msg := fmt.Sprintf("gitops: sync collector-config.yaml at %s", time.Now().UTC().Format(time.RFC3339))
	if err := pub.CommitAndPush(msg); err != nil {
		return fmt.Errorf("GitOps push failed: %w", err)
	}
	successf("Committed and pushed collector-config.yaml")
	return nil
}
