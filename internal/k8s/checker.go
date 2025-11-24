package k8s

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"path/filepath"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Checker struct {
	client *kubernetes.Clientset
}

type quietWarningHandler struct{}

func (quietWarningHandler) HandleWarningHeader(code int, agent string, text string) {}

type HealthResult struct {
	Nodes       NodeResult
	Pods        PodResult
	Deployments DeploymentResult
	PVCs        PVCResult
	Services    ServiceResult
	Events      EventResult
	Certs       CertResult
}

type NodeResult struct {
	Total      int
	Ready      int
	NotReady   []string
	Conditions []string
}

type PodResult struct {
	Total           int
	Running         int
	CrashLoopBackOff []string
	ImagePullBackOff []string
	Pending         []string
	Failed          []string
}

type DeploymentResult struct {
	Total      int
	Available  int
	Unavailable []string
}

type PVCResult struct {
	Total   int
	Bound   int
	Pending []string
}

type ServiceResult struct {
	Total     int
	Ready     int
	NoIP      []string
}

type EventResult struct {
	Warnings []string
}

type CertInfo struct {
	Host      string
	Namespace string
	Secret    string
	ExpiresIn int
	Expiry    time.Time
}

type CertResult struct {
	Total    int
	Valid    int
	Expiring []CertInfo
	Expired  []CertInfo
}

func NewChecker(kubeconfigPath string) (*Checker, error) {
	if kubeconfigPath == "" {
		if home := homedir.HomeDir(); home != "" {
			kubeconfigPath = filepath.Join(home, ".kube", "config")
		}
	}

	config, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load kubeconfig: %w", err)
	}

	config.Timeout = 10 * time.Second
	config.WarningHandler = quietWarningHandler{}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return &Checker{client: clientset}, nil
}

func (c *Checker) CheckHealth(ctx context.Context, namespace string) (*HealthResult, error) {
	result := &HealthResult{}

	nodeResult, err := c.checkNodes(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to check nodes: %w", err)
	}
	result.Nodes = *nodeResult

	podResult, err := c.checkPods(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to check pods: %w", err)
	}
	result.Pods = *podResult

	deployResult, err := c.checkDeployments(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to check deployments: %w", err)
	}
	result.Deployments = *deployResult

	pvcResult, err := c.checkPVCs(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to check pvcs: %w", err)
	}
	result.PVCs = *pvcResult

	svcResult, err := c.checkServices(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to check services: %w", err)
	}
	result.Services = *svcResult

	eventResult, err := c.checkEvents(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to check events: %w", err)
	}
	result.Events = *eventResult

	certResult, err := c.checkCertificates(ctx, namespace)
	if err != nil {
		return nil, fmt.Errorf("failed to check certificates: %w", err)
	}
	result.Certs = *certResult

	return result, nil
}

func (c *Checker) checkNodes(ctx context.Context) (*NodeResult, error) {
	nodes, err := c.client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := &NodeResult{
		Total: len(nodes.Items),
	}

	for _, node := range nodes.Items {
		ready := false
		for _, condition := range node.Status.Conditions {
			if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
				ready = true
			}
			if condition.Type == corev1.NodeMemoryPressure && condition.Status == corev1.ConditionTrue {
				result.Conditions = append(result.Conditions, fmt.Sprintf("%s MemoryPressure", node.Name))
			}
			if condition.Type == corev1.NodeDiskPressure && condition.Status == corev1.ConditionTrue {
				result.Conditions = append(result.Conditions, fmt.Sprintf("%s DiskPressure", node.Name))
			}
			if condition.Type == corev1.NodePIDPressure && condition.Status == corev1.ConditionTrue {
				result.Conditions = append(result.Conditions, fmt.Sprintf("%s PIDPressure", node.Name))
			}
		}
		if ready {
			result.Ready++
		} else {
			result.NotReady = append(result.NotReady, node.Name)
		}
	}

	return result, nil
}

func (c *Checker) checkPods(ctx context.Context, namespace string) (*PodResult, error) {
	pods, err := c.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := &PodResult{
		Total: len(pods.Items),
	}

	for _, pod := range pods.Items {
		podName := fmt.Sprintf("%s/%s", pod.Namespace, pod.Name)

		switch pod.Status.Phase {
		case corev1.PodRunning:
			hasIssue := false
			for _, cs := range pod.Status.ContainerStatuses {
				if cs.State.Waiting != nil {
					reason := cs.State.Waiting.Reason
					switch reason {
					case "CrashLoopBackOff":
						result.CrashLoopBackOff = append(result.CrashLoopBackOff, podName)
						hasIssue = true
					case "ImagePullBackOff", "ErrImagePull":
						result.ImagePullBackOff = append(result.ImagePullBackOff, podName)
						hasIssue = true
					}
				}
			}
			if !hasIssue {
				result.Running++
			}
		case corev1.PodPending:
			result.Pending = append(result.Pending, podName)
		case corev1.PodFailed:
			result.Failed = append(result.Failed, podName)
		default:
			result.Running++
		}
	}

	return result, nil
}

