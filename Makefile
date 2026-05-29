.PHONY: all dev build lint test clean help

# Default target
all: build

## help: Show this help message
help:
	@echo "NexTunnel Build System"
	@echo ""
	@echo "Usage: make <target>"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'

## dev: Start Wails development server
dev:
	cd desktop && wails dev

## build: Build the Wails desktop application
build:
	cd desktop && wails build

## build-server: Build all server binaries
build-server:
	cd server && go build -o ../build/control-plane ./cmd/control-plane
	cd server && go build -o ../build/relay-server ./cmd/relay
	cd server && go build -o ../build/nat-detector ./cmd/nat-detector

## lint: Run all linters
lint: lint-go lint-frontend

## lint-go: Run Go linter
lint-go:
	cd desktop && golangci-lint run ./...
	cd server && golangci-lint run ./...

## lint-frontend: Run ESLint on frontend
lint-frontend:
	cd desktop/frontend && npm run lint

## test: Run all tests
test: test-go test-frontend

## test-go: Run Go tests
test-go:
	cd desktop && go test ./...
	cd server && go test ./...
	cd pkg && go test ./...

## test-frontend: Run frontend tests
test-frontend:
	cd desktop/frontend && npm run test

## clean: Remove build artifacts
clean:
	rm -rf desktop/build/bin/
	rm -rf desktop/frontend/dist/
	rm -rf build/

## install-deps: Install all dependencies
install-deps:
	cd desktop && go mod tidy
	cd server && go mod tidy
	cd pkg && go mod tidy
	cd desktop/frontend && npm install
