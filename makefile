FRONTEND_DIR = ./web/default
FRONTEND_CLASSIC_DIR = ./web/classic
SWAG_FRONTEND_DIR ?= $(FRONTEND_CLASSIC_DIR)
SWAG_DIST_DIR = $(SWAG_FRONTEND_DIR)/dist/swag
# 不进 package.json:vendor 到 scripts/vendor/ 入库,保持 bun.lock 与上游一致、避免 rebase 冲突,并锁版本防 CDN 投毒
SCALAR_API_REFERENCE_VERSION ?= 1.40.9
SCALAR_VENDOR_FILE = scripts/vendor/scalar-api-reference-$(SCALAR_API_REFERENCE_VERSION).js
BACKEND_DIR = .
DEV_FRONTEND_DEFAULT_PORT ?= 5173
DEV_FRONTEND_CLASSIC_PORT ?= 5174
COMMIT_ID?=$(shell git describe --tags --match 'v*' --always --dirty)
VERSION?=$(COMMIT_ID)
BUILD_TIME?=$(shell date -u '+%Y%m%dT%H%M%SZ')
DOCKER_IMAGE_NAME ?= new-api
DOCKER_VOLCES_IMAGE_NAME ?=
DOCKER_VERSION?=latest
DOCKER_IMAGE=$(DOCKER_IMAGE_NAME):$(DOCKER_VERSION)
DOCKER_VOLCES_IMAGE=$(DOCKER_VOLCES_IMAGE_NAME):$(DOCKER_VERSION)
DEV_COMPOSE_FILE = docker-compose.dev.yml
DEV_POSTGRES_SERVICE = postgres
DEV_BACKEND_SERVICE = new-api
DEV_POSTGRES_DB = new-api
DEV_POSTGRES_USER = root
DEV_SQLITE_PATH ?= one-api.db

.PHONY: all swag docker-build docker-push docker-build-push mcp pr build-frontend build-frontend-classic build-all-frontends start-backend dev dev-api dev-api-rebuild dev-web dev-web-classic reset-setup

all: build-all-frontends start-backend

build-frontend:
	@echo "Building default frontend..."
	@cd ./web && bun install --frozen-lockfile
	@cd $(FRONTEND_DIR) && DISABLE_ESLINT_PLUGIN='true' VITE_REACT_APP_VERSION=$(cat ../../VERSION) bun run build

build-frontend-classic:
	@echo "Building classic frontend..."
	@cd ./web && bun install --frozen-lockfile
	@cd $(FRONTEND_CLASSIC_DIR) && VITE_REACT_APP_VERSION=$(cat ../../VERSION) bun run build
	@$(MAKE) swag

build-all-frontends: build-frontend build-frontend-classic

start-backend:
	@echo "Starting backend dev server..."
	@cd $(BACKEND_DIR) && go run main.go &

dev-api:
	@echo "Starting backend services (docker)..."
	@docker compose -f $(DEV_COMPOSE_FILE) up -d

dev-api-rebuild:
	@echo "Rebuilding and starting backend service (docker)..."
	@docker compose -f $(DEV_COMPOSE_FILE) up -d --build $(DEV_BACKEND_SERVICE)

