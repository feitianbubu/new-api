FRONTEND_DIR = ./web
BACKEND_DIR = .
COMMIT_ID?=$(shell git rev-parse --short HEAD)
VERSION?=v0.0.1-${COMMIT_ID}

.PHONY: all build-frontend start-backend swag

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