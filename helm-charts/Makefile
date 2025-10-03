.PHONY: setup install-helm install-argocd add-argo-repo install-argo-release \
        get-password login-argo status-argocd clean help check-cluster install-argo-fast debug-path

# Variáveis
HELM_VERSION ?= v3.13.1
ARGOCD_VERSION ?= v2.9.7
BIN_DIR ?= ../.bin
ARGOCD_HELM_REPO ?= https://argoproj.github.io/argo-helm
ARGOCD_RELEASE_NAME ?= argo-cd
ARGOCD_NAMESPACE ?= argocd
INGRESS_HOST ?= argocd.app.io

PROJECT_ROOT ?= $(CURDIR)
VALUES_FILE ?= $(PROJECT_ROOT)/argo-cd/values.yml

# ==============================================================================
# Verificações
# ==============================================================================

check-cluster:
	@echo "🔍 Verificando conexão com o cluster..."
	@kubectl cluster-info >/dev/null 2>&1 || (echo "❌ Cluster não acessível. Execute 'make setup' no cluster Kind primeiro."; exit 1)
	@echo "✅ Cluster conectado"

check-ingress:
	@echo "🔍 Verificando Ingress Controller..."
	@kubectl get pods -n ingress-nginx >/dev/null 2>&1 || (echo "❌ Ingress Controller não instalado. Execute 'make install-ingress-controller' primeiro."; exit 1)
	@echo "✅ Ingress Controller pronto"

check-values-file:
	@echo "🔍 Verificando se values.yml existe..."
	@test -f $(VALUES_FILE) || (echo "❌ ARQUIVO NÃO ENCONTRADO: $(VALUES_FILE)"; exit 1)
	@echo "✅ Values file encontrado: $(VALUES_FILE)"

debug-path:
	@echo "PROJECT_ROOT: $(PROJECT_ROOT)"
	@echo "VALUES_FILE: $(VALUES_FILE)"
	@echo "Arquivo existe? $$(test -f $(VALUES_FILE) && echo 'SIM' || echo 'NÃO')"

# ==============================================================================
# Instalação de Binários
# ==============================================================================

$(BIN_DIR):
	@mkdir -p $(BIN_DIR)

install-helm: $(BIN_DIR)
	@echo "📦 Verificando Helm..."
	@if [ ! -f $(BIN_DIR)/helm ]; then \
		echo "⬇️  Baixando Helm $(HELM_VERSION)..."; \
		HELM_PLATFORM=$$(uname -s | tr '[:upper:]' '[:lower:]')-amd64; \
		curl -sSL "https://get.helm.sh/helm-$(HELM_VERSION)-$${HELM_PLATFORM}.tar.gz" | tar xz -C /tmp; \
		mv /tmp/$${HELM_PLATFORM}/helm $(BIN_DIR)/helm; \
		rm -rf /tmp/$${HELM_PLATFORM}; \
		echo "✅ Helm instalado"; \
	else \
		echo "✅ Helm já instalado"; \
	fi

install-argocd: $(BIN_DIR)
	@echo "📦 Verificando ArgoCD CLI..."
	@if [ ! -f $(BIN_DIR)/argocd ]; then \
		echo "⬇️  Baixando ArgoCD CLI $(ARGOCD_VERSION)..."; \
		ARGO_PLATFORM=$$(uname -s | tr '[:upper:]' '[:lower:]')-amd64; \
		curl -sSL -o $(BIN_DIR)/argocd "https://github.com/argoproj/argo-cd/releases/download/$(ARGOCD_VERSION)/argocd-$${ARGO_PLATFORM}"; \
		chmod +x $(BIN_DIR)/argocd; \
		echo "✅ ArgoCD CLI instalado"; \
	else \
		echo "✅ ArgoCD CLI já instalado"; \
	fi

# ==============================================================================
# Instalação do ArgoCD
# ==============================================================================

add-argo-repo: install-helm
	@echo "📚 Adicionando repositório Helm do ArgoCD..."
	$(BIN_DIR)/helm repo add argo-helm $(ARGOCD_HELM_REPO) --force-update
	$(BIN_DIR)/helm repo update
	@echo "✅ Repositório adicionado"

install-argo-fast: check-cluster check-ingress add-argo-repo
	@echo "🚀 Instalando ArgoCD RÁPIDO (~30 segundos)..."
	
	@echo "🧹 Limpando instalação anterior..."
	-$(BIN_DIR)/helm uninstall $(ARGOCD_RELEASE_NAME) --namespace $(ARGOCD_NAMESPACE) --ignore-not-found
	
	@echo "📦 Instalando ArgoCD (sem wait)..."
	$(BIN_DIR)/helm upgrade --install $(ARGOCD_RELEASE_NAME) argo-helm/argo-cd \
		--namespace $(ARGOCD_NAMESPACE) \
		--create-namespace \
		--values $(PROJECT_ROOT)/argo-cd/values.yml
	
	echo "⏳ Aguardando 15 segundos..."
	@sleep 15
	@echo "✅ ArgoCD instalado!"
	
	@echo "🔍 Verificando status dos pods..."
	@kubectl get pods -n $(ARGOCD_NAMESPACE) -l app.kubernetes.io/part-of=argocd
	
	@echo "✅ ArgoCD instalado! Use 'make status-argocd' para acompanhar."