dev-web:
	@echo "Starting both frontend dev servers..."
	@echo "Default frontend: http://localhost:$(DEV_FRONTEND_DEFAULT_PORT)"
	@echo "Classic frontend: http://localhost:$(DEV_FRONTEND_CLASSIC_PORT)"
	@cd ./web && bun install
	@(cd $(FRONTEND_DIR) && bun run dev -- --host 0.0.0.0 --port $(DEV_FRONTEND_DEFAULT_PORT)) & \
		default_pid=$$!; \
		(cd $(FRONTEND_CLASSIC_DIR) && bun run dev -- --host 0.0.0.0 --port $(DEV_FRONTEND_CLASSIC_PORT)) & \
		classic_pid=$$!; \
		trap 'kill $$default_pid $$classic_pid 2>/dev/null; wait $$default_pid $$classic_pid 2>/dev/null; exit 130' INT TERM; \
		while kill -0 $$default_pid 2>/dev/null && kill -0 $$classic_pid 2>/dev/null; do \
			sleep 1; \
		done; \
		if ! kill -0 $$default_pid 2>/dev/null; then \
			wait $$default_pid; \
			status=$$?; \
			kill $$classic_pid 2>/dev/null; \
			wait $$classic_pid 2>/dev/null; \
			exit $$status; \
		fi; \
		wait $$classic_pid; \
		status=$$?; \
		kill $$default_pid 2>/dev/null; \
		wait $$default_pid 2>/dev/null; \
		exit $$status

dev-web-classic:
	@echo "Starting classic frontend dev server..."
	@cd ./web && bun install
	@cd $(FRONTEND_CLASSIC_DIR) && bun run dev -- --host 0.0.0.0 --port $(DEV_FRONTEND_CLASSIC_PORT)

dev: dev-api dev-web

swag:
	@echo "Generating Swagger documentation to $(SWAG_DIST_DIR)..."
	# go install github.com/swaggo/swag/v2/cmd/swag@latest
	@cd $(BACKEND_DIR) && swag init --generatedTime --parseDependency --parseDepth 1 --ot=json -o=$(SWAG_DIST_DIR) --md=docs/api-descriptions --tags="!OIDC Provider,!Origin,!Forward,!Video,!TopUp"
	@sed 's/{{\.Version}}/$(VERSION)-$(BUILD_TIME)/g' $(SWAG_DIST_DIR)/swagger.json > $(SWAG_DIST_DIR)/swagger.json.tmp && mv $(SWAG_DIST_DIR)/swagger.json.tmp $(SWAG_DIST_DIR)/swagger.json
	@bunx swagger2openapi $(SWAG_DIST_DIR)/swagger.json -o $(SWAG_DIST_DIR)/openapi3.json
	@bun scripts/patch_openapi.js $(SWAG_DIST_DIR)/openapi3.json
	@bun scripts/patch_image_generation_openapi.js $(SWAG_DIST_DIR)/openapi3.json
	@bun scripts/patch_audio.js $(SWAG_DIST_DIR)/openapi3.json
	@echo "Vendoring Scalar API Reference standalone bundle (v$(SCALAR_API_REFERENCE_VERSION)) from $(SCALAR_VENDOR_FILE)..."
	@cp $(SCALAR_VENDOR_FILE) $(SWAG_DIST_DIR)/api-reference
	@echo "Mirroring swag/ to default theme dist..."
	@rm -rf $(FRONTEND_DIR)/dist/swag
	@mkdir -p $(FRONTEND_DIR)/dist
	@cp -R $(SWAG_DIST_DIR) $(FRONTEND_DIR)/dist/swag
	@echo $(VERSION) > VERSION
	@{ grep -v "^VERSION=" .env 2>/dev/null || true; echo "VERSION=$(VERSION)-$(BUILD_TIME)"; } > .env.tmp && mv .env.tmp .env
	@echo "Swagger documentation generated."

image: swag
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE) -t $(DOCKER_IMAGE_NAME):$(VERSION) -t $(DOCKER_VOLCES_IMAGE) -t $(DOCKER_VOLCES_IMAGE_NAME):$(VERSION) --build-arg VERSION=$(VERSION) $(BACKEND_DIR)
	@echo "Docker image built with tags $(DOCKER_IMAGE), $(DOCKER_IMAGE_NAME):$(VERSION), $(DOCKER_VOLCES_IMAGE) and $(DOCKER_VOLCES_IMAGE_NAME):$(VERSION)."
build: image

