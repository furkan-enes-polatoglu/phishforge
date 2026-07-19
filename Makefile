.PHONY: help build run test lint vet migrate web docker up down logs seed-secrets

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN{FS=":.*?## "}{printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2}'

build: ## Build the Go binary
	go build -o out/phishforge ./cmd/phishforge

vet: ## go vet
	go vet ./...

test: ## Run Go tests
	go test ./...

lint: vet ## Alias for vet (add golangci-lint if desired)

web: ## Build the frontend
	cd web && npm install && npm run build

migrate: ## Run DB migrations locally (needs DATABASE_URL)
	go run ./cmd/phishforge migrate

run: ## Run the API locally (needs Postgres+Redis and a .env)
	go run ./cmd/phishforge api

docker: ## Build the Docker image
	docker build -t phishforge:local .

up: ## Start the full stack (docker compose)
	docker compose up -d --build

down: ## Stop the stack
	docker compose down

logs: ## Tail stack logs
	docker compose logs -f --tail=100

seed-secrets: ## Print strong random secrets for .env
	@echo "JWT_SECRET=$$(openssl rand -hex 32)"
	@echo "RID_SECRET=$$(openssl rand -hex 32)"
