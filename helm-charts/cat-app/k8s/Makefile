deploy:
	@echo "Deploying the application..."
	kubectl apply -f ./manifests/deployment.yml

service:
	@echo "Creating the service..."
	kubectl apply -f ./manifests/service.yml

ingress:
	@echo "Creating the ingress..."
	kubectl apply -f ./manifests/ingress.yml

all: deploy service ingress
	@echo "All resources have been deployed successfully!"

.PHONY: deploy service ingress all