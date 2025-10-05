package monitor

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"sync"
	"time"

	"meu-monitor/pkg/config"

	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type ApplicationMonitor struct {
	config          *config.AppConfig
	clientset       *kubernetes.Clientset
	lastStatus      map[string]string // app+namespace -> último status
	lastCheck       map[string]time.Time
	deploymentCache map[string]*v1.Deployment
	mutex           sync.RWMutex
}

type WebhookMessage struct {
	Application string    `json:"application"`
	Namespace   string    `json:"namespace"`
	Status      string    `json:"status"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	Critical    bool      `json:"critical"`
}

// ✅ HTTP Client otimizado
var httpClient = &http.Client{
	Timeout: 5 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        50,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     30 * time.Second,
		ForceAttemptHTTP2:   true,
	},
}

// ✅ Client K8s otimizado
func getK8sClient() (*kubernetes.Clientset, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to create in-cluster config: %v", err)
	}

	// Timeouts otimizados
	config.Timeout = 15 * time.Second
	config.QPS = 50
	config.Burst = 100

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes clientset: %v", err)
	}
	return clientset, nil
}

func NewApplicationMonitor(config *config.AppConfig) (*ApplicationMonitor, error) {
	clientset, err := getK8sClient()
	if err != nil {
		return nil, err
	}

	return &ApplicationMonitor{
		config:          config,
		clientset:       clientset,
		lastStatus:      make(map[string]string),
		lastCheck:       make(map[string]time.Time),
		deploymentCache: make(map[string]*v1.Deployment),
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
	startTime := time.Now()
	log.Printf("🔍 Starting application checks at %v", startTime.Format(time.RFC3339))

	// ✅ Worker pool simples - máximo 5 verificações simultâneas
	maxWorkers := 5
	if len(m.config.Applications) < maxWorkers {
		maxWorkers = len(m.config.Applications)
	}

	jobs := make(chan config.Application, len(m.config.Applications))
	results := make(chan bool, len(m.config.Applications))

	// Inicia workers
	for w := 1; w <= maxWorkers; w++ {
		go m.worker(w, jobs, results)
	}

	// Envia jobs
	for _, app := range m.config.Applications {
		jobs <- app
	}
	close(jobs)

	// Aguarda resultados
	for i := 0; i < len(m.config.Applications); i++ {
		<-results
	}

	duration := time.Since(startTime)
	log.Printf("✅ Completed %d application checks in %v",
		len(m.config.Applications), duration)
}

func (m *ApplicationMonitor) worker(id int, jobs <-chan config.Application, results chan<- bool) {
	for app := range jobs {
		m.checkApplication(app)
		results <- true
	}
}

func (m *ApplicationMonitor) checkApplication(app config.Application) {
	key := app.Name + "/" + app.Namespace

	// ✅ Verifica cache (30 segundos)
	m.mutex.RLock()
	cachedDeployment, hasCache := m.deploymentCache[key]
	lastCheck, checked := m.lastCheck[key]
	m.mutex.RUnlock()

	if hasCache && checked && time.Since(lastCheck) < 30*time.Second {
		// ✅ Usa cache
		status := m.analyzeDeployment(cachedDeployment, app)
		m.maybeSendWebhook(app, status.Level, status.Message, status.Critical)
		return
	}

	// ✅ Busca fresh data
	deployment, err := m.clientset.AppsV1().Deployments(app.Namespace).Get(
		context.TODO(), app.Name, metav1.GetOptions{})
	if err != nil {
		message := fmt.Sprintf("❌ Failed to get deployment %s: %v", app.Name, err)
		m.maybeSendWebhook(app, "ERROR", message, true)
		return
	}

	// ✅ Atualiza cache
	m.mutex.Lock()
	m.deploymentCache[key] = deployment
	m.lastCheck[key] = time.Now()
	m.mutex.Unlock()

	status := m.analyzeDeployment(deployment, app)
	m.maybeSendWebhook(app, status.Level, status.Message, status.Critical)
}

// ✅ Só envia webhook se o status mudou
func (m *ApplicationMonitor) maybeSendWebhook(app config.Application, level, message string, critical bool) {
	key := app.Name + "/" + app.Namespace
	currentStatus := level + ":" + message

	// ✅ Lógica inteligente:
	// - SEMPRE envia se for CRÍTICO
	// - Envia apenas se mudou para status não-crítico
	if critical || m.lastStatus[key] != currentStatus {
		m.sendWebhook(app, level, message, critical)
		m.lastStatus[key] = currentStatus

		if critical {
			log.Printf("🚨 CRITICAL status for %s: %s", key, level)
		} else {
			log.Printf("📤 Status CHANGED for %s: %s", key, level)
		}
	} else {
		log.Printf("🔁 Status UNCHANGED for %s: %s", key, level)
	}
}

type DeploymentStatus struct {
	Level    string
	Message  string
	Critical bool
}

func (m *ApplicationMonitor) analyzeDeployment(deployment *v1.Deployment, app config.Application) DeploymentStatus {
	// ✅ Check rápido primeiro - replicas disponíveis
	if deployment.Status.AvailableReplicas < *deployment.Spec.Replicas {
		message := fmt.Sprintf("⚠️ Deployment %s has %d/%d replicas available",
			app.Name, deployment.Status.AvailableReplicas, *deployment.Spec.Replicas)
		return DeploymentStatus{
			Level:    "WARNING",
			Message:  message,
			Critical: true,
		}
	}

	// ✅ Check condições rapidamente
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

	// ✅ Só checa pods se necessário
	if *deployment.Spec.Replicas > 0 && deployment.Status.AvailableReplicas == 0 {
		return m.checkPods(deployment, app)
	}

	message := fmt.Sprintf("✅ Deployment %s is healthy - %d/%d replicas available",
		app.Name, deployment.Status.AvailableReplicas, *deployment.Spec.Replicas)
	return DeploymentStatus{
		Level:    "HEALTHY",
		Message:  message,
		Critical: false,
	}
}

// ✅ Só chama pods quando realmente necessário
func (m *ApplicationMonitor) checkPods(deployment *v1.Deployment, app config.Application) DeploymentStatus {
	pods, err := m.clientset.CoreV1().Pods(app.Namespace).List(context.TODO(), metav1.ListOptions{
		LabelSelector: metav1.FormatLabelSelector(deployment.Spec.Selector),
		Limit:         10, // ✅ Limita resultado
	})

	if err != nil {
		message := fmt.Sprintf("⚠️ Failed to get pods for deployment %s: %v", app.Name, err)
		return DeploymentStatus{
			Level:    "WARNING",
			Message:  message,
			Critical: false,
		}
	}

	// Analyze pod statuses rapidamente
	runningPods := 0
	for _, pod := range pods.Items {
		if pod.Status.Phase == corev1.PodRunning {
			runningPods++
		} else if pod.Status.Phase == corev1.PodPending {
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
			Critical: runningPods == 0,
		}
	}

	message := fmt.Sprintf("✅ Deployment %s pods are healthy", app.Name)
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

	resp, err := httpClient.Post(app.WebhookURL, "application/json", strings.NewReader(string(payload)))
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
