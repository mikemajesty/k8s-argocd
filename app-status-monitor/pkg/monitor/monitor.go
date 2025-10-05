// pkg/monitor/monitor.go
package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"meu-monitor/pkg/config"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ApplicationMonitor struct {
	config    *config.AppConfig
	clientset *kubernetes.Clientset
}

type WebhookMessage struct {
	Application string    `json:"application"`
	Namespace   string    `json:"namespace"`
	Status      string    `json:"status"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	Critical    bool      `json:"critical"`
}

func NewApplicationMonitor(config *config.AppConfig) (*ApplicationMonitor, error) {
	// Create in-cluster config
	k8sConfig, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %v", err)
	}

	clientset, err := kubernetes.NewForConfig(k8sConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %v", err)
	}

	return &ApplicationMonitor{
		config:    config,
		clientset: clientset,
	}, nil
}

func (m *ApplicationMonitor) Start(ctx context.Context) error {
	log.Printf("🔄 Starting to monitor %d applications every %v...",
		len(m.config.Applications), m.config.WatchInterval)

	// Do initial check immediately
	m.checkAllApplications()

	ticker := time.NewTicker(m.config.WatchInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("🛑 Stopping monitor...")
			return nil
		case <-ticker.C:
			m.checkAllApplications()
		}
	}
}

func (m *ApplicationMonitor) checkAllApplications() {
	log.Printf("🔍 Starting application checks at %v", time.Now().Format(time.RFC3339))

	for _, app := range m.config.Applications {
		log.Printf("📋 Checking application: %s in namespace: %s", app.Name, app.Namespace)
		m.checkApplication(app)
	}

	log.Printf("✅ Completed application checks at %v", time.Now().Format(time.RFC3339))
}

func (m *ApplicationMonitor) checkApplication(app config.Application) {
	deployment, err := m.clientset.AppsV1().Deployments(app.Namespace).Get(
		context.TODO(), app.Name, metav1.GetOptions{})
	if err != nil {
		message := fmt.Sprintf("❌ Failed to get deployment %s in namespace %s: %v",
			app.Name, app.Namespace, err)
		log.Println(message)

		// ✅ CORREÇÃO: SEMPRE enviar webhook quando deployment não existe
		// (consideramos que um app que não existe é CRÍTICO)
		m.sendWebhook(app, "ERROR", message, true)
		return
	}

	status := m.analyzeDeployment(deployment, app)
	if status.ShouldSendWebhook(app.CriticalOnly) {
		m.sendWebhook(app, status.Level, status.Message, status.Critical)
	}
}

type DeploymentStatus struct {
	Level    string
	Message  string
	Critical bool
}

func (ds DeploymentStatus) ShouldSendWebhook(criticalOnly bool) bool {
	if criticalOnly {
		return ds.Critical
	}
	return true
}

func (m *ApplicationMonitor) analyzeDeployment(deployment *v1.Deployment, app config.Application) DeploymentStatus {
	// Check if deployment is available
	if deployment.Status.AvailableReplicas < *deployment.Spec.Replicas {
		message := fmt.Sprintf("⚠️ Deployment %s has %d/%d replicas available",
			app.Name, deployment.Status.AvailableReplicas, *deployment.Spec.Replicas)
		return DeploymentStatus{
			Level:    "WARNING",
			Message:  message,
			Critical: true,
		}
	}

	// Check deployment conditions
	for _, condition := range deployment.Status.Conditions {
		if condition.Type == v1.DeploymentAvailable && condition.Status == corev1.ConditionFalse {
			message := fmt.Sprintf("❌ Deployment %s is not available: %s",
				app.Name, condition.Message)
			return DeploymentStatus{
				Level:    "ERROR",
				Message:  message,
				Critical: true,
			}
		}
	}

	// Check pod status
	pods, err := m.clientset.CoreV1().Pods(app.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(deployment.Spec.Selector),
	})
	if err != nil {
		message := fmt.Sprintf("⚠️ Failed to get pods for deployment %s: %v", app.Name, err)
		return DeploymentStatus{
			Level:    "WARNING",
			Message:  message,
			Critical: false,
		}
	}

	// Analyze pod statuses
	runningPods := 0
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			runningPods++
		} else if pod.Status.Phase == corev1.PodPending {
			// Check if pod is stuck in pending
			if time.Since(pod.CreationTimestamp.Time) > 5*time.Minute {
				message := fmt.Sprintf("❌ Pod %s is stuck in Pending state", pod.Name)
				return DeploymentStatus{
					Level:    "ERROR",
					Message:  message,
					Critical: true,
				}
			}
		} else if pod.Status.Phase == corev1.PodFailed {
			message := fmt.Sprintf("❌ Pod %s has failed", pod.Name)
			return DeploymentStatus{
				Level:    "ERROR",
				Message:  message,
				Critical: true,
			}
		}
	}

	if runningPods < len(pods.Items) {
		message := fmt.Sprintf("⚠️ Only %d/%d pods are running for deployment %s",
			runningPods, len(pods.Items), app.Name)
		return DeploymentStatus{
			Level:    "WARNING",
			Message:  message,
			Critical: runningPods == 0, // Critical if no pods are running
		}
	}

	message := fmt.Sprintf("✅ Deployment %s is healthy - %d/%d replicas available",
		app.Name, deployment.Status.AvailableReplicas, *deployment.Spec.Replicas)
	return DeploymentStatus{
		Level:    "HEALTHY",
		Message:  message,
		Critical: false,
	}
}

func (m *ApplicationMonitor) sendWebhook(app config.Application, status, message string, critical bool) {
	if app.WebhookURL == "" {
		return
	}

	webhookMsg := WebhookMessage{
		Application: app.Name,
		Namespace:   app.Namespace,
		Status:      status,
		Message:     message,
		Timestamp:   time.Now(),
		Critical:    critical,
	}

	payload, err := json.Marshal(webhookMsg)
	if err != nil {
		log.Printf("❌ Failed to marshal webhook message: %v", err)
		return
	}

	resp, err := http.Post(app.WebhookURL, "application/json", strings.NewReader(string(payload)))
	if err != nil {
		log.Printf("❌ Failed to send webhook for %s: %v", app.Name, err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("⚠️ Webhook returned error status for %s: %d", app.Name, resp.StatusCode)
	} else {
		log.Printf("📤 Webhook sent successfully for %s", app.Name)
	}
}
