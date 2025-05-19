FRONTEND_DIR = ./web
BACKEND_DIR = .
COMMIT_ID?=$(git describe --tags --always --dirty)
VERSION?=${COMMIT_ID}-$(shell date +%Y%m%d%H%M)
DOCKER_VERSION=latest
DOCKER_IMAGE=skynono/clinx:${DOCKER_VERSION}

.PHONY: all build-frontend start-backend swag docker-build

all: build-frontend start-backend

build-frontend:
	@echo "Building frontend..."
	@cd $(FRONTEND_DIR) && npm install && DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat VERSION) npm run build

start-backend:
	@echo "Starting backend dev server..."
	@cd $(BACKEND_DIR) && go run main.go &

swag:
	@echo "Generating Swagger documentation..."
	@cd $(BACKEND_DIR) && swag init --generatedTime --parseDependency --ot=json -o=web/dist/swag
	@sed -i 's/"version": ".*"/"version": "$(VERSION)"/' web/dist/swag/swagger.json
	@echo $(VERSION) > VERSION
	@echo "Swagger documentation generated."

docker-build:
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE) --build-arg VERSION=$(VERSION) $(BACKEND_DIR)
	@echo "Docker image built with tag $(DOCKER_IMAGE)."

docker-push:
	@echo "Pushing Docker image..."
	@docker push $(DOCKER_IMAGE)
	@echo "Docker image pushed to repository."

docker-build-push: swag docker-build docker-push
	@echo "Docker image built and pushed with tag $(DOCKER_IMAGE)."

mcp:
	@echo "start mcp server..."
	npx @agentdeskai/browser-tools-server@latest
	@echo "mcp server started."