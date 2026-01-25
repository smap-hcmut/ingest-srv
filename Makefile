include .env
export
BINARY=engine

run-consumer:
	@echo "Running the consumer"
	@go run cmd/consumer/main.go

build-docker-compose:
	@echo "make models first"
	@make models
	@echo "Building docker compose"
	docker compose up --build -d