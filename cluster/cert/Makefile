# Define o nome do cluster Kind (ajuste se for diferente)
KIND_CLUSTER_NAME ?= argocd

# Define o nome do domínio
INGRESS_HOST = argocd.app.io

# Define a pasta onde os arquivos temporários serão criados
CERT_DIR = ./

# Cria o diretório de certificados se não existir
.PHONY: setup create

# ==============================================================================
# 1. GERAÇÃO DO CERTIFICADO SAN
# ==============================================================================
create:
	@echo "-> Criando diretório de certificados em $(CERT_DIR)..."
	@mkdir -p $(CERT_DIR)
	@echo "-> Gerando certificado autoassinado com Subject Alternative Name (SAN) para $(INGRESS_HOST)..."
	
	# 1. Cria o arquivo de configuração SAN CORRETO (san.cnf)
	@echo "[v3_req]" > $(CERT_DIR)/san.cnf
	@echo "subjectAltName = @alt_names" >> $(CERT_DIR)/san.cnf
	@echo "" >> $(CERT_DIR)/san.cnf
	@echo "[alt_names]" >> $(CERT_DIR)/san.cnf
	@echo "DNS.1 = $(INGRESS_HOST)" >> $(CERT_DIR)/san.cnf
	@echo "DNS.2 = localhost" >> $(CERT_DIR)/san.cnf
	@echo "IP.1 = 127.0.0.1" >> $(CERT_DIR)/san.cnf
	
	# 2. Gera a Chave e a Requisição (CSR)
	@openssl req -newkey rsa:2048 -nodes -keyout $(CERT_DIR)/tls.key \
	  -out $(CERT_DIR)/tls.csr -subj "/CN=$(INGRESS_HOST)/O=ArgoCD Ingress"
	
	# 3. Gera o Certificado Final (usando o SAN)
	@openssl x509 -req -days 365 -in $(CERT_DIR)/tls.csr -signkey $(CERT_DIR)/tls.key \
	  -out $(CERT_DIR)/tls.crt -extfile $(CERT_DIR)/san.cnf -extensions v3_req

# ==============================================================================
# 2. CRIAÇÃO DO SECRET NO KUBERNETES
# ==============================================================================
setup: create
	@echo "-> Criando ou atualizando o Secret TLS no Kubernetes..."
	# Remove o Secret antigo (para garantir que o novo SAN seja usado)
	@kubectl delete secret argocd-server-tls -n argocd --ignore-not-found
	
	# Cria o novo Secret TLS no namespace 'argocd'
	@kubectl create secret tls argocd-server-tls -n argocd \
	  --key $(CERT_DIR)/tls.key --cert $(CERT_DIR)/tls.crt

	@echo "-> Secret TLS 'argocd-server-tls' criado com sucesso!"
	# Não se esqueça de rodar o comando para instalar ou atualizar o Argo CD
	# helm upgrade --install argo-cd argo-helm/argo-cd ...