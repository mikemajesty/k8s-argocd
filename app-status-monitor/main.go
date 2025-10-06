package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"meu-monitor/pkg/config"
	"meu-monitor/pkg/monitor"
)

type AppConfig struct {
	Port        string
	LogLevel    string
	LogFormat   string
	MaxWorkers  int
	HTTPTimeout time.Duration
}

func main() {
	// 🎯 Carrega configurações
	appConfig := loadConfig()
	setupLogger(appConfig)

	// 🚀 Cria o monitor
	monitorConfig := &config.AppConfig{
		WatchInterval: 30 * time.Second,
	}

	appMonitor, err := monitor.NewApplicationMonitor(monitorConfig)
	if err != nil {
		log.Fatalf("❌ Failed to create application monitor: %v", err)
	}

	log.Printf("🚀 Starting Crossplane Resource Monitor")
	log.Printf("📡 Port: %s", appConfig.Port)

	// 🏥 Health check
	http.HandleFunc("/health", healthHandler)

	// 📊 Metrics endpoint
	if getEnvBool("ENABLE_METRICS", true) {
		http.HandleFunc("/metrics", metricsHandler)
		log.Printf("📈 Metrics endpoint enabled at /metrics")
	}

	// 🎯 Endpoints principais
	http.HandleFunc("/api/monitor/start", startMonitoringHandler(appMonitor))
	http.HandleFunc("/api/monitor/stop", stopMonitoringHandler(appMonitor))
	http.HandleFunc("/api/monitor/sessions", sessionsHandler)
	http.HandleFunc("/", apiInfoHandler)

	// 🚀 Inicia servidor
	startServer(appConfig.Port)
}

// 🎯 Handler para iniciar monitoramento
func startMonitoringHandler(appMonitor *monitor.ApplicationMonitor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 🔒 CORS
		setupCORS(w)

		var req struct {
			Type        string                 `json:"type"`
			Name        string                 `json:"name"`
			Namespace   string                 `json:"namespace"`
			Timeout     string                 `json:"timeout"`
			WebhookURL  string                 `json:"webhookUrl"`
			UserContext map[string]interface{} `json:"userContext"`
			ClusterName string                 `json:"clusterName"`
			GVR         struct {
				Group    string `json:"group"`
				Version  string `json:"version"`
				Resource string `json:"resource"`
			} `json:"gvr"`
		}

		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		// ✅ Validações
		if req.ClusterName == "" || req.Name == "" || req.Namespace == "" || req.GVR.Resource == "" {
			http.Error(w, "clusterName, name, namespace and gvr.resource are required", http.StatusBadRequest)
			return
		}

		// ⏱️ Parse timeout
		timeout, err := time.ParseDuration(req.Timeout)
		if err != nil || timeout == 0 {
			timeout = 30 * time.Minute // Default para Crossplane
		}

		// 🔧 Prepara request
		userContext := req.UserContext
		if userContext == nil {
			userContext = make(map[string]interface{})
		}
		if req.WebhookURL != "" {
			userContext["webhookUrl"] = req.WebhookURL
		}

		monitorReq := config.MonitorRequest{
			ID:          generateID(),
			Type:        req.Type,
			Name:        req.Name,
			Namespace:   req.Namespace,
			Timeout:     timeout,
			WebhookURL:  req.WebhookURL,
			UserContext: userContext,
			ClusterName: req.ClusterName,
			GVR: config.MonitorRequestGVR{
				Group:    req.GVR.Group,
				Version:  req.GVR.Version,
				Resource: req.GVR.Resource,
			},
		}

		// ✅ Inicia monitoramento
		sessionID, err := appMonitor.StartMonitoring(context.Background(), monitorReq)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to start monitoring: %v", err), http.StatusInternalServerError)
			return
		}

		response := map[string]interface{}{
			"sessionId": sessionID,
			"status":    "monitoring_started",
			"resource":  fmt.Sprintf("%s/%s/%s", req.GVR.Group, req.GVR.Version, req.GVR.Resource),
			"cluster":   req.ClusterName,
			"namespace": req.Namespace,
			"timeout":   timeout.String(),
			"method":    "watch",
			"timestamp": time.Now().Format(time.RFC3339),
		}

		log.Printf("✅ Monitoring started - Session: %s, Resource: %s/%s/%s",
			sessionID, req.ClusterName, req.Namespace, req.Name)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(response)
	}
}

// 🛑 Handler para parar monitoramento
func stopMonitoringHandler(appMonitor *monitor.ApplicationMonitor) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.URL.Query().Get("sessionId")
		if sessionID == "" {
			http.Error(w, "sessionId is required", http.StatusBadRequest)
			return
		}

		if err := appMonitor.StopMonitoring(sessionID); err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":    "monitoring_stopped",
			"sessionId": sessionID,
		})
	}
}

// 📋 Handler para sessões ativas
func sessionsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"activeSessions": 0,
		"timestamp":      time.Now().Format(time.RFC3339),
	})
}

// 📚 Handler para info da API
func apiInfoHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"name":          "Crossplane Resource Monitor API",
		"version":       "1.0.0",
		"description":   "Generic Crossplane resource monitoring with Watch support",
		"compatibility": "Works with ANY Crossplane resource (AWS, GCP, Azure, Kafka, etc.)",
		"endpoints": map[string]string{
			"health":           "GET  /health",
			"start_monitoring": "POST /api/monitor/start",
			"stop_monitoring":  "GET  /api/monitor/stop?sessionId=<id>",
			"active_sessions":  "GET  /api/monitor/sessions",
			"metrics":          "GET  /metrics",
		},
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// 🏥 Health check handler
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"version":   "1.0.0",
	})
}

// 📊 Metrics handler
func metricsHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	fmt.Fprintf(w, "# HELP crossplane_monitor_active_sessions Current active monitoring sessions\n")
	fmt.Fprintf(w, "# TYPE crossplane_monitor_active_sessions gauge\n")
	fmt.Fprintf(w, "crossplane_monitor_active_sessions 0\n")
}

// 🔧 CORS setup
func setupCORS(w http.ResponseWriter) {
	if allowedOrigins := getEnv("ALLOWED_ORIGINS", "*"); allowedOrigins != "" {
		w.Header().Set("Access-Control-Allow-Origin", allowedOrigins)
		w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	}
}

// 🚀 Inicia servidor
func startServer(port string) {
	port = ":" + port
	log.Printf("🌐 Server starting on port %s", port)

	server := &http.Server{
		Addr:         port,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	if err := server.ListenAndServe(); err != nil {
		log.Fatalf("❌ Failed to start server: %v", err)
	}
}

// 🎯 Carrega configuração
func loadConfig() *AppConfig {
	return &AppConfig{
		Port:        getEnv("PORT", "8080"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		LogFormat:   getEnv("LOG_FORMAT", "text"),
		MaxWorkers:  getEnvInt("MAX_WORKERS", 10),
		HTTPTimeout: getEnvDuration("HTTP_CLIENT_TIMEOUT", 30*time.Second),
	}
}

// 📊 Configura logger
func setupLogger(config *AppConfig) {
	log.SetFlags(0)
	if config.LogFormat == "json" {
		log.SetFlags(log.LstdFlags)
	} else {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}
}

// 🆔 Gera ID único
func generateID() string {
	return fmt.Sprintf("mon-%d", time.Now().UnixNano())
}

// 🔧 Helper functions
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvInt(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		if boolValue, err := strconv.ParseBool(value); err == nil {
			return boolValue
		}
	}
	return defaultValue
}

func getEnvDuration(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
