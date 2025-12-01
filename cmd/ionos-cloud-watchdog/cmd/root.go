package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/peterpisarcik/ionos-cloud-watchdog/internal/config"
	"github.com/peterpisarcik/ionos-cloud-watchdog/internal/output"
	"github.com/spf13/cobra"
)

var (
	kubeconfig string
	namespace  string
	outputFmt  string
	verbose    bool
	watch      int
)

var rootCmd = &cobra.Command{
	Use:   "ionos-cloud-watchdog",
	Short: "A diagnostic tool for IONOS Cloud and Kubernetes health checks",
	Long: `ionos-cloud-watchdog performs health checks on IONOS Cloud infrastructure
and Kubernetes clusters, reporting issues with exit codes:
  0 - OK
  1 - WARNING (1-3 issues)
  2 - CRITICAL (>3 issues)

Configuration:
  Config file: ~/.ionos-cloud-watchdog/config.yaml
  Priority: config file < environment variables < command-line flags

Environment variables:
  IONOS_TOKEN      IONOS Cloud API token
  IONOS_USERNAME   IONOS Cloud username (alternative to token)
  IONOS_PASSWORD   IONOS Cloud password (alternative to token)`,
	RunE: runChecks,
}

func Execute() error {
	return rootCmd.Execute()
}

func init() {
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "path to kubeconfig file")
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "kubernetes namespace to check (default: all)")
	rootCmd.PersistentFlags().StringVarP(&outputFmt, "output", "o", "text", "output format: text or json")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")
	rootCmd.PersistentFlags().IntVarP(&watch, "watch", "w", 0, "watch mode: refresh interval in seconds (0 = disabled)")
}

func runChecks(cmd *cobra.Command, args []string) error {
	fileCfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("error loading config: %w", err)
	}

	fileCfg.ApplyEnvironment()

	if fileCfg.IONOS.Token != "" && os.Getenv("IONOS_TOKEN") == "" {
		_ = os.Setenv("IONOS_TOKEN", fileCfg.IONOS.Token)
	}
	if fileCfg.IONOS.Username != "" && os.Getenv("IONOS_USERNAME") == "" {
		_ = os.Setenv("IONOS_USERNAME", fileCfg.IONOS.Username)
	}
	if fileCfg.IONOS.Password != "" && os.Getenv("IONOS_PASSWORD") == "" {
		_ = os.Setenv("IONOS_PASSWORD", fileCfg.IONOS.Password)
	}
	if fileCfg.IONOS.APIURL != "" && os.Getenv("IONOS_API_URL") == "" {
		_ = os.Setenv("IONOS_API_URL", fileCfg.IONOS.APIURL)
	}

	if kubeconfig == "" && fileCfg.Kubeconfig != "" {
		kubeconfig = fileCfg.Kubeconfig
	}

	if watch > 0 {
		runWatchMode()
	} else {
		runCheckOnce(false)
	}

	return nil
}

func runWatchMode() {
	first := true
	for {
		if outputFmt == "text" {
			if !first {
				fmt.Print("\033[H\033[2J")
			}
			if first {
				fmt.Println("Starting watch mode...")
				fmt.Println()
			} else {
				fmt.Printf("Last check: %s\n\n", time.Now().Format("2006-01-02 15:04:05"))
			}
		}
		runCheckOnce(true)
		first = false
		time.Sleep(time.Duration(watch) * time.Second)
	}
}

func runCheckOnce(watchMode bool) {
	report, err := output.RunChecks(kubeconfig, namespace)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		if !watchMode {
			os.Exit(1)
		}
		return
	}

	if outputFmt == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		_ = enc.Encode(report)
	} else {
		outputCfg := &output.Config{
			Verbose: verbose,
		}
		output.PrintText(report, outputCfg)
	}

	if !watchMode {
		switch report.Status {
		case "CRITICAL":
			os.Exit(2)
		case "WARNING":
			os.Exit(1)
		}
	}
}
