package output

import (
	"fmt"

	"github.com/peterpisarcik/ionos-cloud-watchdog/internal/feed"
	"github.com/peterpisarcik/ionos-cloud-watchdog/internal/ionos"
	"github.com/peterpisarcik/ionos-cloud-watchdog/internal/k8s"
)

type Report struct {
	Status      string
	StatusPage  *feed.StatusResult
	APICheck    *ionos.CheckResult
	AuthCheck   *ionos.CheckResult
	Datacenters []ionos.DatacenterStatus
	Clusters    []ionos.K8sClusterStatus
	DBaaS       *ionos.DBaaSStatus
	Health      *k8s.HealthResult
	Issues      []string
}

type Config struct {
	Verbose bool
}

func PrintText(report *Report, cfg *Config) {
	fmt.Println()
	printIONOSCloud(report)
	printDatacenters(report, cfg)
	printClusters(report, cfg)
	printDBaaS(report, cfg)
	printHealth(report)
	printIssues(report)
	fmt.Println()
	fmt.Printf("Status: %s\n", report.Status)
}

func printIONOSCloud(report *Report) {
	fmt.Println("IONOS Cloud")
	fmt.Println("-----------")

	if report.StatusPage != nil {
		if report.StatusPage.Status == feed.StatusOK {
			fmt.Printf("  %-14s %s\n", "Status Page", "OK")
		} else {
			fmt.Printf("  %-14s %s\n", "Status Page", report.StatusPage.Status)
			if len(report.StatusPage.ActiveIncidents) > 0 {
				for _, incident := range report.StatusPage.ActiveIncidents {
					fmt.Printf("    - %s\n", incident.Title)
				}
			}
		}
	}

	if report.APICheck != nil {
		if report.APICheck.OK {
			fmt.Printf("  %-14s %s\n", "API", "OK")
		} else {
			fmt.Printf("  %-14s %s\n", "API", "FAILED")
		}
	} else {
		fmt.Printf("  %-14s %s\n", "API", "SKIPPED")
	}

	if report.AuthCheck != nil {
		if report.AuthCheck.OK {
			fmt.Printf("  %-14s %s\n", "Authentication", "OK")
		} else {
			fmt.Printf("  %-14s %s\n", "Authentication", "FAILED")
		}
	}
}

func printDatacenters(report *Report, cfg *Config) {
	if len(report.Datacenters) == 0 {
		return
	}

	fmt.Println()
	fmt.Println("Datacenters")
	fmt.Println("-----------")

	for _, status := range report.Datacenters {
		fmt.Printf("  %s (%s)\n", status.Datacenter.Properties.Name, status.Datacenter.Properties.Location)
		fmt.Printf("    Servers: %d\n", len(status.Servers))
		if cfg.Verbose {
			for _, srv := range status.Servers {
				state := srv.Properties.VMState
				if state == "" {
					state = srv.Metadata.State
				}
				fmt.Printf("      - %s (%s)\n", srv.Properties.Name, state)
			}
		}
		fmt.Printf("    Volumes: %d\n", len(status.Volumes))
		if cfg.Verbose {
			for _, vol := range status.Volumes {
				fmt.Printf("      - %s (%.0fGB %s)\n", vol.Properties.Name, vol.Properties.Size, vol.Properties.Type)
			}
		}
		if len(status.Issues) == 0 {
			fmt.Println("    State: OK")
		} else {
			fmt.Println("    State: ISSUES")
		}
	}
}

func printClusters(report *Report, cfg *Config) {
	if len(report.Clusters) == 0 {
		return
	}

	fmt.Println()
	fmt.Println("Kubernetes Clusters")
	fmt.Println("-------------------")

	for _, status := range report.Clusters {
		fmt.Printf("  %s (v%s)\n", status.Cluster.Properties.Name, status.Cluster.Properties.K8sVersion)
		fmt.Printf("    Node Pools: %d\n", len(status.NodePools))
		if cfg.Verbose {
			for _, np := range status.NodePools {
				fmt.Printf("      - %s (%d nodes, %s)\n", np.Properties.Name, np.Properties.NodeCount, np.Metadata.State)
			}
		}
		if len(status.Issues) == 0 {
			fmt.Println("    State: ACTIVE")
		} else {
			fmt.Println("    State: ISSUES")
		}
	}
}

func printPostgreSQL(dbaas *ionos.DBaaSStatus, cfg *Config) {
	if len(dbaas.PostgreSQL) > 0 {
		fmt.Printf("  PostgreSQL: %d cluster(s)\n", len(dbaas.PostgreSQL))
		if cfg.Verbose {
			for _, cluster := range dbaas.PostgreSQL {
				state := cluster.Metadata.State
				fmt.Printf("    - %s (v%s, %s, %d instances, %s)\n",
					cluster.Properties.DisplayName,
					cluster.Properties.PostgresVersion,
					cluster.Properties.Location,
					cluster.Properties.Instances,
					state)
			}
		}
	}
}

