.PHONY: build run clean docker-build docker-run docker-compose

# Default Go build flags
GOFLAGS=-trimpath

# Output binary name
BINARY_NAME=slack-bot-api

# Docker image name
DOCKER_IMAGE=gen-alpha-slack-bot

build:
	@echo "Building $(BINARY_NAME)..."
	@go build $(GOFLAGS) -o $(BINARY_NAME) ./cmd/bot

run: build
	@echo "Running $(BINARY_NAME)..."
	@./$(BINARY_NAME)

clean:
	@echo "Cleaning up..."
	@rm -f $(BINARY_NAME)
	@go clean

test:
	@echo "Running tests..."
	@go test -v ./...

docker-build:
	@echo "Building Docker image..."
	@docker build -t $(DOCKER_IMAGE) .

docker-run: docker-build
	@echo "Running Docker container..."
	@docker run --env-file .env $(DOCKER_IMAGE)

docker-compose:
	@echo "Starting services with Docker Compose..."
	@docker-compose up --build

docker-compose-detach:
	@echo "Starting services with Docker Compose in detached mode..."
	@docker-compose up --build -d

setup:
	@echo "Setting up development environment..."
	@cp -n .env.example .env || true
	@go mod download
	@echo "Setup complete! Edit the .env file with your credentials."

help:
	@echo "Available commands:"
	@echo "  make build              - Build the application"
	@echo "  make run                - Build and run the application"
	@echo "  make clean              - Remove build artifacts"
	@echo "  make test               - Run tests"
	@echo "  make docker-build       - Build Docker image"
	@echo "  make docker-run         - Run Docker container"
	@echo "  make docker-compose     - Start services with Docker Compose"
	@echo "  make docker-compose-detach - Start services with Docker Compose in detached mode"
	@echo "  make setup              - Set up development environment"
	@echo "  make help               - Show this help message" 