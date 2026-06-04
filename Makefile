.DEFAULT_GOAL := help

# Local-dev overlay: loaded only for dev/test targets so cluster targets
# (deploy/teardown) run against the host environment unchanged.
DEV_TARGETS := dev backend worker worker-v2 worker-v3 \
               app-up app-worker-v2 app-worker-v3 app-down app-logs \
               infra-up infra-down infra-logs \
               test check
GOALS := $(or $(MAKECMDGOALS),$(.DEFAULT_GOAL))
ifneq (,$(filter $(DEV_TARGETS),$(GOALS)))
ifneq (,$(wildcard .env.local))
include .env.local
export
endif
endif

# Connection settings (override via .env.local or the environment).
TEMPORAL_ADDRESS ?= localhost:7233
TEMPORAL_NAMESPACE ?= default
DEPLOYMENT_NAME ?= default.pizza
PIZZA_TASK_QUEUE ?= pizza

WORKER_BIN := ./bin/worker
BACKEND_BIN := ./bin/backend

# Connection env shared by the host worker targets.
WORKER_ENV = TEMPORAL_ADDRESS=$(TEMPORAL_ADDRESS) TEMPORAL_NAMESPACE=$(TEMPORAL_NAMESPACE) \
             TEMPORAL_DEPLOYMENT_NAME=$(DEPLOYMENT_NAME) PIZZA_TASK_QUEUE=$(PIZZA_TASK_QUEUE)

##@ Infra

.PHONY: infra-up
infra-up: ## Start the Temporal dev server (waits until healthy)
	docker compose up -d --wait temporal

.PHONY: infra-down
infra-down: ## Stop the Temporal dev server
	docker compose stop temporal

.PHONY: infra-logs
infra-logs: ## Follow Temporal dev server logs
	docker compose logs -f temporal

##@ App (host, hot reload)

.PHONY: dev
dev: infra-up ## Start Temporal, then run backend + worker v1 on the host
	@$(MAKE) -j backend worker

.PHONY: backend
backend: ## Run the backend on :8080 with hot reload (requires Air)
	TEMPORAL_ADDRESS=$(TEMPORAL_ADDRESS) TEMPORAL_NAMESPACE=$(TEMPORAL_NAMESPACE) \
		PIZZA_DEPLOYMENT_NAME=$(DEPLOYMENT_NAME) PIZZA_TASK_QUEUE=$(PIZZA_TASK_QUEUE) \
		air -c .air.toml

.PHONY: worker
worker: ## Run the v1 worker on the host
	$(WORKER_ENV) PIZZA_VERSION=v1 TEMPORAL_WORKER_BUILD_ID=v1-local go run ./cmd/worker

.PHONY: worker-v2
worker-v2: ## Run the v2 worker on the host (demo: ship v2)
	$(WORKER_ENV) PIZZA_VERSION=v2 TEMPORAL_WORKER_BUILD_ID=v2-local go run ./cmd/worker

.PHONY: worker-v3
worker-v3: ## Run the v3 worker on the host (demo: ship v3)
	$(WORKER_ENV) PIZZA_VERSION=v3 TEMPORAL_WORKER_BUILD_ID=v3-local go run ./cmd/worker

##@ Stack (Docker)

.PHONY: app-up
app-up: ## Bring up the full stack in Docker (Temporal + backend + worker v1)
	docker compose up -d --build

.PHONY: app-worker-v2
app-worker-v2: ## Start the v2 worker in the Docker stack (demo: ship v2)
	docker compose --profile v2 up -d --build

.PHONY: app-worker-v3
app-worker-v3: ## Start the v3 worker in the Docker stack (demo: ship v3)
	docker compose --profile v3 up -d --build

.PHONY: app-down
app-down: ## Tear down the Docker stack (removes containers and network)
	docker compose down

.PHONY: app-logs
app-logs: ## Follow logs from every stack container
	docker compose logs -f

##@ Quality

.PHONY: test
test: ## Run tests (race + shuffle)
	go test -race -shuffle=on ./...

.PHONY: lint
lint: ## Run golangci-lint (requires golangci-lint v2)
	golangci-lint run

.PHONY: format
format: ## Format code (gofumpt + goimports)
	go run mvdan.cc/gofumpt@latest -l -w .
	go run golang.org/x/tools/cmd/goimports@latest -w .

.PHONY: tidy
tidy: ## Tidy and verify module dependencies
	go mod tidy
	go mod verify

.PHONY: check
check: lint test ## Run all checks (lint + test)

##@ Build

.PHONY: build
build: ## Build the worker and backend binaries into ./bin
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(WORKER_BIN) ./cmd/worker
	CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o $(BACKEND_BIN) ./cmd/backend

##@ Deploy

.PHONY: deploy
deploy: ## Deploy to the temporal-k8s cluster
	kubectl apply -k k8s/

.PHONY: teardown
teardown: ## Remove the demo from the cluster
	kubectl delete -k k8s/

##@ Helpers

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "Usage: make \033[36m<target>\033[0m\n"} \
		/^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } \
		/^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) }' $(firstword $(MAKEFILE_LIST))
