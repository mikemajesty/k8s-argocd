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
