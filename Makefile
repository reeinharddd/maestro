# Binary name
BINARY=okit
# Go parameters
GOCMD=go
GOBUILD=$(GOCMD) build
GOCLEAN=$(GOCMD) clean
GOTEST=$(GOCMD) test
GOGET=$(GOCMD) get
GOFMT=$(GOCMD) fmt

# Build for current platform
build:
	$(GOBUILD) -o $(BINARY) -ldflags="-s -w" ./cmd/okit/

# Build for all platforms
build-all: build-linux-amd64 build-linux-arm64 build-darwin-amd64 build-darwin-arm64 build-windows-amd64

build-linux-amd64:
	GOOS=linux GOARCH=amd64 $(GOBUILD) -o build/$(BINARY)-linux-amd64 -ldflags="-s -w" ./cmd/okit/

build-linux-arm64:
	GOOS=linux GOARCH=arm64 $(GOBUILD) -o build/$(BINARY)-linux-arm64 -ldflags="-s -w" ./cmd/okit/

build-darwin-amd64:
	GOOS=darwin GOARCH=amd64 $(GOBUILD) -o build/$(BINARY)-darwin-amd64 -ldflags="-s -w" ./cmd/okit/

build-darwin-arm64:
	GOOS=darwin GOARCH=arm64 $(GOBUILD) -o build/$(BINARY)-darwin-arm64 -ldflags="-s -w" ./cmd/okit/

build-windows-amd64:
	GOOS=windows GOARCH=amd64 $(GOBUILD) -o build/$(BINARY)-windows-amd64.exe -ldflags="-s -w" ./cmd/okit/

# Test
test:
	$(GOTEST) -v ./...

# Lint
lint:
	$(GOCMD) vet ./...

# Clean
clean:
	$(GOCLEAN)
	rm -f $(BINARY)
	rm -rf build/

# Format
fmt:
	$(GOFMT) ./...

# Tidy
tidy:
	$(GOCMD) mod tidy

# Install locally
install: build
	cp $(BINARY) $(GOPATH)/bin/

# Default
all: build

.PHONY: build build-all test lint clean fmt tidy install