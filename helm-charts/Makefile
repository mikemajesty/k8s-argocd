.PHONY: setup clean help install-helm

# ==============================================================================
# Variáveis de Configuração
# ==============================================================================

HELM_VERSION ?= v3.13.1
BIN_DIR ?= ./bin

# ==============================================================================
# Instalação do Helm
# ==============================================================================

# Alvo para instalar o Helm
install-helm:
	@echo "-> Verificando a instalação do Helm..."
	@if [ ! -f $(BIN_DIR)/helm ]; then \
		echo "-> Helm não encontrado. Baixando a versão $(HELM_VERSION)..."; \
		mkdir -p $(BIN_DIR); \
		HELM_PLATFORM=$$(go env GOOS)-amd64; \
		curl -sSL "https://get.helm.sh/helm-$(HELM_VERSION)-$${HELM_PLATFORM}.tar.gz" | tar xz; \
		mv $${HELM_PLATFORM}/helm $(BIN_DIR)/helm; \
		rm -rf $${HELM_PLATFORM}; \
	else \
		echo "-> Helm já está instalado."; \
	fi

# ==============================================================================
# Fluxo de Trabalho
# ==============================================================================

setup: install-helm
	@echo "-> Instalação do Helm concluída. Use 'make clean' para remover os binários."

# ==============================================================================
# Alvos Utilitários e de Limpeza
# ==============================================================================

clean:
	@echo "-> Removendo o diretório de binários..."
	rm -rf $(BIN_DIR)

# ==============================================================================
# Ajuda
# ==============================================================================

help:
	@echo ""
	@echo "Bem-vindo ao gerenciador de binários do Helm!"
	@echo ""
	@echo "Comandos disponíveis:"
	@echo ""
	@echo "  \033[0;32mmake setup\033[0m"
	@echo "    Instala o binário do Helm no diretório ./bin do projeto."
	@echo ""
	@echo "  \033[0;32mmake clean\033[0m"
	@echo "    Remove os binários do Helm e o diretório ./bin."
	@echo ""
	@echo "  \033[0;32mmake help\033[0m"
	@echo "    Exibe esta mensagem de ajuda."
	@echo ""