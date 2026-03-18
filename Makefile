.PHONY: all build install uninstall clean help test

# Build variables
BINARY_NAME=xagent
BUILD_DIR=build
CMD_DIR=cmd/$(BINARY_NAME)
MAIN_GO=$(CMD_DIR)/main.go

# Version
VERSION?=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT=$(shell git rev-parse --short=8 HEAD 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date +%FT%T%z)
GO_VERSION=$(shell $(GO) version | awk '{print $$3}')
# SWE100821: -s -w strips debug info and DWARF tables, reducing binary ~30% (from PicoClaw Makefile)
LDFLAGS=-ldflags "-s -w -X main.version=$(VERSION) -X main.gitCommit=$(GIT_COMMIT) -X main.buildTime=$(BUILD_TIME) -X main.goVersion=$(GO_VERSION)"

# Go variables
GO?=go
GOFLAGS?=-v

# Installation
INSTALL_PREFIX?=$(HOME)/.local
INSTALL_BIN_DIR=$(INSTALL_PREFIX)/bin
INSTALL_MAN_DIR=$(INSTALL_PREFIX)/share/man/man1

# Workspace and Skills
XAGENT_HOME?=$(HOME)/.xagent
WORKSPACE_DIR?=$(XAGENT_HOME)/workspace
WORKSPACE_SKILLS_DIR=$(WORKSPACE_DIR)/skills
BUILTIN_SKILLS_DIR=$(CURDIR)/skills

# OS detection
UNAME_S:=$(shell uname -s)
UNAME_M:=$(shell uname -m)

# Platform-specific settings
ifeq ($(UNAME_S),Linux)
	PLATFORM=linux
	ifeq ($(UNAME_M),x86_64)
		ARCH=amd64
	else ifeq ($(UNAME_M),aarch64)
		ARCH=arm64
	else ifeq ($(UNAME_M),riscv64)
		ARCH=riscv64
	else
		ARCH=$(UNAME_M)
	endif
else ifeq ($(UNAME_S),Darwin)
	PLATFORM=darwin
	ifeq ($(UNAME_M),x86_64)
		ARCH=amd64
	else ifeq ($(UNAME_M),arm64)
		ARCH=arm64
	else
		ARCH=$(UNAME_M)
	endif
else
	PLATFORM=$(UNAME_S)
	ARCH=$(UNAME_M)
endif

BINARY_PATH=$(BUILD_DIR)/$(BINARY_NAME)-$(PLATFORM)-$(ARCH)

# Default target
all: build

## generate: Run generate (SWE100821: scope to project dirs only — ./... scans reference/ and OOMs)
generate:
	@echo "Run generate..."
	@rm -r ./$(CMD_DIR)/workspace 2>/dev/null || true
	@$(GO) generate ./cmd/... ./pkg/...
	@echo "Run generate complete"

## build: Build the xagent binary for current platform
build: generate check-embed
	@echo "Building $(BINARY_NAME) for $(PLATFORM)/$(ARCH)..."
	@mkdir -p $(BUILD_DIR)
	@$(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_PATH) ./$(CMD_DIR)
	@echo "Build complete: $(BINARY_PATH)"
	@ln -sf $(BINARY_NAME)-$(PLATFORM)-$(ARCH) $(BUILD_DIR)/$(BINARY_NAME)

## build-all: Build xagent for all platforms
build-all: generate
	@echo "Building for multiple platforms..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 ./$(CMD_DIR)
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-arm64 ./$(CMD_DIR)
	GOOS=linux GOARCH=riscv64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-riscv64 ./$(CMD_DIR)
	GOOS=darwin GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 ./$(CMD_DIR)
	GOOS=windows GOARCH=amd64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe ./$(CMD_DIR)
	@echo "All builds complete"

## xavier: Cross-compile for Jetson Xavier (ARM64) + bundle PicoLM for deployment (SWE100821)
xavier: generate check-embed
	@echo "Building xagent for Jetson Xavier (linux/arm64)..."
	@mkdir -p $(BUILD_DIR)/xavier-deploy
	GOOS=linux GOARCH=arm64 $(GO) build $(LDFLAGS) -o $(BUILD_DIR)/xavier-deploy/xagent ./$(CMD_DIR)
	@echo "Cross-compiling PicoLM for ARM64..."
	@if [ -d reference/picolm/picolm ]; then \
		cd reference/picolm/picolm && \
		$(MAKE) clean 2>/dev/null; \
		if command -v aarch64-linux-gnu-gcc >/dev/null 2>&1; then \
			$(MAKE) cross-pi && cp picolm ../../../$(BUILD_DIR)/xavier-deploy/picolm; \
		else \
			echo "  ⚠ aarch64-linux-gnu-gcc not found — PicoLM must be built on Xavier"; \
			echo "  Install: sudo apt install gcc-aarch64-linux-gnu"; \
		fi; \
		cd ../../..; \
	else \
		echo "  ⚠ reference/picolm not found — run: git submodule update --init reference/picolm"; \
	fi
	@cp start.sh $(BUILD_DIR)/xavier-deploy/start.sh
	@echo ""
	@echo "Deploy bundle ready: $(BUILD_DIR)/xavier-deploy/"
	@echo "  xagent         — ARM64 binary"
	@echo "  picolm          — PicoLM inference binary (if cross-compiled)"
	@echo "  start.sh        — Full installer"
	@echo ""
	@echo "Transfer to Xavier:"
	@echo "  scp -r $(BUILD_DIR)/xavier-deploy/ xavier:~/xagent/"
	@echo "  ssh xavier 'cd ~/xagent && bash start.sh'"

