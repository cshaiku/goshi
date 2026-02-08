# ------------------------------------------------------------------------------
# Project
# ------------------------------------------------------------------------------

APP_NAME := goshi
BIN_DIR  := bin
BIN_PATH := $(BIN_DIR)/$(APP_NAME)

GO       := go
GOFLAGS  :=

# ------------------------------------------------------------------------------
# Default target
# ------------------------------------------------------------------------------

.PHONY: all
all: build

# ------------------------------------------------------------------------------
# Build
# ------------------------------------------------------------------------------

.PHONY: build
build:
	@mkdir -p $(BIN_DIR)
	$(GO) build $(GOFLAGS) -o $(BIN_PATH)

# Build directly to repo root (local dev convenience)
.PHONY: build-local
build-local:
	$(GO) build $(GOFLAGS) -o $(APP_NAME)

# ------------------------------------------------------------------------------
# Run
# ------------------------------------------------------------------------------

.PHONY: run
run: build
	./$(BIN_PATH)

# ------------------------------------------------------------------------------
# Test / Verify
# ------------------------------------------------------------------------------

.PHONY: test
test:
	$(GO) test ./...

.PHONY: vet
vet:
	$(GO) vet ./...

.PHONY: fmt
fmt:
	$(GO) fmt ./...

# ------------------------------------------------------------------------------
# Clean
# ------------------------------------------------------------------------------

.PHONY: clean
clean:
	rm -rf $(BIN_DIR)
	rm -f $(APP_NAME)

# ------------------------------------------------------------------------------
# Install (developer-local, not system-wide)
# ------------------------------------------------------------------------------

.PHONY: install
install:
	$(GO) install .

# ------------------------------------------------------------------------------
# Safety helpers
# ------------------------------------------------------------------------------

.PHONY: check-dirty
check-dirty:
	@if ! git diff --quiet || ! git diff --cached --quiet; then \
		echo "ERROR: working tree is dirty"; \
		exit 1; \
	fi

# ------------------------------------------------------------------------------
# Info
# ------------------------------------------------------------------------------

.PHONY: help
help:
	@echo "Targets:"
	@echo "  build         Build binary to ./bin/goshi"
	@echo "  build-local   Build binary to ./goshi (local only)"
	@echo "  run           Build and run"
	@echo "  test          Run tests"
	@echo "  vet           Run go vet"
	@echo "  fmt           Run go fmt"
	@echo "  clean         Remove build artifacts"
	@echo "  install       go install (developer-local)"
