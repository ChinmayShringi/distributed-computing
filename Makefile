# EdgeCLI Makefile

# Read version from VERSION file, fallback to dev
VERSION ?= $(shell cat VERSION 2>/dev/null || echo "dev")
COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "none")
BUILD_TIME ?= $(shell date -u '+%Y-%m-%d_%H:%M:%S')

LDFLAGS := -ldflags "-X github.com/edgecli/edgecli/cmd/edgecli/commands.Version=$(VERSION) \
	-X github.com/edgecli/edgecli/cmd/edgecli/commands.Commit=$(COMMIT)"

.PHONY: all build install clean test lint fmt help proto server server-quick build-web qaihub-example qaihub-download-models

all: build

## proto: Generate gRPC code from proto definitions
proto:
	protoc -I proto \
		--go_out=proto --go_opt=paths=source_relative \
		--go-grpc_out=proto --go-grpc_opt=paths=source_relative \
		proto/orchestrator.proto

## build: Build the CLI binary
build:
	go build $(LDFLAGS) -o edgecli ./cmd/edgecli

## install: Install the CLI to GOPATH/bin
install:
	go install $(LDFLAGS) ./cmd/edgecli

## clean: Remove build artifacts
clean:
	rm -f edgecli
	rm -rf dist/

## test: Run tests
test:
	go test -v ./...

## lint: Run linter
lint:
	golangci-lint run ./...

## fmt: Format code
fmt:
	go fmt ./...
	goimports -w .

## tidy: Tidy go modules
tidy:
	go mod tidy

## deps: Download dependencies
deps:
	go mod download

## build-release: Build the CLI binary for release
build-release:
	go build -ldflags "-X github.com/edgecli/edgecli/cmd/edgecli/commands.Version=$(VERSION) \
		-X github.com/edgecli/edgecli/cmd/edgecli/commands.Commit=$(COMMIT)" \
		-o edgecli ./cmd/edgecli

## build-all: Build for all platforms
build-all:
	@mkdir -p dist
	GOOS=darwin GOARCH=arm64 go build -ldflags "-X github.com/edgecli/edgecli/cmd/edgecli/commands.Version=$(VERSION) \
		-X github.com/edgecli/edgecli/cmd/edgecli/commands.Commit=$(COMMIT)" \
		-o dist/edgecli-darwin-arm64 ./cmd/edgecli
	GOOS=darwin GOARCH=amd64 go build -ldflags "-X github.com/edgecli/edgecli/cmd/edgecli/commands.Version=$(VERSION) \
		-X github.com/edgecli/edgecli/cmd/edgecli/commands.Commit=$(COMMIT)" \
		-o dist/edgecli-darwin-amd64 ./cmd/edgecli
	GOOS=linux GOARCH=amd64 go build -ldflags "-X github.com/edgecli/edgecli/cmd/edgecli/commands.Version=$(VERSION) \
		-X github.com/edgecli/edgecli/cmd/edgecli/commands.Commit=$(COMMIT)" \
		-o dist/edgecli-linux-amd64 ./cmd/edgecli
	GOOS=linux GOARCH=arm64 go build -ldflags "-X github.com/edgecli/edgecli/cmd/edgecli/commands.Version=$(VERSION) \
		-X github.com/edgecli/edgecli/cmd/edgecli/commands.Commit=$(COMMIT)" \
		-o dist/edgecli-linux-arm64 ./cmd/edgecli
	GOOS=windows GOARCH=amd64 go build -ldflags "-X github.com/edgecli/edgecli/cmd/edgecli/commands.Version=$(VERSION) \
		-X github.com/edgecli/edgecli/cmd/edgecli/commands.Commit=$(COMMIT)" \
		-o dist/edgecli-windows-amd64.exe ./cmd/edgecli

## run: Build and run with arguments
run: build
	./edgecli $(ARGS)

## bump-version: Increment version by 0.1 for next deploy
bump-version:
	@current=$$(cat VERSION); \
	new=$$(awk "BEGIN {printf \"%.1f\", $$current + 0.1}"); \
	echo "$$new" > VERSION; \
	echo "Version bumped: $$current -> $$new"

## show-version: Show current version
show-version:
	@echo "Current version: $$(cat VERSION)"

## build-web: Build the React web UI and copy to cmd/server/webui
build-web:
	@echo "Building edge_web..."
	@cd edge_web && npm run build
	@echo "Copying build to cmd/server/webui..."
	@rm -rf cmd/server/webui
	@cp -r edge_web/dist cmd/server/webui
	@echo "Web UI build complete"

## server: Build web UI and run the unified server (gRPC :50051 + HTTP Web UI :8080)
## Loads .env file if present for chat configuration
server: build-web
	@if [ -f .env ]; then set -a && . ./.env && set +a; fi && go run ./cmd/server

## server-quick: Run the server without rebuilding web UI (use existing webui/)
server-quick:
	@if [ -f .env ]; then set -a && . ./.env && set +a; fi && go run ./cmd/server

## qaihub-example: Run QAI Hub compile example (Windows only)
qaihub-example:
ifeq ($(OS),Windows_NT)
	powershell -ExecutionPolicy Bypass -File scripts/windows/qaihub_compile_example.ps1
else
	@echo "QAI Hub compile example requires Windows."
	@echo "Run the Python helper directly: python scripts/qaihub_download_job.py --job <id> --out <dir>"
endif

## qaihub-download-models: Download/export models from qai_hub_models (Windows only)
qaihub-download-models:
ifeq ($(OS),Windows_NT)
	powershell -ExecutionPolicy Bypass -File scripts/windows/qaihub_download_models.ps1
else
	@echo "QAI Hub model download requires Windows with Snapdragon."
	@echo "Run directly: python scripts/qaihub_download_models.py --help"
endif

## build-server: Build the unified server binary (gRPC + HTTP)
build-server:
	go build -o dist/edgecli-server ./cmd/server

## help: Show this help
help:
	@echo "EdgeCLI Build Commands:"
	@echo ""
	@sed -n 's/^##//p' $(MAKEFILE_LIST) | column -t -s ':' | sed -e 's/^/ /'
