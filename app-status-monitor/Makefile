# Variáveis
IMAGE_NAME = mikelima1989/app-status-monitor
TAG = latest
NAMESPACE = argocd
DEPLOYMENT_NAME = status-monitor-app-status-monitor
ARGO_APP_NAME = app-status-monitor  # Nome do Application no Argo CD

# Build e push da imagem Docker
deploy-docker:
	@echo "🐳 Building and pushing Docker image..."
	docker build -t $(IMAGE_NAME):$(TAG) .
	docker push $(IMAGE_NAME):$(TAG)
	@echo "✅ Docker image pushed successfully!"

# Atualizar Kubernetes (para quando o Argo CD controla o Deployment)
deploy-k8s:
	@echo "🚀 Triggering Kubernetes update via rollout..."
	kubectl rollout restart deployment/$(DEPLOYMENT_NAME) -n $(NAMESPACE)
	@echo "⏳ Waiting for rollout..."
	kubectl rollout status deployment/$(DEPLOYMENT_NAME) -n $(NAMESPACE) --timeout=300s
	@echo "✅ Kubernetes deployment updated!"

# Deploy completo (APENAS atualiza a imagem do monitor)
deploy: deploy-docker deploy-k8s
	@echo "🎉 Monitor image updated! Argo CD maintains the configuration."

# Sincronizar Argo CD (se você mudou os manifests/helm chart)
sync-argo:
	@echo "🔄 Syncing Argo CD Application..."
	argocd app sync $(ARGO_APP_NAME)
	argocd app wait $(ARGO_APP_NAME) --health
	@echo "✅ Argo CD Application synced!"

# Deploy completo + Argo CD sync
deploy-full: deploy-docker deploy-k8s sync-argo
	@echo "🎉 Full deployment with Argo CD sync completed!"

# Status do Argo CD
status-argo:
	@echo "📊 Argo CD Application status..."
	argocd app get $(ARGO_APP_NAME)

# Status completo
status: status-argo
	@echo "--- Kubernetes Resources ---"
	kubectl get pods,deployments -n $(NAMESPACE) -l "app.kubernetes.io/name=app-status-monitor"