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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/watch"
)

type MonitorSession struct {
	Request    config.MonitorRequest
	CancelFunc context.CancelFunc
	StartTime  time.Time
}

type ApplicationMonitor struct {
	config         *config.AppConfig
	clusterManager *ClusterManager
	mutex          sync.RWMutex
	activeSessions map[string]*MonitorSession
	resultsChan    chan config.MonitorResult
}

// HTTP Client otimizado
var httpClient = &http.Client{
	Timeout: 10 * time.Second,
	Transport: &http.Transport{
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		IdleConnTimeout:     30 * time.Second,
	},
}

func NewApplicationMonitor(cfg *config.AppConfig) (*ApplicationMonitor, error) {
	monitor := &ApplicationMonitor{
		config:         cfg,
		clusterManager: NewClusterManager(),
		activeSessions: make(map[string]*MonitorSession),
		resultsChan:    make(chan config.MonitorResult, 100),
	}

	go monitor.resultWorker()
	return monitor, nil
}

// StartMonitoring inicia monitoramento com clusterName obrigatório
func (m *ApplicationMonitor) StartMonitoring(ctx context.Context, req config.MonitorRequest) (string, error) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if _, exists := m.activeSessions[req.ID]; exists {
		return "", fmt.Errorf("monitoring session already exists for ID: %s", req.ID)
	}

	if req.ClusterName == "" {
		return "", fmt.Errorf("clusterName is required")
	}

	timeout := req.Timeout
	if timeout == 0 {
		timeout = 30 * time.Minute // Default para recursos Crossplane
	}

	monitorCtx, cancel := context.WithTimeout(ctx, timeout)

	session := &MonitorSession{
		Request:    req,
		CancelFunc: cancel,
		StartTime:  time.Now(),
	}

	m.activeSessions[req.ID] = session

	// ✅ APENAS WATCH - SEM FALLBACK
	go m.monitorWithWatch(monitorCtx, req)

	log.Printf("🚀 Started monitoring session %s for %s %s/%s in cluster %s",
		req.ID, req.GVR.Resource, req.Namespace, req.Name, req.ClusterName)

	return req.ID, nil
}

func (m *ApplicationMonitor) StopMonitoring(sessionID string) error {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	session, exists := m.activeSessions[sessionID]
	if !exists {
		return fmt.Errorf("monitoring session not found: %s", sessionID)
	}

	session.CancelFunc()
	delete(m.activeSessions, sessionID)

	log.Printf("🛑 Stopped monitoring session %s", sessionID)
	return nil
}

// ⚡️ APENAS WATCH - SEM FALLBACK
func (m *ApplicationMonitor) monitorWithWatch(ctx context.Context, req config.MonitorRequest) {
	log.Printf("👀 Starting WATCH for %s %s/%s in cluster %s",
		req.GVR.Resource, req.Namespace, req.Name, req.ClusterName)

	client, err := m.clusterManager.GetClientForCluster(req.ClusterName)
	if err != nil {
		m.sendErrorResult(req, fmt.Sprintf("Failed to get client: %v", err))
		return
	}

	gvr := schema.GroupVersionResource{
		Group:    req.GVR.Group,
		Version:  req.GVR.Version,
		Resource: req.GVR.Resource,
	}

	// ✅ Tenta criar watcher - se falhar, erro direto
	watcher, err := client.Resource(gvr).Namespace(req.Namespace).Watch(ctx, metav1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", req.Name),
	})

	if err != nil {
		log.Printf("❌ Watch failed for %s: %v", req.Name, err)
		m.sendErrorResult(req, fmt.Sprintf("Watch failed: %v", err))
		return
	}
	defer watcher.Stop()

	log.Printf("📡 Watch established for %s, waiting for changes...", req.Name)

	// ✅ Processa eventos do Watch
	for {
		select {
		case <-ctx.Done():
			if ctx.Err() == context.DeadlineExceeded {
				m.sendTimeoutResult(req)
			}
			return

		case event, ok := <-watcher.ResultChan():
			if !ok {
				log.Printf("🔁 Watch channel closed for %s", req.Name)
				m.sendErrorResult(req, "Watch channel closed")
				return
			}

			// ✅ Processa APENAS quando há mudança real
			switch event.Type {
			case watch.Added, watch.Modified:
				if obj, ok := event.Object.(*unstructured.Unstructured); ok {
					log.Printf("🔄 Event %s received for %s", event.Type, req.Name)
					status, message, done := m.analyzeCrossplaneResourceStatus(obj, req.Type)
					if done {
						log.Printf("✅ Resource %s reached final state: %s", req.Name, status)
						m.sendFinalResult(req, status, message)
						return
					} else {
						log.Printf("⏳ Resource %s still pending: %s", req.Name, message)
					}
				}

			case watch.Deleted:
				log.Printf("🗑️ Resource %s was deleted", req.Name)
				m.sendErrorResult(req, fmt.Sprintf("Resource %s was deleted", req.Name))
				return

			case watch.Error:
				log.Printf("⚠️ Watch error for %s: %v", req.Name, event.Object)
				m.sendErrorResult(req, fmt.Sprintf("Watch error: %v", event.Object))
				return
			}
		}
	}
}

