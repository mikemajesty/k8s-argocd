# Variáveis
IMAGE_NAME = mikelima1989/app-status-monitor
TAG = latest
NAMESPACE = argocd
DEPLOYMENT_NAME = status-monitor-app-status-monitor
ARGO_APP_NAME = app-status-monitor
ARGO_SERVER = argocd.app.io  # Ajuste conforme seu ambiente
ARGO_CD_CLI = ../bin/argocd  # ← CAMINHO CORRETO para seu binário

# Build e push da imagem Docker
deploy-docker:
	@echo "🐳 Building and pushing Docker image..."
	docker build -t $(IMAGE_NAME):$(TAG) .
	docker push $(IMAGE_NAME):$(TAG)
	@echo "✅ Docker image pushed successfully!"

# Atualizar Kubernetes
deploy-k8s:
	@echo "🚀 Triggering Kubernetes update via rollout..."
	kubectl rollout restart deployment/$(DEPLOYMENT_NAME) -n $(NAMESPACE)
	@echo "⏳ Waiting for rollout..."
	kubectl rollout status deployment/$(DEPLOYMENT_NAME) -n $(NAMESPACE) --timeout=300s
	@echo "✅ Kubernetes deployment updated!"

# Deploy completo (APENAS atualiza a imagem do monitor)
deploy: deploy-docker deploy-k8s
	@echo "🎉 Monitor image updated! Argo CD maintains the configuration."

# Sincronizar Argo CD (OPCIONAL - só se mudou os manifests)
sync-argo:
	@echo "🔄 Syncing Argo CD Application..."
	@if ! test -f "$(ARGO_CD_CLI)"; then \
		echo "❌ Argo CD CLI not found at $(ARGO_CD_CLI)"; \
		echo "💡 Download from: https://github.com/argoproj/argo-cd/releases"; \
		exit 1; \
	fi
	@if ! $(ARGO_CD_CLI) app get $(ARGO_APP_NAME) &> /dev/null; then \
		echo "⚠️  Argo CD not configured. Run: $(ARGO_CD_CLI) login $(ARGO_SERVER)"; \
		echo "📝 Application will sync automatically via GitOps"; \
		exit 0; \
	fi
	$(ARGO_CD_CLI) app sync $(ARGO_APP_NAME)
	$(ARGO_CD_CLI) app wait $(ARGO_APP_NAME) --health
	@echo "✅ Argo CD Application synced!"

# Status do Argo CD (usando seu binário)
status-argo:
	@echo "📊 Argo CD Application status..."
	@if test -f "$(ARGO_CD_CLI)"; then \
		$(ARGO_CD_CLI) app get $(ARGO_APP_NAME); \
	else \
		echo "ℹ️  Argo CD CLI not available at $(ARGO_CD_CLI)"; \
	fi

# Status simplificado
status:
	@echo "📊 Checking status..."
	@echo "--- Pods ---"
	kubectl get pods -n $(NAMESPACE) -l "app.kubernetes.io/name=app-status-monitor"
	@echo "--- Deployment ---" 
	kubectl get deployment/$(DEPLOYMENT_NAME) -n $(NAMESPACE)
	@echo "--- Argo CD Status ---"
	@make status-argo || true

# Ver logs
logs:
	kubectl logs -f deployment/$(DEPLOYMENT_NAME) -n $(NAMESPACE)

# Limpar
clean:
	docker rmi $(IMAGE_NAME):$(TAG) || true

# Help
help:
	@echo "📖 Available commands:"
	@echo "  make deploy        - Build, push image and update deployment"
	@echo "  make deploy-docker - Build and push Docker image only"
	@echo "  make deploy-k8s    - Update Kubernetes deployment only"
	@echo "  make status        - Check deployment status + Argo CD"
	@echo "  make logs          - Show application logs"
	@echo "  make sync-argo     - Sync Argo CD application (usa ../bin/argocd)"
	@echo "  make clean         - Clean up Docker image"