func printMongoDB(dbaas *ionos.DBaaSStatus, cfg *Config) {
	if len(dbaas.MongoDB) > 0 {
		fmt.Printf("  MongoDB: %d cluster(s)\n", len(dbaas.MongoDB))
		if cfg.Verbose {
			for _, cluster := range dbaas.MongoDB {
				state := cluster.Metadata.State
				fmt.Printf("    - %s (v%s, %s, %d instances, %s)\n",
					cluster.Properties.DisplayName,
					cluster.Properties.MongoDBVersion,
					cluster.Properties.Location,
					cluster.Properties.Instances,
					state)
			}
		}
	}
}

func printMariaDB(dbaas *ionos.DBaaSStatus, cfg *Config) {
	if len(dbaas.MariaDB) > 0 {
		fmt.Printf("  MariaDB: %d cluster(s)\n", len(dbaas.MariaDB))
		if cfg.Verbose {
			for _, cluster := range dbaas.MariaDB {
				state := cluster.Metadata.State
				fmt.Printf("    - %s (v%s, %s, %d instances, %s)\n",
					cluster.Properties.DisplayName,
					cluster.Properties.MariaDBVersion,
					cluster.Properties.Location,
					cluster.Properties.Instances,
					state)
			}
		}
	}
}

func printInMemoryDB(dbaas *ionos.DBaaSStatus, cfg *Config) {
	if len(dbaas.InMemoryDB) > 0 {
		fmt.Printf("  In-Memory DB: %d instance(s)\n", len(dbaas.InMemoryDB))
		if cfg.Verbose {
			for _, instance := range dbaas.InMemoryDB {
				state := instance.Metadata.State
				fmt.Printf("    - %s (v%s, %s, %d replicas, %s)\n",
					instance.Properties.DisplayName,
					instance.Properties.Version,
					instance.Properties.Location,
					instance.Properties.Replicas,
					state)
			}
		}
	}
}

func printDBaaS(report *Report, cfg *Config) {
	if report.DBaaS == nil {
		return
	}

	dbaas := report.DBaaS
	totalClusters := len(dbaas.PostgreSQL) + len(dbaas.MongoDB) + len(dbaas.MariaDB) + len(dbaas.InMemoryDB)

	if totalClusters == 0 {
		return
	}

	fmt.Println()
	fmt.Println("Managed Databases")
	fmt.Println("-----------------")

	printPostgreSQL(dbaas, cfg)
	printMongoDB(dbaas, cfg)
	printMariaDB(dbaas, cfg)
	printInMemoryDB(dbaas, cfg)

	issueCount := 0
	for _, cluster := range dbaas.PostgreSQL {
		if cluster.Metadata.State != "AVAILABLE" && cluster.Metadata.State != "ACTIVE" {
			issueCount++
		}
	}
	for _, cluster := range dbaas.MongoDB {
		if cluster.Metadata.State != "AVAILABLE" && cluster.Metadata.State != "ACTIVE" {
			issueCount++
		}
	}
	for _, cluster := range dbaas.MariaDB {
		if cluster.Metadata.State != "AVAILABLE" && cluster.Metadata.State != "ACTIVE" {
			issueCount++
		}
	}
	for _, instance := range dbaas.InMemoryDB {
		if instance.Metadata.State != "AVAILABLE" && instance.Metadata.State != "ACTIVE" {
			issueCount++
		}
	}

	if issueCount == 0 {
		fmt.Println("  State: OK")
	} else {
		fmt.Println("  State: ISSUES")
	}
}

func printHealth(report *Report) {
	if report.Health == nil {
		return
	}

	health := report.Health

	fmt.Println()
	fmt.Println("Health")
	fmt.Println("------")

	fmt.Printf("  %-14s %d/%d Ready\n", "Nodes", health.Nodes.Ready, health.Nodes.Total)
	fmt.Printf("  %-14s %d/%d Running\n", "Pods", health.Pods.Running, health.Pods.Total)
	fmt.Printf("  %-14s %d/%d Available\n", "Deployments", health.Deployments.Available, health.Deployments.Total)

	if health.PVCs.Total > 0 {
		fmt.Printf("  %-14s %d/%d Bound\n", "PVCs", health.PVCs.Bound, health.PVCs.Total)
	}

	if health.Services.Total > 0 {
		fmt.Printf("  %-14s %d/%d Ready\n", "LoadBalancers", health.Services.Ready, health.Services.Total)
	}

	if health.Certs.Total > 0 {
		fmt.Printf("  %-14s %d/%d Valid\n", "Certificates", health.Certs.Valid, health.Certs.Total)
	}
}

func printIssues(report *Report) {
	if len(report.Issues) == 0 {
		return
	}

	health := report.Health

	fmt.Println()
	fmt.Println("Issues")
	fmt.Println("------")
	for _, issue := range report.Issues {
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

		if len(health.Certs.Expired) > 0 {
			fmt.Println()
			fmt.Println("  Certificates Expired:")
			for _, cert := range health.Certs.Expired {
				fmt.Printf("    %s (%s)\n", cert.Host, cert.Secret)
			}
		}

		if len(health.Certs.Expiring) > 0 {
			fmt.Println()
			fmt.Println("  Certificates Expiring:")
			for _, cert := range health.Certs.Expiring {
				fmt.Printf("    %s (%d days)\n", cert.Host, cert.ExpiresIn)
			}
		}
	}
}
