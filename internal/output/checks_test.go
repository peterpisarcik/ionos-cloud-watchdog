package output

import (
	"context"
	"errors"
	"testing"

	"github.com/peterpisarcik/ionos-cloud-watchdog/internal/feed"
	"github.com/peterpisarcik/ionos-cloud-watchdog/internal/ionos"
	"github.com/peterpisarcik/ionos-cloud-watchdog/internal/k8s"
)

func TestRunChecks_OK(t *testing.T) {
	restore := stubDependencies(t, &dependencyStubs{
		feedResult: &feed.StatusResult{Status: feed.StatusOK},
		ionosClient: &fakeIONOSClient{
			connectivity: ionos.CheckResult{OK: true},
			auth:         ionos.CheckResult{OK: true},
		},
		k8sHealth: &k8s.HealthResult{
			Nodes: k8s.NodeResult{Total: 0},
		},
	})
	defer restore()

	report, err := RunChecks("", "default")
	if err != nil {
		t.Fatalf("RunChecks returned error: %v", err)
	}

	if report.Status != "OK" {
		t.Fatalf("expected status OK, got %s", report.Status)
	}
	if len(report.Issues) != 0 {
		t.Fatalf("expected no issues, got %v", report.Issues)
	}
	if report.APICheck == nil || !report.APICheck.OK {
		t.Fatalf("expected API check OK")
	}
	if report.AuthCheck == nil || !report.AuthCheck.OK {
		t.Fatalf("expected auth check OK")
	}
}

func TestRunChecks_CriticalAggregatesIssues(t *testing.T) {
	restore := stubDependencies(t, &dependencyStubs{
		feedResult: &feed.StatusResult{
			Status:          feed.StatusWarning,
			ActiveIncidents: []feed.Entry{{Title: "Incident A"}},
		},
		ionosClient: &fakeIONOSClient{
			connectivity: ionos.CheckResult{OK: true},
			auth:         ionos.CheckResult{OK: false},
			datacenters: []ionos.DatacenterStatus{{
				Datacenter: ionos.DataCenter{Properties: struct {
					Name     string "json:\"name\""
					Location string "json:\"location\""
				}{Name: "DC1"}},
				Issues: []string{"Server busy"},
			}},
			clusters: []ionos.K8sClusterStatus{{
				Cluster: ionos.K8sCluster{Properties: struct {
					Name       string "json:\"name\""
					K8sVersion string "json:\"k8sVersion\""
				}{Name: "Cluster1"}},
				Issues: []string{"Cluster degraded", "Node pool down"},
			}},
		},
		k8sHealth: &k8s.HealthResult{
			Nodes: k8s.NodeResult{
				Total:    1,
				Ready:    0,
				NotReady: []string{"node-1"},
			},
			Pods: k8s.PodResult{
				Total:            1,
				Pending:          []string{"ns/pod"},
				CrashLoopBackOff: []string{"ns/pod"},
			},
		},
	})
	defer restore()

	report, err := RunChecks("", "default")
	if err != nil {
		t.Fatalf("RunChecks returned error: %v", err)
	}

	if report.Status != "CRITICAL" {
		t.Fatalf("expected status CRITICAL, got %s", report.Status)
	}

	assertContains(t, report.Issues, "Status page: Incident A")
	assertContains(t, report.Issues, "IONOS authentication failed")
	assertContains(t, report.Issues, "DC DC1: Server busy")
	assertContains(t, report.Issues, "Cluster Cluster1: Cluster degraded")
	assertContains(t, report.Issues, "1 node issues")
	assertContains(t, report.Issues, "2 pod issues")
}

type dependencyStubs struct {
	feedResult  *feed.StatusResult
	feedErr     error
	ionosClient ionosClient
	ionosErr    error
	k8sHealth   *k8s.HealthResult
	k8sErr      error
}

func stubDependencies(t *testing.T, stubs *dependencyStubs) func() {
	t.Helper()

	origFeed := feedCheckStatus
	origIONOS := newIONOSClient
	origK8s := newK8sChecker

	feedCheckStatus = func() (*feed.StatusResult, error) {
		return stubs.feedResult, stubs.feedErr
	}
	newIONOSClient = func() (ionosClient, error) {
		return stubs.ionosClient, stubs.ionosErr
	}
	newK8sChecker = func(_ string) (k8sChecker, error) {
		if stubs.k8sHealth == nil && stubs.k8sErr == nil {
			return nil, errors.New("missing k8s stub")
		}
		return &fakeK8sChecker{health: stubs.k8sHealth, err: stubs.k8sErr}, nil
	}

	return func() {
		feedCheckStatus = origFeed
		newIONOSClient = origIONOS
		newK8sChecker = origK8s
	}
}

type fakeIONOSClient struct {
	connectivity ionos.CheckResult
	auth         ionos.CheckResult
	datacenters  []ionos.DatacenterStatus
	clusters     []ionos.K8sClusterStatus
	dbaas        ionos.DBaaSStatus
	err          error
}

func (f *fakeIONOSClient) CheckConnectivity() ionos.CheckResult {
	return f.connectivity
}

func (f *fakeIONOSClient) CheckAuthentication() ionos.CheckResult {
	return f.auth
}

func (f *fakeIONOSClient) CheckDatacenters() ([]ionos.DatacenterStatus, error) {
	return f.datacenters, f.err
}

func (f *fakeIONOSClient) CheckK8sClusters() ([]ionos.K8sClusterStatus, error) {
	return f.clusters, f.err
}

func (f *fakeIONOSClient) CheckDBaaS() ionos.DBaaSStatus {
	return f.dbaas
}

type fakeK8sChecker struct {
	health *k8s.HealthResult
	err    error
}

func (f *fakeK8sChecker) CheckHealth(ctx context.Context, namespace string) (*k8s.HealthResult, error) {
	return f.health, f.err
}

func assertContains(t *testing.T, list []string, expected string) {
	t.Helper()
	for _, item := range list {
		if item == expected {
			return
		}
	}
	t.Fatalf("expected %q in %v", expected, list)
}
