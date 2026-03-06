export
BINARY=engine

LOCAL_CONFIG_FILE=./config/ingest-config.local.yaml

ifeq ($(OS),Windows_NT)
RUN_WITH_LOCAL_CONFIG=set "INGEST_CONFIG_FILE=$(LOCAL_CONFIG_FILE)" &&
else
RUN_WITH_LOCAL_CONFIG=INGEST_CONFIG_FILE=$(LOCAL_CONFIG_FILE)
endif

models:
	@echo "Generating models"
	@sqlboiler psql

models-local:
	@echo "Generating models with local sqlboiler config"
	@sqlboiler --config sqlboiler.local.toml psql

swagger:
	@echo "Generating swagger"
	@swag init -g cmd/api/main.go
	@echo "Fixing swagger docs (removing deprecated LeftDelim/RightDelim)..."
	@sed -i '' '/LeftDelim:/d' docs/docs.go
	@sed -i '' '/RightDelim:/d' docs/docs.go

run-api:
# 	@echo "Generating swagger"
# 	@swag init -g cmd/api/main.go
# 	@sed -i '' '/LeftDelim:/d' docs/docs.go
# 	@sed -i '' '/RightDelim:/d' docs/docs.go
# 	@echo "Running the application"
	@go run cmd/api/main.go

run-api-local:
	@echo "Running API with local docker-compose infrastructure config"
	@$(RUN_WITH_LOCAL_CONFIG) go run cmd/api/main.go

run-consumer:
	@echo "Running the consumer"
	@go run cmd/consumer/main.go

run-consumer-local:
	@echo "Running consumer with local docker-compose infrastructure config"
	@$(RUN_WITH_LOCAL_CONFIG) go run cmd/consumer/main.go

run-scheduler-local:
	@echo "Running scheduler with local docker-compose infrastructure config"
	@$(RUN_WITH_LOCAL_CONFIG) go run cmd/scheduler/main.go

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

# Consumer Docker build targets (using optimized Dockerfile)
consumer-build:
	@echo "Building Consumer Docker image for local platform"
	@./scripts/build-consumer.sh local

consumer-build-amd64:
	@echo "Building Consumer Docker image for AMD64"
	@./scripts/build-consumer.sh amd64

consumer-build-multi:
	@echo "Building multi-platform Consumer Docker image (requires REGISTRY)"
	@./scripts/build-consumer.sh multi

consumer-run:
	@echo "Building and running Consumer Docker container"
	@./scripts/build-consumer.sh run

consumer-clean:
	@echo "Cleaning Consumer Docker images"
	@./scripts/build-consumer.sh clean

consumer-push:
	@echo "Building and pushing Consumer to registry (requires REGISTRY)"
	@./scripts/build-consumer.sh push

# Show all available targets
help:
	@echo "Available targets:"
	@echo ""
	@echo "Development:"
	@echo "  models              - Generate SQLBoiler models"
	@echo "  swagger             - Generate Swagger documentation"
	@echo "  run-api             - Run API server locally"
	@echo "  run-consumer        - Run consumer locally"
	@echo "  build-docker-compose - Build with docker-compose"
	@echo ""
	@echo "Docker - API Server:"
	@echo "  docker-build        - Build for local platform"
	@echo "  docker-build-amd64  - Build for AMD64 servers"
	@echo "  docker-build-multi  - Build multi-platform (requires REGISTRY env)"
	@echo "  docker-run          - Build and run container locally"
	@echo "  docker-clean        - Remove all Docker images"
	@echo "  docker-push         - Build and push to registry"
	@echo ""
	@echo "Docker - Consumer Service:"
	@echo "  consumer-build      - Build consumer for local platform"
	@echo "  consumer-build-amd64 - Build consumer for AMD64 servers"
	@echo "  consumer-build-multi - Build multi-platform (requires REGISTRY env)"
	@echo "  consumer-run        - Build and run consumer container locally"
	@echo "  consumer-clean      - Remove all consumer Docker images"
	@echo "  consumer-push       - Build and push consumer to registry"
	@echo ""
	@echo "Examples:"
	@echo "  make docker-build"
	@echo "  make docker-run"
	@echo "  make consumer-build"
	@echo "  make consumer-run"
	@echo "  REGISTRY=docker.io/username make docker-push"
	@echo "  REGISTRY=docker.io/username make consumer-push"

.PHONY: models swagger run-api run-consumer build-docker-compose \
        docker-build docker-build-amd64 docker-build-multi \
        docker-run docker-clean docker-push \
        consumer-build consumer-build-amd64 consumer-build-multi \
        consumer-run consumer-clean consumer-push \
