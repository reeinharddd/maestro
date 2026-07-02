# Binary name
BINARY=maestro
# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOFMT=$(GOCMD) fmt
COVERFILE=coverage.out

# Build for current platform
build:
	$(GOBUILD) -o $(BINARY) -ldflags="-s -w" ./cmd/maestro/

# Build for all platforms
build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64

build-linux-amd64:
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o build/$(BINARY)-linux-amd64 -ldflags="-s -w" ./cmd/maestro/

build-linux-arm64:
	GOOS=linux GOARCH=arm64 $(GOBUILD) -o build/$(BINARY)-linux-arm64 -ldflags="-s -w" ./cmd/maestro/

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o build/$(BINARY)-darwin-amd64 -ldflags="-s -w" ./cmd/maestro/

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o build/$(BINARY)-darwin-arm64 -ldflags="-s -w" ./cmd/maestro/

build-windows-amd64:
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o build/$(BINARY)-windows-amd64.exe -ldflags="-s -w" ./cmd/maestro/

# Test
test:
	$(GOTEST) -v ./...

# Lint (golangci-lint if available, fallback to go vet)
LINT_CMD=$(shell which golangci-lint 2>/dev/null || echo "")
LINT_FLAGS=--fix --timeout=5m
lint:
ifneq ($(LINT_CMD),)
	golangci-lint run $(LINT_FLAGS) ./...
else
	$(GOCMD) vet ./...
endif

# Run all verification gates (run before EVERY commit)
verify: build lint test-race coverage-check

# Quick pre-commit check (fast, no coverage)
precommit: build lint test

# Test with race detector
test-race:
	$(GOTEST) -race -coverprofile=$(COVERFILE) ./...

# Coverage report
coverage:
	$(GOCMD) tool cover -func=$(COVERFILE)

# Coverage check with threshold (advisory for now)
coverage-check: test-race
	@echo "--- Coverage ---"
	@$(GOCMD) tool cover -func=$(COVERFILE) | grep total:

# Clean
clean:
	$(GOCLEAN)
	rm -f $(BINARY)
	rm -rf build/
	rm -f $(COVERFILE)

# Format
fmt:
	$(GOFMT) ./...

# Tidy
tidy:
	$(GOCMD) mod tidy

# Install locally
install: build
	cp $(BINARY) $(GOPATH)/bin/

# Install pre-commit hooks
install-hooks:
	git config core.hooksPath .githooks
	@echo "Pre-commit hooks installed from .githooks/"

# Default
all: build

.PHONY: build build-all test lint verify precommit test-race coverage coverage-check clean fmt tidy install install-hooks