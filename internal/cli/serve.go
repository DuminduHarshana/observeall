package cli

import (
	"fmt"

	"github.com/observeall/observeall/internal/webui"
	"github.com/spf13/cobra"
)

var (
	servePort int
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start the ObserveAll visual pipeline web dashboard",
	Long: `Starts a local web control plane on the specified port.
Allows visual drag-and-drop routing of telemetry signals, live collector health monitoring,
and real-time config updates.`,
	RunE: runServe,
}

func init() {
	serveCmd.Flags().IntVarP(&servePort, "port", "p", 8080, "Port to serve web UI on")
}

func runServe(cmd *cobra.Command, args []string) error {
	configFile, _ := cmd.Flags().GetString("config")
	collectorConfigPath := "collector-config.yaml"
	collectorMetricsURL := "http://localhost:8888/metrics"

	server := webui.NewServer(configFile, collectorConfigPath, collectorMetricsURL)
	addr := fmt.Sprintf(":%d", servePort)

	fmt.Println()
	successf("ObserveAll Web Control Plane started at http://localhost:%d", servePort)
	infof("Live pipeline editor, signal routing, and collector stats available in browser")
	infof("Press Ctrl+C to stop")
	fmt.Println()

	return server.Start(addr)
}

