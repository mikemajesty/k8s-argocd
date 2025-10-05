# Variáveis
IMAGE_NAME = mikelima1989/app-status-monitor
TAG = latest
NAMESPACE = argocd
DEPLOYMENT_NAME = status-monitor-app-status-monitor

# Build e push da imagem Docker
deploy-docker:
	@echo "🐳 Building and pushing Docker image..."
	docker build -t $(IMAGE_NAME):$(TAG) .
	docker push $(IMAGE_NAME):$(TAG)
	@echo "✅ Docker image pushed successfully!"

# Atualizar no Kubernetes (ROLLOUT RESTART)
deploy-k8s:
	@echo "🚀 Updating Kubernetes deployment..."
	kubectl rollout restart deployment/$(DEPLOYMENT_NAME) -n $(NAMESPACE)
	@echo "⏳ Waiting for rollout to complete..."
	kubectl rollout status deployment/$(DEPLOYMENT_NAME) -n $(NAMESPACE) --timeout=300s
	@echo "✅ Kubernetes deployment updated successfully!"

# Deploy completo (Docker + Kubernetes)
deploy: deploy-docker deploy-k8s
	@echo "🎉 Full deployment completed!"

# Deploy usando Helm
deploy-helm:
	@echo "📦 Deploying with Helm..."
	helm upgrade $(DEPLOYMENT_NAME) ./app-status-monitor -n $(NAMESPACE) --install
	@echo "✅ Helm deployment completed!"

# Deploy completo com Helm
deploy-full: deploy-docker deploy-helm
	@echo "🎉 Full Helm deployment completed!"

# Ver status
status:
	@echo "📊 Checking deployment status..."
	@echo "--- Pods ---"
	kubectl get pods -n $(NAMESPACE) -l "app.kubernetes.io/name=app-status-monitor"
	@echo "--- Deployment ---"
	kubectl get deployment/$(DEPLOYMENT_NAME) -n $(NAMESPACE)
	@echo "--- Services ---"
	kubectl get services -n $(NAMESPACE) -l "app.kubernetes.io/name=app-status-monitor"

# Ver logs
logs:
	@echo "📋 Showing logs..."
	kubectl logs -f deployment/$(DEPLOYMENT_NAME) -n $(NAMESPACE)

# Limpar
clean:
	@echo "🧹 Cleaning up..."
	docker rmi $(IMAGE_NAME):$(TAG) || true
	@echo "✅ Cleanup completed!"

# Help
help:
	@echo "📖 Available commands:"
	@echo "  make deploy-docker    - Build and push Docker image only"
	@echo "  make deploy-k8s       - Update Kubernetes deployment (rollout restart)"
	@echo "  make deploy           - Full deploy (Docker + K8s rollout)"
	@echo "  make deploy-helm      - Deploy using Helm"
	@echo "  make deploy-full      - Full deploy with Helm"
	@echo "  make status           - Check deployment status"
	@echo "  make logs             - Show application logs"
	@echo "  make clean            - Clean up Docker image"