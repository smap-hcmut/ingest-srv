export
BINARY=ingest-srv

LOCAL_CONFIG_FILE=./config/ingest-config.local.yaml

models:
	@echo "Generating models"
	@sqlboiler psql

models-local:
	@echo "Generating models with local sqlboiler config"
	@sqlboiler --config sqlboiler.local.toml psql

swagger:
	@echo "Generating swagger"
	@swag init -g cmd/server/main.go --parseVendor
	@echo "Fixing swagger docs (removing deprecated LeftDelim/RightDelim)..."
	@$(FIX_SWAGGER)

run: swagger
	@echo "Running the application"
	@go run cmd/server/main.go

run-local:
	@echo "Running with local docker-compose infrastructure config"
	@$(RUN_WITH_LOCAL_CONFIG) go run cmd/server/main.go

test: 
	@echo "Running tests..."
	@go test -mod=vendor -coverprofile=coverage.out -failfast -timeout 5m ./internal/...
	@grep -v 'mock_' coverage.out > c.out
	@go tool cover -func=c.out
	@echo "Coverage report generated in c.out"
	@rm -f *.out

build-docker-compose:
	@echo "make models first"
	@make models
	@echo "Building docker compose"
	docker compose up --build -d

# Docker build targets (using optimized Dockerfile)
docker-build:
	@echo "Building Docker image for local platform"
	@./scripts/build-api.sh local

docker-build-amd64:
	@echo "Building Docker image for AMD64"
	@./scripts/build-api.sh amd64

docker-build-multi:
	@echo "Building multi-platform Docker image (requires REGISTRY)"
	@./scripts/build-api.sh multi

docker-run:
	@echo "Building and running Docker container"
	@./scripts/build-api.sh run

docker-clean:
	@echo "Cleaning Docker images"
	@./scripts/build-api.sh clean

docker-push:
	@echo "Building and pushing to registry (requires REGISTRY)"
	@./scripts/build-api.sh push

# Show all available targets
help:
	@echo "Available targets:"
	@echo ""
	@echo "Development:"
	@echo "  models              - Generate SQLBoiler models"
	@echo "  swagger             - Generate Swagger documentation"
	@echo "  run                 - Run server locally (API + Consumer + Scheduler)"
	@echo "  run-local           - Run with local docker-compose config"
	@echo "  build-docker-compose - Build with docker-compose"
	@echo ""
	@echo "Docker:"
	@echo "  docker-build        - Build for local platform"
	@echo "  docker-build-amd64  - Build for AMD64 servers"
	@echo "  docker-build-multi  - Build multi-platform (requires REGISTRY env)"
	@echo "  docker-run          - Build and run container locally"
	@echo "  docker-clean        - Remove all Docker images"
	@echo "  docker-push         - Build and push to registry"
	@echo ""
	@echo "Examples:"
	@echo "  make docker-build"
	@echo "  make docker-run"
	@echo "  REGISTRY=docker.io/username make docker-push"

.PHONY: models swagger run run-local test build-docker-compose \
        docker-build docker-build-amd64 docker-build-multi \
        docker-run docker-clean docker-push