push:
	@echo "Pushing Docker image..."
	@docker push $(DOCKER_IMAGE)
	@docker push $(DOCKER_IMAGE_NAME):$(VERSION)
	@docker push $(DOCKER_VOLCES_IMAGE)
	@docker push $(DOCKER_VOLCES_IMAGE_NAME):$(VERSION)
	@echo "Docker image pushed to repository."

publish: build push
	@echo "Docker image published with tag $(DOCKER_IMAGE)."

publish-dev:
	@$(MAKE) publish DOCKER_VERSION=dev

mcp:
	@echo "start mcp server..."
	npx @agentdeskai/browser-tools-server@1.2.0
	@echo "mcp server started."

pr:
	@echo "Creating PR branch from origin/main..."
	@ORIGINAL_BRANCH=$$(git rev-parse --abbrev-ref HEAD); \
	echo "Current branch: $$ORIGINAL_BRANCH"; \
	echo "Fetching latest changes from origin..."; \
	git fetch origin; \
	echo "Getting commit ID..."; \
	if [ -n "$(commitid)" ]; then \
		COMMIT_ID="$(commitid)"; \
		echo "Using provided commit ID: $$COMMIT_ID"; \
	else \
		COMMIT_ID=$$(git rev-parse HEAD); \
		echo "Using current HEAD commit ID: $$COMMIT_ID"; \
	fi; \
	PR_BRANCH="pr/$$COMMIT_ID"; \
	echo "Creating PR branch: $$PR_BRANCH"; \
	git checkout -b $$PR_BRANCH origin/main; \
	echo "Cherry-picking commit: $$COMMIT_ID"; \
	git cherry-pick $$COMMIT_ID; \
	echo "Pushing to fork repository..."; \
	git push fork $$PR_BRANCH; \
	echo "PR branch $$PR_BRANCH has been pushed to fork repository"; \
	echo "You can now create a PR from: https://github.com/feitianbubu/new-api/compare/main...$$PR_BRANCH"; \
	echo "Switching back to original branch: $$ORIGINAL_BRANCH"; \
	git checkout $$ORIGINAL_BRANCH

reset-setup:
	@echo "Resetting local setup wizard state..."
	@if docker compose -f $(DEV_COMPOSE_FILE) ps --services --status running | grep -qx "$(DEV_POSTGRES_SERVICE)"; then \
		echo "Detected running docker dev PostgreSQL. Removing setup record and root users..."; \
		docker compose -f $(DEV_COMPOSE_FILE) exec -T $(DEV_POSTGRES_SERVICE) \
			psql -U $(DEV_POSTGRES_USER) -d $(DEV_POSTGRES_DB) \
			-c 'DELETE FROM setups;' \
			-c 'DELETE FROM users WHERE role = 100;' \
			-c "DELETE FROM options WHERE key IN ('SelfUseModeEnabled', 'DemoSiteEnabled');"; \
		echo "Restarting docker dev backend so setup status is recalculated..."; \
		docker compose -f $(DEV_COMPOSE_FILE) restart $(DEV_BACKEND_SERVICE); \
	elif db_path="$${SQLITE_PATH:-$(DEV_SQLITE_PATH)}"; db_path="$${db_path%%\?*}"; [ -f "$$db_path" ]; then \
		db_path="$${SQLITE_PATH:-$(DEV_SQLITE_PATH)}"; \
		db_path="$${db_path%%\?*}"; \
		echo "Detected local SQLite database: $$db_path"; \
		sqlite3 "$$db_path" \
			"DELETE FROM setups; DELETE FROM users WHERE role = 100; DELETE FROM options WHERE key IN ('SelfUseModeEnabled', 'DemoSiteEnabled');"; \
		echo "SQLite setup state reset. Restart the local backend process before testing the setup wizard."; \
	else \
		echo "No running docker dev PostgreSQL or local SQLite database found."; \
		echo "Start the dev stack with 'make dev-api', or set SQLITE_PATH/DEV_SQLITE_PATH to your local SQLite database."; \
		exit 1; \
	fi
