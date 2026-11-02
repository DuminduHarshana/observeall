package cli

import (
	"fmt"
	"net/http"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show ObserveAll collector pipeline status",
	RunE:  runStatus,
}

var (
	statusHost string
	statusPort int
)

func init() {
	statusCmd.Flags().StringVar(&statusHost, "host", "localhost", "Collector host")
	statusCmd.Flags().IntVar(&statusPort, "port", 8888, "Collector metrics port")
}

func runStatus(cmd *cobra.Command, args []string) error {
	proj, err := loadProjectConfig(cmd)
	if err != nil {
		return err
	}

	fmt.Println()
	color.Cyan("ObserveAll Pipeline Status")
	fmt.Println("─────────────────────────────")
	fmt.Printf("  Project   : %s\n", proj.Name)
	fmt.Printf("  Env       : %s\n", proj.Environment)
	fmt.Printf("  Language  : %s\n", proj.Language)
	fmt.Printf("  Backends  : %v\n", proj.Backends)
	fmt.Printf("  Signals   : %v\n", proj.Signals)
	fmt.Printf("  Sampling  : %s\n", proj.Sampling)
	fmt.Println("─────────────────────────────")

	// Check collector health
	endpoint := fmt.Sprintf("http://%s:%d/metrics", statusHost, statusPort)
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		color.Red("  Collector : ✖ unreachable (%s)", endpoint)
	} else {
		resp.Body.Close()
		if resp.StatusCode == 200 {
			color.Green("  Collector : ✔ healthy (%s)", endpoint)
		} else {
			color.Yellow("  Collector : ⚠ responded with %d", resp.StatusCode)
		}
	}

	fmt.Println()
	infof("To view full metrics: curl http://%s:%d/metrics", statusHost, statusPort)
	fmt.Println()

	return nil
}

