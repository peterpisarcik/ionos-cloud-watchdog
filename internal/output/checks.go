package output

import (
	"context"
	"fmt"
	"sync"

	"github.com/peterpisarcik/ionos-cloud-watchdog/internal/feed"
	"github.com/peterpisarcik/ionos-cloud-watchdog/internal/ionos"
	"github.com/peterpisarcik/ionos-cloud-watchdog/internal/k8s"
)

var (
	feedCheckStatus = feed.CheckStatus
	newIONOSClient  = func() (ionosClient, error) { return ionos.NewClientFromEnv() }
	newK8sChecker   = func(kubeconfig string) (k8sChecker, error) { return k8s.NewChecker(kubeconfig) }
)

type ionosClient interface {
	CheckConnectivity() ionos.CheckResult
	CheckAuthentication() ionos.CheckResult
	CheckDatacenters() ([]ionos.DatacenterStatus, error)
	CheckK8sClusters() ([]ionos.K8sClusterStatus, error)
}

type k8sChecker interface {
	CheckHealth(ctx context.Context, namespace string) (*k8s.HealthResult, error)
}

func RunChecks(kubeconfig, namespace string) (*Report, error) {
	report := &Report{Status: "OK"}
	var issues []string

	var wg sync.WaitGroup

	wg.Add(3)

	go checkStatusPage(&wg, report, &issues)
	go checkIONOS(&wg, report, &issues)
	go checkK8s(&wg, report, &issues, kubeconfig, namespace)

	wg.Wait()

	report.Issues = issues
	if len(issues) > 0 {
		report.Status = "WARNING"
	}
	if len(issues) > 3 {
		report.Status = "CRITICAL"
	}

	return report, nil
}

func checkStatusPage(wg *sync.WaitGroup, report *Report, issues *[]string) {
	defer wg.Done()
	statusResult, err := feedCheckStatus()
	if err != nil {
		*issues = append(*issues, fmt.Sprintf("Status page: %v", err))
	} else {
		report.StatusPage = statusResult
		if statusResult.Status != feed.StatusOK {
			if len(statusResult.ActiveIncidents) > 0 {
				for _, incident := range statusResult.ActiveIncidents {
					*issues = append(*issues, fmt.Sprintf("Status page: %s", incident.Title))
				}
			} else {
				*issues = append(*issues, fmt.Sprintf("Status page: %s", statusResult.Status))
			}
		}
	}
}

func checkIONOS(wg *sync.WaitGroup, report *Report, issues *[]string) {
	defer wg.Done()

	client, err := newIONOSClient()
	if err != nil {
		return
	}

	connResult := client.CheckConnectivity()
	report.APICheck = &connResult
	if !connResult.OK {
		*issues = append(*issues, "IONOS API unreachable")
	}

	authResult := client.CheckAuthentication()
	report.AuthCheck = &authResult
	if !authResult.OK {
		*issues = append(*issues, "IONOS authentication failed")
	}

	datacenterStatuses, err := client.CheckDatacenters()
	if err != nil {
		*issues = append(*issues, fmt.Sprintf("Datacenters: %v", err))
	} else {
		report.Datacenters = datacenterStatuses
		for _, status := range datacenterStatuses {
			for _, issue := range status.Issues {
				*issues = append(*issues, fmt.Sprintf("DC %s: %s", status.Datacenter.Properties.Name, issue))
			}
		}
	}

	clusterStatuses, err := client.CheckK8sClusters()
	if err != nil {
		*issues = append(*issues, fmt.Sprintf("K8s clusters: %v", err))
	} else {
		report.Clusters = clusterStatuses
		for _, status := range clusterStatuses {
			for _, issue := range status.Issues {
				*issues = append(*issues, fmt.Sprintf("Cluster %s: %s", status.Cluster.Properties.Name, issue))
			}
		}
	}
}

func checkK8s(wg *sync.WaitGroup, report *Report, issues *[]string, kubeconfig, namespace string) {
	defer wg.Done()

	checker, err := newK8sChecker(kubeconfig)
	if err != nil {
		return
	}

	health, err := checker.CheckHealth(context.Background(), namespace)
	if err != nil {
		*issues = append(*issues, fmt.Sprintf("K8s health: %v", err))
		return
	}

	report.Health = health

	nodeIssues := len(health.Nodes.NotReady) + len(health.Nodes.Conditions)
	podIssues := len(health.Pods.CrashLoopBackOff) + len(health.Pods.ImagePullBackOff) + len(health.Pods.Pending) + len(health.Pods.Failed)

	if nodeIssues > 0 {
		*issues = append(*issues, fmt.Sprintf("%d node issues", nodeIssues))
	}
	if podIssues > 0 {
		*issues = append(*issues, fmt.Sprintf("%d pod issues", podIssues))
	}
	if len(health.Deployments.Unavailable) > 0 {
		*issues = append(*issues, fmt.Sprintf("%d deployment issues", len(health.Deployments.Unavailable)))
	}
	if len(health.PVCs.Pending) > 0 {
		*issues = append(*issues, fmt.Sprintf("%d PVC issues", len(health.PVCs.Pending)))
	}
	if len(health.Services.NoIP) > 0 {
		*issues = append(*issues, fmt.Sprintf("%d LoadBalancer issues", len(health.Services.NoIP)))
	}
	if len(health.Certs.Expired) > 0 {
		*issues = append(*issues, fmt.Sprintf("%d expired certificates", len(health.Certs.Expired)))
	}
	if len(health.Certs.Expiring) > 0 {
		*issues = append(*issues, fmt.Sprintf("%d certificates expiring soon", len(health.Certs.Expiring)))
	}
}
