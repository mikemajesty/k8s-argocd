# Variáveis de Configuração
DOCKER_USER := mikelima1989
APP_NAME := cat-app
VERSION_TYPE := patch # ou minor, ou major
DOCKERFILE_PATH := apps/cat/Dockerfile

# 🚀 CORREÇÃO 1: Define a variável VERSION usando a sintaxe do Make.
# O uso de 'shell' define a variável VERSION para ser usada em todos os targets.
# O 'tr -d \"' remove as aspas duplas que o 'npm pkg get version' adiciona.
VERSION := $(shell npm pkg get version | tr -d \")

.PHONY: release build_docker push_docker deploy clean

# 1. Altera a versão no package.json
# O npm version imprime a nova versão (ex: v1.0.1)
release:
	@echo "Incrementando a versão..."
	# Use --no-git-tag-version para que o Make controle o fluxo
	@npm version $(VERSION_TYPE) --no-git-tag-version
	
# 2. Constrói a imagem Docker com a nova tag
# CORREÇÃO 2: move a exibição da versão para um local onde a variável já está definida.
build_docker:
	@echo "Versão atualizada: $(VERSION)"
	@echo "Construindo imagem Docker com tag $(DOCKER_USER)/$(APP_NAME):$(VERSION)..."
	docker build -f $(DOCKERFILE_PATH) -t $(DOCKER_USER)/$(APP_NAME):$(VERSION) .
	docker tag $(DOCKER_USER)/$(APP_NAME):$(VERSION) $(DOCKER_USER)/$(APP_NAME):latest

# 3. Faz o push para o Docker Hub
push_docker: build_docker
	@echo "Fazendo push da imagem $(DOCKER_USER)/$(APP_NAME):$(VERSION) e :latest..."
	docker push $(DOCKER_USER)/$(APP_NAME):$(VERSION)
	docker push $(DOCKER_USER)/$(APP_NAME):latest

push_tag:
	@echo "Fazendo commit e push para o repositório Git..."
	git add -A
	git commit -m "Bump version to $(VERSION)"
	git tag v$(VERSION)-cat-app
	git push origin main --tags

push:
	git add -A
	git commit -m "Atualização automática"
	git push origin main

# Target principal para executar tudo
deploy: release build_docker push_docker push_tag
	@echo "🚀 Deploy concluído com a versão $(VERSION)!"