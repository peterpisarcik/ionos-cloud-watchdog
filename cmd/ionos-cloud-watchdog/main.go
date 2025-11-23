package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"github.com/peterpisarcik/ionos-cloud-watchdog/internal/feed"
	"github.com/peterpisarcik/ionos-cloud-watchdog/internal/ionos"
	"github.com/peterpisarcik/ionos-cloud-watchdog/internal/k8s"
)

type Config struct {
	Kubeconfig string
	Namespace  string
	Output     string
	Verbose    bool
}

type Report struct {
	Status     string                   `json:"status"`
	StatusPage *feed.StatusResult       `json:"status_page,omitempty"`
	APICheck   *ionos.CheckResult       `json:"api_check,omitempty"`
	AuthCheck  *ionos.CheckResult       `json:"auth_check,omitempty"`
	Clusters   []ionos.K8sClusterStatus `json:"clusters,omitempty"`
	Health     *k8s.HealthResult        `json:"health,omitempty"`
	Issues     []string                 `json:"issues,omitempty"`
}

func main() {
	cfg := parseFlags()

	report := &Report{Status: "OK"}
	var issues []string

	statusResult, err := feed.CheckStatus()
	if err != nil {
		issues = append(issues, fmt.Sprintf("Status page: %v", err))
	} else {
		report.StatusPage = statusResult
		if statusResult.Status != feed.StatusOK {
			issues = append(issues, fmt.Sprintf("Status page: %s", statusResult.Status))
		}
	}

	var apiOK, authOK bool
	var clusterStatuses []ionos.K8sClusterStatus

	client, err := ionos.NewClientFromEnv()
	if err == nil {
		connResult := client.CheckConnectivity()
		report.APICheck = &connResult
		apiOK = connResult.OK
		if !apiOK {
			issues = append(issues, "IONOS API unreachable")
		}

		authResult := client.CheckAuthentication()
		report.AuthCheck = &authResult
		authOK = authResult.OK
		if !authOK {
			issues = append(issues, "IONOS authentication failed")
		}

		clusterStatuses, err = client.CheckK8sClusters()
		if err != nil {
			issues = append(issues, fmt.Sprintf("K8s clusters: %v", err))
		} else {
			report.Clusters = clusterStatuses
			for _, status := range clusterStatuses {
				for _, issue := range status.Issues {
					issues = append(issues, fmt.Sprintf("Cluster %s: %s", status.Cluster.Properties.Name, issue))
				}
			}
		}
	}

	var health *k8s.HealthResult
	checker, err := k8s.NewChecker(cfg.Kubeconfig)
	if err == nil {
		health, err = checker.CheckHealth(context.Background(), cfg.Namespace)
		if err != nil {
			issues = append(issues, fmt.Sprintf("K8s health: %v", err))
		} else {
			report.Health = health

			nodeIssues := len(health.Nodes.NotReady) + len(health.Nodes.Conditions)
			podIssues := len(health.Pods.CrashLoopBackOff) + len(health.Pods.ImagePullBackOff) + len(health.Pods.Pending) + len(health.Pods.Failed)

			if nodeIssues > 0 {
				issues = append(issues, fmt.Sprintf("%d node issues", nodeIssues))
			}
			if podIssues > 0 {
				issues = append(issues, fmt.Sprintf("%d pod issues", podIssues))
			}
			if len(health.Deployments.Unavailable) > 0 {
				issues = append(issues, fmt.Sprintf("%d deployment issues", len(health.Deployments.Unavailable)))
			}
			if len(health.PVCs.Pending) > 0 {
				issues = append(issues, fmt.Sprintf("%d PVC issues", len(health.PVCs.Pending)))
			}
			if len(health.Services.NoIP) > 0 {
				issues = append(issues, fmt.Sprintf("%d LoadBalancer issues", len(health.Services.NoIP)))
			}
		}
	}

	report.Issues = issues
	if len(issues) > 0 {
		report.Status = "WARNING"
	}
	if len(issues) > 3 {
		report.Status = "CRITICAL"
	}

	if cfg.Output == "json" {
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		enc.Encode(report)
	} else {
		fmt.Println()
		fmt.Println("IONOS Cloud")
		fmt.Println("-----------")

		if statusResult != nil {
			if statusResult.Status == feed.StatusOK {
				fmt.Println("  Status Page     OK")
			} else {
				fmt.Printf("  Status Page     %s\n", statusResult.Status)
			}
		}

		if client != nil {
			if apiOK {
				fmt.Println("  API             OK")
			} else {
				fmt.Println("  API             FAILED")
			}

			if authOK {
				fmt.Println("  Authentication  OK")
			} else {
				fmt.Println("  Authentication  FAILED")
			}
		} else {
			fmt.Println("  API             SKIPPED")
		}

		if len(clusterStatuses) > 0 {
			fmt.Println()
			fmt.Println("Kubernetes Clusters")
			fmt.Println("-------------------")

			for _, status := range clusterStatuses {
				fmt.Printf("  %s (v%s)\n", status.Cluster.Properties.Name, status.Cluster.Properties.K8sVersion)
				fmt.Printf("    Node Pools: %d\n", len(status.NodePools))
				if len(status.Issues) == 0 {
					fmt.Println("    State: ACTIVE")
				} else {
					fmt.Println("    State: ISSUES")
				}
			}
		}

		if health != nil {
			fmt.Println()
			fmt.Println("Health")
			fmt.Println("------")

			fmt.Printf("  Nodes         %d/%d Ready\n", health.Nodes.Ready, health.Nodes.Total)
			fmt.Printf("  Pods          %d/%d Running\n", health.Pods.Running, health.Pods.Total)
			fmt.Printf("  Deployments   %d/%d Available\n", health.Deployments.Available, health.Deployments.Total)

			if health.PVCs.Total > 0 {
				fmt.Printf("  PVCs          %d/%d Bound\n", health.PVCs.Bound, health.PVCs.Total)
			}

			if health.Services.Total > 0 {
				fmt.Printf("  LoadBalancers %d/%d Ready\n", health.Services.Ready, health.Services.Total)
			}
		}

		if len(issues) > 0 {
			fmt.Println()
			fmt.Println("Issues")
			fmt.Println("------")
			for _, issue := range issues {
				fmt.Printf("  - %s\n", issue)
			}

			if health != nil {
				if len(health.Nodes.NotReady) > 0 {
					fmt.Println()
					fmt.Println("  Nodes NotReady:")
					for _, node := range health.Nodes.NotReady {
						fmt.Printf("    %s\n", node)
					}
				}

				if len(health.Pods.CrashLoopBackOff) > 0 {
					fmt.Println()
					fmt.Println("  Pods CrashLoopBackOff:")
					for _, pod := range health.Pods.CrashLoopBackOff {
						fmt.Printf("    %s\n", pod)
					}
				}

				if len(health.Pods.Pending) > 0 {
					fmt.Println()
					fmt.Println("  Pods Pending:")
					for _, pod := range health.Pods.Pending {
						fmt.Printf("    %s\n", pod)
					}
				}

				if len(health.PVCs.Pending) > 0 {
					fmt.Println()
					fmt.Println("  PVCs Pending:")
					for _, pvc := range health.PVCs.Pending {
						fmt.Printf("    %s\n", pvc)
					}
				}

				if len(health.Deployments.Unavailable) > 0 {
					fmt.Println()
					fmt.Println("  Deployments Unavailable:")
					for _, deploy := range health.Deployments.Unavailable {
						fmt.Printf("    %s\n", deploy)
					}
				}

				if len(health.Services.NoIP) > 0 {
					fmt.Println()
					fmt.Println("  LoadBalancers NoIP:")
					for _, svc := range health.Services.NoIP {
						fmt.Printf("    %s\n", svc)
					}
				}
			}
		}

		fmt.Println()
		fmt.Printf("Status: %s\n", report.Status)
	}

	if report.Status == "CRITICAL" {
		os.Exit(2)
	} else if report.Status == "WARNING" {
		os.Exit(1)
	}
}

