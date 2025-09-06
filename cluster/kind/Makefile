.PHONY: setup kind-up kind-down clean install-ingress-controller start-cluster help

KIND_VERSION ?= v0.20.0
KIND_CLUSTER_NAME ?= argocd

# Variável para o diretório de binários
BIN_DIR ?= ./bin

# ==============================================================================
# Instalação do Kind
# ==============================================================================

# Cria o diretório de binários se não existir
$(BIN_DIR):
	mkdir -p $(BIN_DIR)

# Alvo para instalar o Kind
$(BIN_DIR)/kind: $(BIN_DIR)
	@echo "-> Verificando a instalação do Kind..."
	@if [ ! -f $(BIN_DIR)/kind ]; then \
		echo "-> Kind não encontrado. Baixando a versão $(KIND_VERSION)..."; \
		GO111MODULE="on" go install sigs.k8s.io/kind@$(KIND_VERSION); \
		mv $$(go env GOPATH)/bin/kind $(BIN_DIR)/kind; \
	fi

# ==============================================================================
# Gerenciamento do Cluster
# ==============================================================================

# Cria o cluster Kind com a configuração de portas
kind-up: $(BIN_DIR)/kind
	@echo "-> Criando o cluster Kind com o nome: $(KIND_CLUSTER_NAME)..."
	@$(BIN_DIR)/kind create cluster --name $(KIND_CLUSTER_NAME) --config ./manifests/cluster.yml

# Deleta o cluster Kind
kind-down: $(BIN_DIR)/kind
	@echo "-> Deletando o cluster Kind com o nome: $(KIND_CLUSTER_NAME)..."
	@$(BIN_DIR)/kind delete cluster --name $(KIND_CLUSTER_NAME)

# ==============================================================================
# Instalação de Componentes do Cluster
# ==============================================================================

# Instala o Ingress Controller no Kind
install-ingress-controller:
	@echo "-> Instalando o NGINX Ingress Controller..."
	@kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
	@echo "-> Dando um tempo para o cluster processar..."
	@sleep 60
	@echo "-> Aguardando a implantação do NGINX Ingress Controller..."
	@kubectl wait --namespace ingress-nginx \
		--for=condition=ready pod \
		--selector=app.kubernetes.io/component=controller \
		--timeout=180s
	@echo "-> NGINX Ingress Controller instalado com sucesso!"

# ==============================================================================
# Fluxo de Trabalho
# ==============================================================================

# Alvo principal para iniciar o cluster e instalar os componentes
start-cluster: kind-up install-ingress-controller
	@echo "-> Cluster e Ingress Controller prontos. Use 'make kind-down' para desligar."

# ==============================================================================
# Alvos Utilitários
# ==============================================================================

# Alvo para configurar o ambiente (instala o Kind)
setup: $(BIN_DIR)/kind
	@echo "-> Configuração do ambiente concluída. Use 'make start-cluster' para iniciar tudo."

# Alvo para limpar arquivos gerados
clean:
	@echo "-> Removendo o diretório de binários..."
	rm -rf $(BIN_DIR)

# ==============================================================================
# Ajuda
# ==============================================================================

help:
	@echo ""
	@echo "Bem-vindo ao sistema de gerenciamento de cluster!"
	@echo ""
	@echo "Comandos disponíveis:"
	@echo ""
	@echo "  \033[0;32mmake setup\033[0m"
	@echo "    Instala o binário do Kind no diretório ./bin do projeto."
	@echo ""
	@echo "  \033[1;32mmake start-cluster\033[0m"
	@echo "    Cria o cluster Kind e instala o NGINX Ingress Controller. Este é o comando para iniciar tudo."
	@echo ""
	@echo "  \033[0;32mmake kind-up\033[0m"
	@echo "    Inicia apenas o cluster Kind."
	@echo ""
	@echo "  \033[0;32mmake install-ingress-controller\033[0m"
	@echo "    Instala apenas o NGINX Ingress Controller."
	@echo ""
	@echo "  \033[0;32mmake kind-down\033[0m"
	@echo "    Deleta o cluster Kind."
	@echo ""
	@echo "  \033[0;32mmake clean\033[0m"
	@echo "    Remove os binários do Kind."
	@echo ""