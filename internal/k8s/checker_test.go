package k8s

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"math/big"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes/fake"
)

func TestCheckHealth_AggregatesClusterState(t *testing.T) {
	ctx := context.Background()
	ns := "default"

	replicas := int32(2)

	client := fake.NewSimpleClientset(
		// Nodes
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "node-1"},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{{
					Type:   corev1.NodeReady,
					Status: corev1.ConditionTrue,
				}},
			},
		},
		&corev1.Node{
			ObjectMeta: metav1.ObjectMeta{Name: "node-2"},
			Status: corev1.NodeStatus{
				Conditions: []corev1.NodeCondition{
					{Type: corev1.NodeReady, Status: corev1.ConditionFalse},
					{Type: corev1.NodeMemoryPressure, Status: corev1.ConditionTrue},
				},
			},
		},
		// Pods
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "web-healthy", Namespace: ns},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "web-crash", Namespace: ns},
			Status: corev1.PodStatus{
				Phase: corev1.PodRunning,
				ContainerStatuses: []corev1.ContainerStatus{{
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"},
					},
				}},
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "job-pending", Namespace: ns},
			Status: corev1.PodStatus{
				Phase: corev1.PodPending,
			},
		},
		&corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: "job-failed", Namespace: ns},
			Status: corev1.PodStatus{
				Phase: corev1.PodFailed,
			},
		},
		// Deployments
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "api", Namespace: ns},
			Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
			Status:     appsv1.DeploymentStatus{AvailableReplicas: 0},
		},
		&appsv1.Deployment{
			ObjectMeta: metav1.ObjectMeta{Name: "frontend", Namespace: ns},
			Spec:       appsv1.DeploymentSpec{Replicas: &replicas},
			Status:     appsv1.DeploymentStatus{AvailableReplicas: 2},
		},
		// PVCs
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "data-1", Namespace: ns},
			Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimPending},
		},
		&corev1.PersistentVolumeClaim{
			ObjectMeta: metav1.ObjectMeta{Name: "data-2", Namespace: ns},
			Status:     corev1.PersistentVolumeClaimStatus{Phase: corev1.ClaimBound},
		},
		// Services
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "lb-ready", Namespace: ns},
			Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeLoadBalancer},
			Status: corev1.ServiceStatus{
				LoadBalancer: corev1.LoadBalancerStatus{
					Ingress: []corev1.LoadBalancerIngress{{IP: "1.2.3.4"}},
				},
			},
		},
		&corev1.Service{
			ObjectMeta: metav1.ObjectMeta{Name: "lb-noip", Namespace: ns},
			Spec:       corev1.ServiceSpec{Type: corev1.ServiceTypeLoadBalancer},
		},
		// Events (Warning within last hour)
		&corev1.Event{
			ObjectMeta: metav1.ObjectMeta{Name: "ev1", Namespace: ns},
			InvolvedObject: corev1.ObjectReference{
				Namespace: ns,
				Name:      "web-crash",
			},
			Type:          corev1.EventTypeWarning,
			Message:       "CrashLoop detected",
			LastTimestamp: metav1.NewTime(time.Now().Add(-30 * time.Minute)),
		},
		// Ingresses and secrets for certs
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: "ing-valid", Namespace: ns},
			Spec: networkingv1.IngressSpec{
				TLS: []networkingv1.IngressTLS{{
					Hosts:      []string{"ok.example.com"},
					SecretName: "tls-valid",
				}},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "tls-valid", Namespace: ns},
			Data: map[string][]byte{
				"tls.crt": mustCertPEM(t, time.Now().Add(60*24*time.Hour)),
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: "ing-soon", Namespace: ns},
			Spec: networkingv1.IngressSpec{
				TLS: []networkingv1.IngressTLS{{
					Hosts:      []string{"soon.example.com"},
					SecretName: "tls-soon",
				}},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "tls-soon", Namespace: ns},
			Data: map[string][]byte{
				"tls.crt": mustCertPEM(t, time.Now().Add(10*24*time.Hour)),
			},
		},
		&networkingv1.Ingress{
			ObjectMeta: metav1.ObjectMeta{Name: "ing-expired", Namespace: ns},
			Spec: networkingv1.IngressSpec{
				TLS: []networkingv1.IngressTLS{{
					Hosts:      []string{"old.example.com"},
					SecretName: "tls-old",
				}},
			},
		},
		&corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{Name: "tls-old", Namespace: ns},
			Data: map[string][]byte{
				"tls.crt": mustCertPEM(t, time.Now().Add(-24*time.Hour)),
			},
		},
	)

	checker := &Checker{client: client}

	result, err := checker.CheckHealth(ctx, ns)
	if err != nil {
		t.Fatalf("CheckHealth returned error: %v", err)
	}

	if result.Nodes.Total != 2 || result.Nodes.Ready != 1 {
		t.Fatalf("unexpected node counts: %+v", result.Nodes)
	}
	assertContains(t, result.Nodes.NotReady, "node-2")
	assertContains(t, result.Nodes.Conditions, "node-2 MemoryPressure")

	if result.Pods.Total != 4 || result.Pods.Running != 1 {
		t.Fatalf("unexpected pod counts: %+v", result.Pods)
	}
	assertContains(t, result.Pods.CrashLoopBackOff, "default/web-crash")
	assertContains(t, result.Pods.Pending, "default/job-pending")
	assertContains(t, result.Pods.Failed, "default/job-failed")

	if result.Deployments.Total != 2 || result.Deployments.Available != 1 {
		t.Fatalf("unexpected deployment counts: %+v", result.Deployments)
	}
	assertContains(t, result.Deployments.Unavailable, "default/api")

	if result.PVCs.Total != 2 || result.PVCs.Bound != 1 {
		t.Fatalf("unexpected pvc counts: %+v", result.PVCs)
	}
	assertContains(t, result.PVCs.Pending, "default/data-1")

	if result.Services.Total != 2 || result.Services.Ready != 1 {
		t.Fatalf("unexpected service counts: %+v", result.Services)
	}
	assertContains(t, result.Services.NoIP, "default/lb-noip")

	if len(result.Events.Warnings) != 1 || result.Events.Warnings[0] == "" {
		t.Fatalf("unexpected events: %+v", result.Events)
	}

	if result.Certs.Total != 3 || result.Certs.Valid != 1 {
		t.Fatalf("unexpected cert counts: %+v", result.Certs)
	}
	assertContains(t, hostList(result.Certs.Expiring), "soon.example.com")
	assertContains(t, hostList(result.Certs.Expired), "old.example.com")
}

func mustCertPEM(t *testing.T, notAfter time.Time) []byte {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("failed to generate key: %v", err)
	}

	template := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "example"},
		NotBefore:    time.Now().Add(-1 * time.Hour),
		NotAfter:     notAfter,
	}

	certDER, err := x509.CreateCertificate(rand.Reader, template, template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("failed to create cert: %v", err)
	}

	return pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: certDER})
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

func hostList(certs []CertInfo) []string {
	hosts := make([]string, 0, len(certs))
	for _, c := range certs {
		hosts = append(hosts, c.Host)
	}
	return hosts
}