## release: Build static release binary with CGO disabled (SWE100821: from PicoClaw pattern)
release: generate check-embed
	@echo "Building static release $(BINARY_NAME) for $(PLATFORM)/$(ARCH)..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 $(GO) build $(GOFLAGS) $(LDFLAGS) -o $(BINARY_PATH) ./$(CMD_DIR)
	@echo "Release build complete: $(BINARY_PATH)"
	@ln -sf $(BINARY_NAME)-$(PLATFORM)-$(ARCH) $(BUILD_DIR)/$(BINARY_NAME)

## picolm-native: Build PicoLM with -march=native for host SIMD (SWE100821: 2-4x tok/s on ARM)
picolm-native:
	@if [ -d reference/picolm/picolm ]; then \
		echo "Building PicoLM with native SIMD optimizations..."; \
		cd reference/picolm/picolm && $(MAKE) native; \
		echo "PicoLM built. Install: cd reference/picolm/picolm && sudo make install"; \
	else \
		echo "reference/picolm not found — run: git submodule update --init reference/picolm"; \
	fi

## install: Install xagent to system and copy builtin skills
install: build
	@echo "Installing $(BINARY_NAME)..."
	@mkdir -p $(INSTALL_BIN_DIR)
	@cp $(BUILD_DIR)/$(BINARY_NAME) $(INSTALL_BIN_DIR)/$(BINARY_NAME)
	@chmod +x $(INSTALL_BIN_DIR)/$(BINARY_NAME)
	@echo "Installed binary to $(INSTALL_BIN_DIR)/$(BINARY_NAME)"
	@echo "Installation complete!"

## uninstall: Remove xagent from system
uninstall:
	@echo "Uninstalling $(BINARY_NAME)..."
	@rm -f $(INSTALL_BIN_DIR)/$(BINARY_NAME)
	@echo "Removed binary from $(INSTALL_BIN_DIR)/$(BINARY_NAME)"
	@echo "Note: Only the executable file has been deleted."
	@echo "If you need to delete all configurations (config.json, workspace, etc.), run 'make uninstall-all'"

## uninstall-all: Remove xagent and all data
uninstall-all:
	@echo "Removing workspace and skills..."
	@rm -rf $(XAGENT_HOME)
	@echo "Removed workspace: $(XAGENT_HOME)"
	@echo "Complete uninstallation done!"

## clean: Remove build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf $(BUILD_DIR)
	@echo "Clean complete"

## vet: Run go vet
vet:
	@$(GO) vet ./...

## test: Run go test
test:
	@$(GO) test ./...

## fmt: Format Go code
fmt:
	@$(GO) fmt ./...

## check-embed: Verify embedded files exist before build (SWE100821)
check-embed:
	@test -f pkg/skills/catalog.json || (echo "ERROR: pkg/skills/catalog.json missing (required by //go:embed). Run 'git checkout pkg/skills/catalog.json' or rebuild the catalog." && exit 1)

## deps: Update dependencies (SWE100821: scoped to cmd/pkg to avoid skills/ stray Go files)
deps:
	@$(GO) get -u ./cmd/... ./pkg/...
	@$(GO) mod tidy

## run: Build and run xagent
run: build
	@$(BUILD_DIR)/$(BINARY_NAME) $(ARGS)

## help: Show this help message
help:
	@echo "Xagent Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make [target]"
	@echo ""
	@echo "Targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'
	@echo ""
	@echo "Examples:"
	@echo "  make build              # Build for current platform"
	@echo "  make install            # Install to ~/.local/bin"
	@echo "  make uninstall          # Remove from ~/.local/bin"
	@echo "  make install-skills     # Install skills to workspace"
	@echo ""
	@echo "Environment Variables:"
	@echo "  INSTALL_PREFIX          # Installation prefix (default: ~/.local)"
	@echo "  WORKSPACE_DIR           # Workspace directory (default: ~/.xagent/workspace)"
	@echo "  VERSION                 # Version string (default: git describe)"
	@echo ""
	@echo "Current Configuration:"
	@echo "  Platform: $(PLATFORM)/$(ARCH)"
	@echo "  Binary: $(BINARY_PATH)"
	@echo "  Install Prefix: $(INSTALL_PREFIX)"
	@echo "  Workspace: $(WORKSPACE_DIR)"