# Versão com wait (se realmente precisar)
install-argo-release: check-cluster check-ingress add-argo-repo
	@echo "🚀 Instalando ArgoCD com wait (~60 segundos)..."
	
	@echo "🧹 Limpando instalação anterior..."
	-$(BIN_DIR)/helm uninstall $(ARGOCD_RELEASE_NAME) --namespace $(ARGOCD_NAMESPACE) --ignore-not-found
	
	@echo "📦 Instalando ArgoCD..."
	$(BIN_DIR)/helm upgrade --install $(ARGOCD_RELEASE_NAME) argo-helm/argo-cd \
		--namespace $(ARGOCD_NAMESPACE) \
		--create-namespace \
		--values ./argo-cd/values.yml \
		--wait \
		--timeout 90s
	
	@echo "✅ ArgoCD instalado com sucesso!"

# ==============================================================================
# Acesso e Gerenciamento
# ==============================================================================

get-password: check-cluster
	@echo "🔐 Obtendo senha admin do ArgoCD..."
	@kubectl -n $(ARGOCD_NAMESPACE) get secret argocd-initial-admin-secret -o jsonpath="{.data.password}" | base64 -d 2>/dev/null || \
	(echo "❌ Secret não encontrado. Aguarde a instalação completar."; exit 1)

login-argo: get-password
	@echo ""
	@echo "🌐 ARGOCD PRONTO!"
	@echo "📍 URL: https://$(INGRESS_HOST)"
	@echo "👤 Usuário: admin"
	@echo "🔐 Senha: $$(kubectl -n $(ARGOCD_NAMESPACE) get secret argocd-initial-admin-secret -o jsonpath='{.data.password}' | base64 -d)"
	@echo ""
	@echo "💡 Dica: Certifique-se de que '$(INGRESS_HOST)' está no /etc/hosts apontando para 127.0.0.1"

status-argocd: check-cluster
	@echo "📊 Status do ArgoCD:"
	@kubectl get pods -n $(ARGOCD_NAMESPACE) -l app.kubernetes.io/part-of=argocd

wait-argocd:
	@echo "⏳ Aguardando ArgoCD ficar pronto..."
	@kubectl wait --for=condition=ready pod \
		--selector=app.kubernetes.io/part-of=argocd \
		--namespace $(ARGOCD_NAMESPACE) \
		--timeout=120s
	@echo "✅ ArgoCD pronto!"

# ==============================================================================
# Fluxo Principal
# ==============================================================================

setup: check-cluster check-ingress install-helm install-argocd install-argo-fast login-argo
	@echo ""
	@echo "🎉 ARGOCD CONFIGURADO COM SUCESSO!"
	@echo "✅ Cluster verificado"
	@echo "✅ Ingress verificado" 
	@echo "✅ Binários instalados"
	@echo "✅ ArgoCD implantado"
	@echo ""
	@echo "📋 Próximos passos:"
	@echo "   make status-argocd  - Verificar status dos pods"
	@echo "   make wait-argocd    - Aguardar ficar 100% pronto"
	@echo "   Acesse: https://$(INGRESS_HOST)"

setup-wait: check-cluster check-ingress install-helm install-argocd install-argo-release login-argo
	@echo ""
	@echo "🎉 ARGOCD CONFIGURADO COM SUCESSO (com wait)!"

# ==============================================================================
# Utilitários
# ==============================================================================

clean:
	@echo "🧹 Removendo binários..."
	rm -rf $(BIN_DIR)
	@echo "✅ Binários removidos"

# ==============================================================================
# Ajuda
# ==============================================================================

help:
	@echo ""
	@echo "🚀 SISTEMA DE GERENCIAMENTO DO ARGOCD"
	@echo ""
	@echo "Comandos disponíveis:"
	@echo ""
	@echo "  \033[1;32mmake setup\033[0m              - Instalação RÁPIDA (~30s)"
	@echo "  \033[0;32mmake setup-wait\033[0m         - Instalação com wait (~90s)"
	@echo "  \033[0;32mmake install-argo-fast\033[0m  - Apenas instalação rápida"
	@echo "  \033[0;32mmake install-argo-release\033[0m - Instalação com wait"
	@echo "  \033[0;32mmake login-argo\033[0m         - Mostra URL e credenciais"
	@echo "  \033[0;32mmake status-argocd\033[0m      - Status dos pods"
	@echo "  \033[0;32mmake wait-argocd\033[0m        - Aguarda ficar pronto"
	@echo "  \033[0;32mmake clean\033[0m              - Remove binários"
	@echo ""