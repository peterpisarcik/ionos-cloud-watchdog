package output

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/peterpisarcik/ionos-cloud-watchdog/internal/feed"
	"github.com/peterpisarcik/ionos-cloud-watchdog/internal/ionos"
	"github.com/peterpisarcik/ionos-cloud-watchdog/internal/k8s"
)

func TestPrintText_Basic(t *testing.T) {
	report := &Report{
		Status: "WARNING",
		StatusPage: &feed.StatusResult{
			Status:          feed.StatusWarning,
			ActiveIncidents: []feed.Entry{{Title: "Incident A"}},
		},
		APICheck:  &ionos.CheckResult{OK: false},
		AuthCheck: &ionos.CheckResult{OK: true},
		Issues:    []string{"Issue one"},
	}

	out := captureOutput(t, func() {
		PrintText(report, &Config{Verbose: false})
	})

	expectContains(t, out, "IONOS Cloud")
	expectContains(t, out, "Status Page")
	expectContains(t, out, "Incident A")
	expectContains(t, out, "API")
	expectContains(t, out, "- Issue one")
	expectContains(t, out, "Status: WARNING")
}

func TestPrintText_VerboseSections(t *testing.T) {
	report := &Report{
		Status: "CRITICAL",
		StatusPage: &feed.StatusResult{
			Status: feed.StatusOK,
		},
		APICheck:  &ionos.CheckResult{OK: true},
		AuthCheck: &ionos.CheckResult{OK: true},
		Datacenters: []ionos.DatacenterStatus{{
			Datacenter: ionos.DataCenter{
				Properties: struct {
					Name     string "json:\"name\""
					Location string "json:\"location\""
				}{Name: "DC1", Location: "loc"},
			},
			Servers: []ionos.Server{{
				Properties: struct {
					Name    string "json:\"name\""
					Cores   int    "json:\"cores\""
					Ram     int    "json:\"ram\""
					VMState string "json:\"vmState\""
				}{Name: "srv1", VMState: "AVAILABLE"},
				Metadata: struct {
					State string "json:\"state\""
				}{State: "AVAILABLE"},
			}},
			Volumes: []ionos.Volume{{
				Properties: struct {
					Name string  "json:\"name\""
					Size float64 "json:\"size\""
					Type string  "json:\"type\""
				}{Name: "vol1", Size: 10, Type: "HDD"},
				Metadata: struct {
					State string "json:\"state\""
				}{State: "AVAILABLE"},
			}},
			Issues: []string{"Server issue"},
		}},
		Clusters: []ionos.K8sClusterStatus{{
			Cluster: ionos.K8sCluster{
				Properties: struct {
					Name       string "json:\"name\""
					K8sVersion string "json:\"k8sVersion\""
				}{Name: "cluster", K8sVersion: "1.2.3"},
			},
			NodePools: []ionos.K8sNodePool{{
				Properties: struct {
					Name             string "json:\"name\""
					NodeCount        int    "json:\"nodeCount\""
					K8sVersion       string "json:\"k8sVersion\""
					AvailabilityZone string "json:\"availabilityZone\""
				}{Name: "pool1", NodeCount: 3},
				Metadata: struct {
					State string "json:\"state\""
				}{State: "ACTIVE"},
			}},
			Issues: []string{"Cluster issue"},
		}},
		Health: &k8s.HealthResult{
			Nodes: k8s.NodeResult{
				Ready:    1,
				Total:    2,
				NotReady: []string{"node-2"},
			},
			Pods: k8s.PodResult{
				Running:          1,
				Total:            2,
				CrashLoopBackOff: []string{"ns/pod1"},
				Pending:          []string{"ns/pod2"},
			},
			Deployments: k8s.DeploymentResult{
				Available:   1,
				Total:       1,
				Unavailable: []string{"ns/deploy"},
			},
			PVCs: k8s.PVCResult{
				Bound:   1,
				Total:   2,
				Pending: []string{"ns/pvc"},
			},
			Services: k8s.ServiceResult{
				Ready: 0,
				Total: 1,
				NoIP:  []string{"ns/svc"},
			},
			Certs: k8s.CertResult{
				Total:   1,
				Valid:   0,
				Expired: []k8s.CertInfo{{Host: "old.example.com", Secret: "s1"}},
			},
		},
		Issues: []string{"Server issue", "Cluster issue", "2 node issues", "2 pod issues"},
	}

	out := captureOutput(t, func() {
		PrintText(report, &Config{Verbose: true})
	})

	expectContains(t, out, "Datacenters")
	expectContains(t, out, "srv1 (AVAILABLE)")
	expectContains(t, out, "vol1 (10GB HDD)")
	expectContains(t, out, "Kubernetes Clusters")
	expectContains(t, out, "pool1 (3 nodes")
	expectContains(t, out, "Nodes NotReady:")
	expectContains(t, out, "Pods CrashLoopBackOff:")
	expectContains(t, out, "Pods Pending:")
	expectContains(t, out, "Deployments Unavailable:")
	expectContains(t, out, "LoadBalancers NoIP:")
	expectContains(t, out, "Certificates Expired:")
	expectContains(t, out, "Status: CRITICAL")
}

func captureOutput(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("failed to create pipe: %v", err)
	}
	os.Stdout = w

	fn()

	if err := w.Close(); err != nil {
		t.Fatalf("failed to close writer: %v", err)
	}
	var buf bytes.Buffer
	if _, err := io.Copy(&buf, r); err != nil {
		t.Fatalf("failed to copy output: %v", err)
	}
	os.Stdout = orig
	return buf.String()
}

func expectContains(t *testing.T, output, substring string) {
	t.Helper()
	if !strings.Contains(output, substring) {
		t.Fatalf("expected output to contain %q\nGot:\n%s", substring, output)
	}
}
