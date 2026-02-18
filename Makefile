BINARY_NAME ?= summary-sys
VARS_PKG ?= github.com/SisyphusSQ/summary-sys/vars
BUILD_DIR ?= bin
GO ?= go
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)

BUILD_FLAGS  = -X '$(VARS_PKG).AppName=$(BINARY_NAME)'
BUILD_FLAGS += -X '$(VARS_PKG).AppVersion=$(VERSION)'
BUILD_FLAGS += -X '$(VARS_PKG).GoVersion=$(shell $(GO) version)'
BUILD_FLAGS += -X '$(VARS_PKG).BuildTime=$(shell date +"%Y-%m-%d %H:%M:%S")'
BUILD_FLAGS += -X '$(VARS_PKG).GitCommit=$(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)'
BUILD_FLAGS += -X '$(VARS_PKG).GitRemote=$(shell git config --get remote.origin.url 2>/dev/null || echo unknown)'

LDFLAGS = -ldflags="$(BUILD_FLAGS)"

.PHONY: help all build release run test coverage lint fmt tidy clean linux-amd64 linux-arm64 darwin-amd64 darwin-arm64

help:
	@echo "Available targets:"
	@echo "  all           - Run fmt, test, then build"
	@echo "  build         - Build local binary"
	@echo "  release       - Build release binary with trimpath"
	@echo "  run           - Run CLI locally"
	@echo "  test          - Run tests with race detector"
	@echo "  coverage      - Run tests with coverage report"
	@echo "  lint          - Run golangci-lint"
	@echo "  fmt           - Run go fmt"
	@echo "  tidy          - Run go mod tidy"
	@echo "  clean         - Remove build artifacts"
	@echo "  linux-amd64   - Cross-compile for Linux amd64"
	@echo "  linux-arm64   - Cross-compile for Linux arm64"
	@echo "  darwin-amd64  - Cross-compile for macOS amd64"
	@echo "  darwin-arm64  - Cross-compile for macOS arm64"

all: fmt test build

build:
	@mkdir -p $(BUILD_DIR)
	$(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./main.go

release:
	@mkdir -p $(BUILD_DIR)
	$(GO) build -trimpath $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./main.go

run:
	$(GO) run ./main.go

test:
	$(GO) test -race ./...

coverage:
	$(GO) test -race -coverprofile=coverage.out ./...
	$(GO) tool cover -func=coverage.out

lint:
	golangci-lint run

fmt:
	$(GO) fmt ./...

tidy:
	$(GO) mod tidy

linux-amd64:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME).linux.amd64 ./main.go

linux-arm64:
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME).linux.arm64 ./main.go

darwin-amd64:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME).darwin.amd64 ./main.go

darwin-arm64:
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME).darwin.arm64 ./main.go

clean:
	rm -rf $(BUILD_DIR) coverage.out
