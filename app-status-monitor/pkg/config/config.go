package config

import (
	"time"
)

type AppConfig struct {
	WatchInterval time.Duration `yaml:"watchInterval" json:"watchInterval"`
}

type MonitorRequestGVR struct {
	Group    string `json:"group"`
	Version  string `json:"version"`
	Resource string `json:"resource"`
}

type MonitorRequest struct {
	ID          string                 `json:"id"`
	Type        string                 `json:"type"`
	Name        string                 `json:"name"`
	Namespace   string                 `json:"namespace"`
	Timeout     time.Duration          `json:"timeout"`
	WebhookURL  string                 `json:"webhookUrl"`
	UserContext map[string]interface{} `json:"userContext"`
	ClusterName string                 `json:"clusterName"`
	GVR         MonitorRequestGVR      `json:"gvr"`
}

type MonitorResult struct {
	RequestID   string                 `json:"requestId"`
	Type        string                 `json:"type"`
	Name        string                 `json:"name"`
	Namespace   string                 `json:"namespace"`
	Status      string                 `json:"status"`
	Message     string                 `json:"message"`
	Timestamp   time.Time              `json:"timestamp"`
	UserContext map[string]interface{} `json:"userContext"`
	GVR         string                 `json:"gvr,omitempty"`
	ClusterName string                 `json:"clusterName,omitempty"`
}
