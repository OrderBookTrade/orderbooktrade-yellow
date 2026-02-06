.PHONY: build run test clean deps

# Binary name
BINARY=orderbook-server

# Build the server
build:
	go build -o bin/$(BINARY) ./cmd/server

# Run the server
run:
	go run ./cmd/server

# Run with hot reload (requires air: go install github.com/cosmtrek/air@latest)
dev:
	air

# Run tests
test:
	go test -v ./...

# Run tests with coverage
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Clean build artifacts
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# Install dependencies
deps:
	go mod tidy
	go mod download

# Format code
fmt:
	go fmt ./...

# Run linter (requires golangci-lint)
lint:
	golangci-lint run

# Build for production
build-prod:
	CGO_ENABLED=0 GOOS=linux go build -ldflags="-w -s" -o bin/$(BINARY) ./cmd/server

# Docker build
docker-build:
	docker build -t orderbook-backend .

# Docker run
docker-run:
	docker run -p 8080:8080 --env-file .env orderbook-backend
