package monitor

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type ClusterManager struct {
	clients map[string]dynamic.Interface
	mutex   sync.RWMutex
}

func NewClusterManager() *ClusterManager {
	return &ClusterManager{
		clients: make(map[string]dynamic.Interface),
	}
}

// GetClientForCluster cria ou retorna um client para um cluster pelo nome
func (cm *ClusterManager) GetClientForCluster(clusterName string) (dynamic.Interface, error) {
	cm.mutex.RLock()
	client, exists := cm.clients[clusterName]
	cm.mutex.RUnlock()

	if exists {
		return client, nil
	}

	// Cria novo client
	cm.mutex.Lock()
	defer cm.mutex.Unlock()

	// Double-check
	if client, exists := cm.clients[clusterName]; exists {
		return client, nil
	}

	client, err := createDynamicClientForCluster(clusterName)
	if err != nil {
		return nil, fmt.Errorf("failed to create client for cluster %s: %v", clusterName, err)
	}

	cm.clients[clusterName] = client
	log.Printf("🔗 Created new client for cluster: %s", clusterName)
	return client, nil
}

// createDynamicClientForCluster cria client baseado no nome do cluster
func createDynamicClientForCluster(clusterName string) (dynamic.Interface, error) {
	var restConfig *rest.Config
	var err error

	// Tenta carregar do kubeconfig padrão primeiro
	kubeconfigPath := getKubeconfigPath()
	if kubeconfigPath != "" {
		restConfig, err = clientcmd.NewNonInteractiveDeferredLoadingClientConfig(
			&clientcmd.ClientConfigLoadingRules{ExplicitPath: kubeconfigPath},
			&clientcmd.ConfigOverrides{CurrentContext: clusterName},
		).ClientConfig()

		if err == nil {
			log.Printf("✅ Using kubeconfig context: %s", clusterName)
			return createClientWithConfig(restConfig)
		}
		log.Printf("⚠️ Failed to use kubeconfig context %s: %v", clusterName, err)
	}

	// Se não encontrou no kubeconfig, tenta configuração in-cluster
	log.Printf("🔄 Trying in-cluster config for context: %s", clusterName)
	restConfig, err = rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("failed to get config for cluster %s: %v", clusterName, err)
	}

	return createClientWithConfig(restConfig)
}

func createClientWithConfig(restConfig *rest.Config) (dynamic.Interface, error) {
	// Timeouts otimizados
	restConfig.Timeout = 15 * time.Second
	restConfig.QPS = 50
	restConfig.Burst = 100

	return dynamic.NewForConfig(restConfig)
}

func getKubeconfigPath() string {
	// 1. KUBECONFIG env variable
	if kubeconfig := os.Getenv("KUBECONFIG"); kubeconfig != "" {
		return kubeconfig
	}

	// 2. Default kubeconfig location
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	defaultPath := filepath.Join(home, ".kube", "config")
	if _, err := os.Stat(defaultPath); err == nil {
		return defaultPath
	}

	return ""
}
