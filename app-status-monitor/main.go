// main.go
package main

import (
	"context"
	"log"
	"meu-monitor/pkg/config"
	"meu-monitor/pkg/monitor"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	log.Println("🚀 Starting Application Status Monitor...")

	configPath := config.GetEnv("CONFIG_PATH", "/etc/monitor/config.yaml")
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("❌ Failed to load config: %v", err)
	}

	log.Printf("✅ Loaded config for %d applications", len(cfg.Applications))

	// Lista as aplicações carregadas
	for _, app := range cfg.Applications {
		log.Printf("   📱 App: %s, Namespace: %s, Webhook: %s",
			app.Name, app.Namespace, app.WebhookURL)
	}

	// Create monitor
	appMonitor, err := monitor.NewApplicationMonitor(cfg)
	if err != nil {
		log.Fatalf("❌ Failed to create monitor: %v", err)
	}

	// Setup graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Handle signals
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigCh
		log.Printf("📦 Received signal: %v, shutting down...", sig)
		cancel()
		time.Sleep(2 * time.Second)
		os.Exit(0)
	}()

	// Start monitoring
	log.Println("🟢 Starting monitoring loop...")
	if err := appMonitor.Start(ctx); err != nil {
		log.Fatalf("❌ Monitor failed: %v", err)
	}
}
