package cli

import (
	"fmt"
	"os"

	"github.com/observeall/observeall/internal/ebpf"
	"github.com/spf13/cobra"
)

var (
	ebpfTool       string
	ebpfPort       int
	ebpfEnv        string
	ebpfOutputFile string
)

var ebpfCmd = &cobra.Command{
	Use:   "ebpf",
	Short: "Generate zero-code eBPF network instrumentation manifests",
	Long: `Generates zero-code eBPF instrumentation manifests (e.g. Grafana Beyla)
to capture HTTP, gRPC, and SQL latency directly from the Linux kernel without
requiring language agents or source code modifications.`,
	RunE: runEBPF,
}

func init() {
	ebpfCmd.Flags().StringVar(&ebpfTool, "tool", "beyla", "eBPF tool: beyla, pixie, or coroot")
	ebpfCmd.Flags().IntVarP(&ebpfPort, "port", "p", 8080, "Target application port to monitor with eBPF")
	ebpfCmd.Flags().StringVar(&ebpfEnv, "env", "docker", "Target deployment environment: docker or k8s")
	ebpfCmd.Flags().StringVarP(&ebpfOutputFile, "output", "o", "", "Output filename (default: beyla-compose.yaml or beyla-daemonset.yaml)")
}

func runEBPF(cmd *cobra.Command, args []string) error {
	cfg := ebpf.EBPFConfig{
		Enabled:    true,
		Tool:       ebpfTool,
		TargetPort: ebpfPort,
	}

	if err := cfg.Validate(); err != nil {
		return err
	}

	var content string
	defaultOutput := "beyla-compose.yaml"

	if ebpfEnv == "k8s" {
		defaultOutput = "beyla-daemonset.yaml"
		content = ebpf.GenerateBeylaK8sManifest(cfg)
	} else {
		content = ebpf.GenerateBeylaDockerCompose(cfg)
	}

	targetFile := defaultOutput
	if ebpfOutputFile != "" {
		targetFile = ebpfOutputFile
	}

	if err := os.WriteFile(targetFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing manifest: %w", err)
	}

	fmt.Println()
	successf("Generated %s eBPF instrumentation manifest: %s", ebpfTool, targetFile)
	infof("Target monitored port: %d (Environment: %s)", ebpfPort, ebpfEnv)
	fmt.Println()
	infof("Prerequisites for running eBPF in production:")
	fmt.Println("  1. Linux kernel >= 4.14 (kernel >= 5.4 recommended for full socket tracing)")
	fmt.Println("  2. Container capabilities: CAP_SYS_ADMIN, CAP_NET_ADMIN, CAP_SYS_PTRACE")
	fmt.Println("  3. Host PID namespace shared (pid: host / hostPID: true)")
	fmt.Println("  4. Traces & metrics stream directly to ObserveAll at port 4317")
	fmt.Println()

	return nil
}