func (c *Checker) checkDeployments(ctx context.Context, namespace string) (*DeploymentResult, error) {
	deployments, err := c.client.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := &DeploymentResult{
		Total: len(deployments.Items),
	}

	for _, deploy := range deployments.Items {
		deployName := fmt.Sprintf("%s/%s", deploy.Namespace, deploy.Name)

		if deploy.Status.AvailableReplicas >= *deploy.Spec.Replicas {
			result.Available++
		} else {
			result.Unavailable = append(result.Unavailable, deployName)
		}
	}

	return result, nil
}

func (c *Checker) checkPVCs(ctx context.Context, namespace string) (*PVCResult, error) {
	pvcs, err := c.client.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := &PVCResult{
		Total: len(pvcs.Items),
	}

	for _, pvc := range pvcs.Items {
		pvcName := fmt.Sprintf("%s/%s", pvc.Namespace, pvc.Name)

		if pvc.Status.Phase == corev1.ClaimBound {
			result.Bound++
		} else if pvc.Status.Phase == corev1.ClaimPending {
			result.Pending = append(result.Pending, pvcName)
		}
	}

	return result, nil
}

func (c *Checker) checkServices(ctx context.Context, namespace string) (*ServiceResult, error) {
	services, err := c.client.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	result := &ServiceResult{}

	for _, svc := range services.Items {
		if svc.Spec.Type != corev1.ServiceTypeLoadBalancer {
			continue
		}

		result.Total++
		svcName := fmt.Sprintf("%s/%s", svc.Namespace, svc.Name)

		if len(svc.Status.LoadBalancer.Ingress) > 0 {
			result.Ready++
		} else {
			result.NoIP = append(result.NoIP, svcName)
		}
	}

	return result, nil
}

func (c *Checker) checkEvents(ctx context.Context, namespace string) (*EventResult, error) {
	events, err := c.client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "type=Warning",
	})
	if err != nil {
		return nil, err
	}

	result := &EventResult{}

	cutoff := time.Now().Add(-1 * time.Hour)

	for _, event := range events.Items {
		eventTime := event.LastTimestamp.Time
		if eventTime.IsZero() {
			eventTime = event.EventTime.Time
		}

		if eventTime.After(cutoff) {
			msg := fmt.Sprintf("%s/%s: %s", event.InvolvedObject.Namespace, event.InvolvedObject.Name, event.Message)
			result.Warnings = append(result.Warnings, msg)
		}
	}

	return result, nil
}

func (c *Checker) checkCertificates(ctx context.Context, namespace string) (*CertResult, error) {
	result := &CertResult{}

	ingresses, err := c.client.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool)

	for _, ing := range ingresses.Items {
		for _, tls := range ing.Spec.TLS {
			if tls.SecretName == "" {
				continue
			}

			key := fmt.Sprintf("%s/%s", ing.Namespace, tls.SecretName)
			if seen[key] {
				continue
			}
			seen[key] = true

			secret, err := c.client.CoreV1().Secrets(ing.Namespace).Get(ctx, tls.SecretName, metav1.GetOptions{})
			if err != nil {
				continue
			}

			certData, ok := secret.Data["tls.crt"]
			if !ok {
				continue
			}

			block, _ := pem.Decode(certData)
			if block == nil {
				continue
			}

			cert, err := x509.ParseCertificate(block.Bytes)
			if err != nil {
				continue
			}

			result.Total++
			daysUntilExpiry := int(time.Until(cert.NotAfter).Hours() / 24)

			host := ""
			if len(tls.Hosts) > 0 {
				host = tls.Hosts[0]
			}

			info := CertInfo{
				Host:      host,
				Namespace: ing.Namespace,
				Secret:    tls.SecretName,
				ExpiresIn: daysUntilExpiry,
				Expiry:    cert.NotAfter,
			}

			if daysUntilExpiry < 0 {
				result.Expired = append(result.Expired, info)
			} else if daysUntilExpiry < 30 {
				result.Expiring = append(result.Expiring, info)
			} else {
				result.Valid++
			}
		}
	}

	return result, nil
}