// ============================================================================
// ✅ VALIDAÇÃO GENÉRICA PARA QUALQUER RECURSO CROSSPLANE
// ============================================================================

func (m *ApplicationMonitor) analyzeCrossplaneResourceStatus(obj *unstructured.Unstructured, resourceType string) (string, string, bool) {
	name := obj.GetName()

	// ✅ CONDIÇÕES PADRÃO CROSSPLANE - Funciona para QUALQUER recurso
	conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if found {
		for _, condition := range conditions {
			if conditionMap, ok := condition.(map[string]interface{}); ok {
				conditionType, _ := conditionMap["type"].(string)
				status, _ := conditionMap["status"].(string)
				reason, _ := conditionMap["reason"].(string)
				message, _ := conditionMap["message"].(string)

				// ✅ PADRÃO CROSSPLANE: Ready e Synced
				if conditionType == "Ready" || conditionType == "Synced" {
					switch status {
					case "True":
						return "SUCCESS", fmt.Sprintf("✅ %s '%s' is ready and synced", resourceType, name), true
					case "False":
						return "ERROR", fmt.Sprintf("❌ %s '%s' failed: %s - %s", resourceType, name, reason, message), true
					case "Unknown":
						return "PENDING", fmt.Sprintf("⏳ %s '%s' status unknown: %s", resourceType, name, message), false
					}
				}
			}
		}
	}

	// ✅ FALLBACK: Verifica fases genéricas (opcional)
	if phase, found, _ := unstructured.NestedString(obj.Object, "status", "phase"); found {
		switch phase {
		case "Ready", "Running", "Succeeded", "Bound", "Available":
			return "SUCCESS", fmt.Sprintf("✅ %s '%s' is ready", resourceType, name), true
		case "Failed", "Error":
			return "ERROR", fmt.Sprintf("❌ %s '%s' failed", resourceType, name), true
		}
	}

	// ⏳ Recurso ainda não tem condições definidas - ainda provisionando
	return "PENDING", fmt.Sprintf("⏳ %s '%s' is being provisioned", resourceType, name), false
}

// ============================================================================
// ✅ MÉTODOS AUXILIARES
// ============================================================================

func (m *ApplicationMonitor) sendFinalResult(req config.MonitorRequest, status, message string) {
	m.resultsChan <- config.MonitorResult{
		RequestID:   req.ID,
		Type:        req.Type,
		Name:        req.Name,
		Namespace:   req.Namespace,
		Status:      status,
		Message:     message,
		Timestamp:   time.Now(),
		UserContext: req.UserContext,
		GVR:         fmt.Sprintf("%s/%s/%s", req.GVR.Group, req.GVR.Version, req.GVR.Resource),
		ClusterName: req.ClusterName,
	}

	m.mutex.Lock()
	delete(m.activeSessions, req.ID)
	m.mutex.Unlock()

	log.Printf("📤 Final result sent for %s: %s", req.Name, status)
}

func (m *ApplicationMonitor) sendErrorResult(req config.MonitorRequest, message string) {
	m.sendFinalResult(req, "ERROR", message)
}

func (m *ApplicationMonitor) sendTimeoutResult(req config.MonitorRequest) {
	m.sendFinalResult(req, "TIMEOUT", fmt.Sprintf("Monitoring timeout after %v", req.Timeout))
}

func (m *ApplicationMonitor) resultWorker() {
	for result := range m.resultsChan {
		log.Printf("📦 Result for session %s: %s - %s", result.RequestID, result.Status, result.Message)

		// Envia webhook se configurado
		if result.UserContext["webhookUrl"] != nil {
			m.sendResultWebhook(result)
		}
	}
}

func (m *ApplicationMonitor) sendResultWebhook(result config.MonitorResult) {
	webhookURL, ok := result.UserContext["webhookUrl"].(string)
	if !ok || webhookURL == "" {
		return
	}

	payload, err := json.Marshal(result)
	if err != nil {
		log.Printf("❌ Failed to marshal result webhook: %v", err)
		return
	}

	resp, err := httpClient.Post(webhookURL, "application/json", strings.NewReader(string(payload)))
	if err != nil {
		log.Printf("❌ Failed to send result webhook: %v", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		log.Printf("⚠️ Webhook returned error status: %d", resp.StatusCode)
	} else {
		log.Printf("📤 Result webhook sent successfully for session %s", result.RequestID)
	}
}
