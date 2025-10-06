package main

import (
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
	// 🎯 Carrega configurações from environment variables
	appConfig := loadConfig()

	// 📊 Configura logger
	setupLogger(appConfig)

	// 🚀 Cria o monitor
	monitorConfig := &config.AppConfig{
		WatchInterval: 30 * time.Second, // Para compatibilidade
	}

	appMonitor, err := monitor.NewApplicationMonitor(monitorConfig)
	if err != nil {
		log.Fatalf("❌ Failed to create application monitor: %v", err)
	}

	log.Printf("🚀 Starting Application Monitor")
	log.Printf("📡 Port: %s", appConfig.Port)
	log.Printf("📊 Log Level: %s", appConfig.LogLevel)
	log.Printf("⚡ Max Workers: %d", appConfig.MaxWorkers)
	log.Printf("🌐 HTTP Timeout: %v", appConfig.HTTPTimeout)

	// 🏥 Health check
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "healthy",
			"timestamp": time.Now().Format(time.RFC3339),
			"version":   "1.0.0",
		})
	})

	// 📊 Metrics endpoint (se habilitado)
	if getEnvBool("ENABLE_METRICS", true) {
		http.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/plain")
			// TODO: Implementar métricas Prometheus
			fmt.Fprintf(w, "# HELP app_monitor_active_sessions Current active monitoring sessions\n")
			fmt.Fprintf(w, "# TYPE app_monitor_active_sessions gauge\n")
			fmt.Fprintf(w, "app_monitor_active_sessions 0\n")
		})
		log.Printf("📈 Metrics endpoint enabled at /metrics")
	}

	// 🎯 Iniciar monitoramento
	http.HandleFunc("/api/monitor/start", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		// 🔒 CORS headers
		if allowedOrigins := getEnv("ALLOWED_ORIGINS", "*"); allowedOrigins != "" {
			w.Header().Set("Access-Control-Allow-Origin", allowedOrigins)
			w.Header().Set("Access-Control-Allow-Methods", "POST, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		}

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusOK)
			return
		}

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
			log.Printf("❌ Invalid request body: %v", err)
			http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
			return
		}

		// ✅ Validações
		if req.ClusterName == "" {
			http.Error(w, "clusterName is required", http.StatusBadRequest)
			return
		}

		if req.Name == "" || req.Namespace == "" || req.GVR.Resource == "" {
			http.Error(w, "name, namespace and gvr.resource are required", http.StatusBadRequest)
			return
		}

		// ⏱️ Parse timeout com valores padrão inteligentes
		timeout, err := time.ParseDuration(req.Timeout)
		if err != nil || timeout == 0 {
			timeout = getDefaultTimeout(req.Type)
		}

		// 🔧 Prepara userContext
		userContext := req.UserContext
		if userContext == nil {
			userContext = make(map[string]interface{})
		}
		if req.WebhookURL != "" {
			userContext["webhookUrl"] = req.WebhookURL
		}

		// Adiciona metadados ao userContext
		userContext["requestTimestamp"] = time.Now().Format(time.RFC3339)
		userContext["timeout"] = timeout.String()

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

		sessionID, err := appMonitor.StartMonitoring(r.Context(), monitorReq)
		if err != nil {
			log.Printf("❌ Failed to start monitoring: %v", err)
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
			"method":    "watch", // Indica que está usando Watch
			"timestamp": time.Now().Format(time.RFC3339),
		}

		log.Printf("✅ Monitoring started - Session: %s, Resource: %s/%s/%s",
			sessionID, req.ClusterName, req.Namespace, req.Name)

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		json.NewEncoder(w).Encode(response)
	})

	// 🛑 Parar monitoramento
	http.HandleFunc("/api/monitor/stop", func(w http.ResponseWriter, r *http.Request) {
		sessionID := r.URL.Query().Get("sessionId")
		if sessionID == "" {
			http.Error(w, "sessionId is required", http.StatusBadRequest)
			return
		}

		if err := appMonitor.StopMonitoring(sessionID); err != nil {
			log.Printf("❌ Failed to stop monitoring session %s: %v", sessionID, err)
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		log.Printf("✅ Monitoring stopped - Session: %s", sessionID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"status":    "monitoring_stopped",
			"sessionId": sessionID,
		})
	})

	// 📋 Status das sessões ativas
	http.HandleFunc("/api/monitor/sessions", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		// TODO: Implementar listagem de sessões ativas
		json.NewEncoder(w).Encode(map[string]interface{}{
			"activeSessions": 0,
			"timestamp":      time.Now().Format(time.RFC3339),
		})
	})

	// 📚 API Info
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"name":        "Application Monitor API",
			"version":     "1.0.0",
			"description": "Kubernetes resource monitoring with Watch support",
			"endpoints": map[string]string{
				"health":           "GET  /health",
				"start_monitoring": "POST /api/monitor/start",
				"stop_monitoring":  "GET  /api/monitor/stop?sessionId=<id>",
				"active_sessions":  "GET  /api/monitor/sessions",
				"metrics":          "GET  /metrics",
			},
			"timestamp": time.Now().Format(time.RFC3339),
		})
	})

	// 🚀 Inicia servidor
	port := ":" + appConfig.Port
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

// 🎯 Carrega configuração from environment variables
func loadConfig() *AppConfig {
	config := &AppConfig{
		Port:        getEnv("PORT", "8080"),
		LogLevel:    getEnv("LOG_LEVEL", "info"),
		LogFormat:   getEnv("LOG_FORMAT", "text"),
		MaxWorkers:  getEnvInt("MAX_WORKERS", 10),
		HTTPTimeout: getEnvDuration("HTTP_CLIENT_TIMEOUT", 30*time.Second),
	}
	return config
}

// 📊 Configura o logger baseado nas envs
func setupLogger(config *AppConfig) {
	log.SetFlags(0) // Remove flags padrão

	if config.LogFormat == "json" {
		// TODO: Implementar logger JSON structured
		log.SetFlags(log.LstdFlags)
	} else {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	// Log level seria implementado com um logger mais avançado
	log.Printf("🔧 Logger configured - Level: %s, Format: %s", config.LogLevel, config.LogFormat)
}

// ⏱️ Retorna timeout padrão baseado no tipo de recurso
func getDefaultTimeout(resourceType string) time.Duration {
	switch resourceType {
	case "kafkatopic":
		return getEnvDuration("DEFAULT_TIMEOUT_KAFKA", 20*time.Minute)
	case "deployment":
		return getEnvDuration("DEFAULT_TIMEOUT_DEPLOYMENT", 10*time.Minute)
	case "postgresqlinstance", "mysqlinstance":
		return getEnvDuration("DEFAULT_TIMEOUT_DATABASE", 25*time.Minute)
	case "rediscluster":
		return getEnvDuration("DEFAULT_TIMEOUT_REDIS", 15*time.Minute)
	default:
		return 15 * time.Minute
	}
}

// 🆔 Gera ID único para sessão
func generateID() string {
	return fmt.Sprintf("mon-%d", time.Now().UnixNano())
}

// 🔧 Helper functions para environment variables
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
