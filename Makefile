minikube:
	@echo "Starting the application..."
	minikube start

start: kill-port-3000
	@echo "Starting the application..."
	@node apps/cat/src/app.js

kill-port-3000:
	@echo "Killing any process on port 3000..."
	@lsof -ti:3000 | xargs -r kill
	@echo "Port 3000 is now free."

docker: docker-run
	@echo "Building and starting the Docker container..."
	docker build -t cat-app .

docker-run:
	@echo "Running the Docker container..."
	docker run -p 3000:3000 cat-app

docker-stop:
	@echo "Stopping the Docker container..."
	docker stop cat-app

docker-deploy:
	@echo "Deploying the Docker container to Minikube..."
	minikube image load cat-app:latest