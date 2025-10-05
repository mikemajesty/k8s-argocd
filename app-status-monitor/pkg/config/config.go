// pkg/config/config.go
package config

import (
	"log"
	"os"
	"time"

	"gopkg.in/yaml.v2"
)

// AppConfig - estrutura principal de configuração
type AppConfig struct {
	WatchInterval time.Duration `yaml:"watchInterval"`
	Applications  []Application `yaml:"applications"`
}

// Application - configuração de cada aplicação monitorada
type Application struct {
	Name         string `yaml:"appName"`
	Namespace    string `yaml:"namespace"`
	WebhookURL   string `yaml:"webhookUrl"`
	CriticalOnly bool   `yaml:"criticalOnly"`
}

// LoadConfig - carrega configuração do arquivo YAML
func LoadConfig(path string) (*AppConfig, error) {
	log.Printf("📁 Loading config from: %s", path) // ← ADD THIS

	data, err := os.ReadFile(path)
	if err != nil {
		log.Printf("❌ Failed to read config file: %v", err) // ← ADD THIS
		return nil, err
	}

	log.Printf("📄 Config file size: %d bytes", len(data)) // ← ADD THIS

	var config AppConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		log.Printf("❌ Failed to parse YAML: %v", err) // ← ADD THIS
		return nil, err
	}

	log.Printf("✅ Successfully loaded %d applications", len(config.Applications)) // ← ADD THIS
	return &config, nil
}

// GetEnv - helper para variáveis de ambiente com valor padrão
func GetEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
