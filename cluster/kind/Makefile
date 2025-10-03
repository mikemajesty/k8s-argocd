.PHONY: install kind-up kind-down clean install-ingress-controller setup help check-dependencies

KIND_VERSION ?= v0.20.0
KIND_CLUSTER_NAME ?= argocd
BIN_DIR ?= ../../.bin

# ==============================================================================
# Dependências e Verificações
# ==============================================================================

$(BIN_DIR):
	@mkdir -p $(BIN_DIR)

$(BIN_DIR)/kind: $(BIN_DIR)
	@echo "📦 Verificando a instalação do Kind..."
	@if [ ! -f $(BIN_DIR)/kind ]; then \
		echo "⬇️  Baixando Kind versão $(KIND_VERSION)..."; \
		GO111MODULE="on" go install sigs.k8s.io/kind@$(KIND_VERSION); \
		mv $$(go env GOPATH)/bin/kind $(BIN_DIR)/kind; \
		echo "✅ Kind instalado em $(BIN_DIR)/kind"; \
	else \
		echo "✅ Kind já instalado"; \
	fi

check-dependencies: $(BIN_DIR)/kind
	@echo "🔍 Verificando dependências..."
	@which kubectl >/dev/null 2>&1 || (echo "❌ kubectl não encontrado. Instale primeiro."; exit 1)
	@which $(BIN_DIR)/helm >/dev/null 2>&1 || (echo "❌ helm não encontrado. Instale primeiro."; exit 1)
	@echo "✅ Todas as dependências atendidas"

# ==============================================================================
# Gerenciamento do Cluster
# ==============================================================================

kind-up: check-dependencies
	@echo "🚀 Criando cluster Kind: $(KIND_CLUSTER_NAME)"
	@$(BIN_DIR)/kind create cluster --name $(KIND_CLUSTER_NAME) --config ./manifests/cluster.yml
	@echo "✅ Cluster Kind criado com sucesso"

kind-down: $(BIN_DIR)/kind
	@echo "🛑 Deletando cluster Kind: $(KIND_CLUSTER_NAME)"
	@$(BIN_DIR)/kind delete cluster --name $(KIND_CLUSTER_NAME)
	@echo "✅ Cluster Kind deletado"

# ==============================================================================
# Ingress Controller (VERSÃO SIMPLIFICADA)
# ==============================================================================

install-ingress-controller: check-dependencies
	@echo "🌐 Instalando NGINX Ingress Controller..."
	
	@echo "➕ Adicionando repositório Helm..."
	$(BIN_DIR)/helm repo add ingress-nginx https://kubernetes.github.io/ingress-nginx --force-update
	$(BIN_DIR)/helm repo update

	@echo "📦 Instalando Ingress Controller..."
	$(BIN_DIR)/helm upgrade --install ingress-nginx ingress-nginx/ingress-nginx \
		--namespace ingress-nginx \
		--create-namespace \
		--set controller.service.type=NodePort \
		--set controller.hostNetwork=true \
		--wait
	
	@echo "⏳ Aguardando Ingress Controller ficar ready..."
	@kubectl wait --namespace ingress-nginx \
		--for=condition=ready pod \
		--selector=app.kubernetes.io/component=controller \
		--timeout=300s
	@echo "✅ NGINX Ingress Controller instalado e pronto"

# ==============================================================================
# Fluxos de Trabalho
# ==============================================================================

setup: kind-up install-ingress-controller
	@echo ""
	@echo "🎉 CLUSTER KIND CONFIGURADO!"
	@echo "📊 Verifique o cluster: kubectl cluster-info"
	@echo "🌐 Ingress Controller pronto na porta 80/443"
	@echo "🛑 Para desligar: make kind-down"

install: $(BIN_DIR)/kind
	@echo "✅ Kind instalado em $(BIN_DIR)/kind"

# ==============================================================================
# Utilitários
# ==============================================================================

clean:
	@echo "🧹 Limpando binários..."
	rm -rf $(BIN_DIR)
	@echo "✅ Binários removidos"

status:
	@echo "📊 Status do Cluster:"
	@kubectl cluster-info
	@echo ""
	@echo "📦 Pods do Ingress:"
	@kubectl get pods -n ingress-nginx

# ==============================================================================
# Ajuda
# ==============================================================================

help:
	@echo ""
	@echo "🏗️  SISTEMA DE GERENCIAMENTO DO CLUSTER KIND"
	@echo ""
	@echo "Comandos disponíveis:"
	@echo ""
	@echo "  \033[1;32mmake setup\033[0m"
	@echo "    Cria o cluster Kind e instala o NGINX Ingress Controller"
	@echo ""
	@echo "  \033[0;32mmake kind-up\033[0m"
	@echo "    Cria apenas o cluster Kind"
	@echo ""
	@echo "  \033[0;32mmake install-ingress-controller\033[0m"
	@echo "    Instala apenas o NGINX Ingress Controller"
	@echo ""
	@echo "  \033[0;32mmake kind-down\033[0m"
	@echo "    Deleta o cluster Kind"
	@echo ""
	@echo "  \033[0;32mmake status\033[0m"
	@echo "    Mostra status do cluster e ingress"
	@echo ""
	@echo "  \033[0;32mmake install\033[0m"
	@echo "    Instala apenas o binário do Kind"
	@echo ""
	@echo "  \033[0;32mmake clean\033[0m"
	@echo "    Remove os binários"
	@echo ""