func parseFlags() *Config {
	cfg := &Config{}

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: ionos-cloud-watchdog [options]\n\n")
		fmt.Fprintf(os.Stderr, "A diagnostic tool for IONOS Cloud and Kubernetes health checks.\n\n")
		fmt.Fprintf(os.Stderr, "Options:\n")
		flag.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\nEnvironment variables:\n")
		fmt.Fprintf(os.Stderr, "  IONOS_TOKEN      IONOS Cloud API token\n")
		fmt.Fprintf(os.Stderr, "  IONOS_USERNAME   IONOS Cloud username (alternative to token)\n")
		fmt.Fprintf(os.Stderr, "  IONOS_PASSWORD   IONOS Cloud password (alternative to token)\n")
		fmt.Fprintf(os.Stderr, "\nExit codes:\n")
		fmt.Fprintf(os.Stderr, "  0  OK\n")
		fmt.Fprintf(os.Stderr, "  1  WARNING (1-3 issues)\n")
		fmt.Fprintf(os.Stderr, "  2  CRITICAL (>3 issues)\n")
	}

	flag.StringVar(&cfg.Kubeconfig, "kubeconfig", "", "path to kubeconfig file")
	flag.StringVar(&cfg.Namespace, "namespace", "", "kubernetes namespace to check (default: all)")
	flag.StringVar(&cfg.Output, "output", "text", "output format: text or json")
	flag.BoolVar(&cfg.Verbose, "verbose", false, "verbose output")
	flag.BoolVar(&cfg.Verbose, "v", false, "verbose output (shorthand)")
	flag.Parse()
	return cfg
}
