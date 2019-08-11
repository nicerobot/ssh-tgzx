# A Self-Documenting Makefile: http://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.DEFAULT_GOAL := build

.PHONY: help
help:
	@grep -h -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

# Project variables
NAME ?= ssh-tgzx
BUILD_DIR ?= bin
VERSION ?= $(shell git describe --tags --exact-match 2>/dev/null || git symbolic-ref -q --short HEAD)
COMMIT_HASH ?= $(shell git rev-parse --short HEAD 2>/dev/null)
BUILD_DATE ?= $(shell date +%FT%T%z)
LDFLAGS = -X main.version=$(VERSION)

## Build

GO_FILES := $(shell find . -type f -name '*.go' ! -path './vendor/*' ! -path './garbage/*')

GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
DIST_BINARY = dist/$(NAME)-$(GOOS)-$(GOARCH)$(if $(filter windows,$(GOOS)),.exe,)
BIN_NAME = $(NAME)-$(GOOS)-$(GOARCH)$(if $(filter windows,$(GOOS)),.exe,)
BIN_TARGET = $(BUILD_DIR)/$(BIN_NAME)

$(BUILD_DIR):
	@mkdir -p $@

$(DIST_BINARY): $(GO_FILES)
	go tool goreleaser build --single-target --snapshot --clean

$(BIN_TARGET): $(BUILD_DIR) $(DIST_BINARY)
	cp $(DIST_BINARY) $@
	ln -sf $(BIN_NAME) $(BUILD_DIR)/$(NAME)

.PHONY: build
build: $(BIN_TARGET) ## Build binary for current platform only

.PHONY: build-all
build-all: ## Build binaries for all platforms
	go tool goreleaser build --snapshot --clean

.PHONY: release
release: ## Create a release with goreleaser
	go tool goreleaser release --clean

.PHONY: release-snapshot
release-snapshot: ## Create a snapshot release (no git tag required)
	go tool goreleaser release --snapshot --clean

.PHONY: clean
clean: ## Clean builds
	rm -rf $(BUILD_DIR)/$(NAME)*
	rm -rf dist/
	rm -rf *.test
	rm -rf *.out
	rm -rf coverage*
	rm -rf .coverage*

## Test

.PHONY: test
test: ## Run tests
	go tool gotestsum --format short -- ./...

.PHONY: test-verbose
test-verbose: ## Run tests with verbose output
	go tool gotestsum --format short-verbose -- -v ./...

.PHONY: test-integration
test-integration: ## Run integration tests
	go test -run ^TestIntegration ./...

## Code Quality

.PHONY: lint
lint: ## Run linter
	go tool golangci-lint run

.PHONY: vuln
vuln: ## Scan for vulnerabilities (including vendor)
	go tool govulncheck ./...

.PHONY: check
check: lint test vuln ## Run tests, linters, and vulnerability scan

## Code Generation

.PHONY: generate
generate: ## Generate code
	go generate ./...

## Utilities

.PHONY: tidy
tidy: ## Tidy and verify dependencies
	go mod tidy
	go mod verify

.PHONY: fmt
fmt: ## Format code
	go tool gofumpt -l -w .

.PHONY: vet
vet: ## Run go vet
	go vet ./...

# Variable outputting/exporting rules
var-%: ; @echo $($*)
varexport-%: ; @echo $*=$($*)
