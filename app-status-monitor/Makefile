# Variáveis
IMAGE_NAME = mikelima1989/app-status-monitor
TAG = latest

# Build e push da imagem Docker
deploy-docker:
	docker build -t $(IMAGE_NAME):$(TAG) .
	docker push $(IMAGE_NAME):$(TAG)

# Aplicar no Kubernetes (opcional)
deploy-k8s:
	kubectl apply -f templates/configmap.yaml
	kubectl apply -f templates/deployment.yaml
	kubectl apply -f templates/rbac.yaml

# Deploy completo
deploy-all: deploy-docker deploy-k8s

# Limpar
clean:
	docker rmi $(IMAGE_NAME):$(TAG) || true