# ------------------------------------------------------------------------------
# Project
# ------------------------------------------------------------------------------

APP_NAME := goshi

GO       ?= go
PREFIX   ?= $(HOME)/.local
GOFLAGS  :=

# ------------------------------------------------------------------------------
# Default target
# ------------------------------------------------------------------------------

.PHONY: all
all: build

# ------------------------------------------------------------------------------
# Build
# ------------------------------------------------------------------------------

.PHONY: build compile
build: install
compile:
	GO="$(GO)" PREFIX="$(PREFIX)" bash scripts/build-local.sh --compile-only

# ------------------------------------------------------------------------------
# Run
# ------------------------------------------------------------------------------

.PHONY: run
run: compile
	./.build/$(APP_NAME)

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
	rm -f $(APP_NAME)

# ------------------------------------------------------------------------------
# Install (developer-local, not system-wide)
# ------------------------------------------------------------------------------

.PHONY: install
install:
	GO="$(GO)" PREFIX="$(PREFIX)" bash scripts/build-local.sh

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
	@echo "  build         Build and install ~/.local/bin/goshi"
	@echo "  run           Build and run"
	@echo "  test          Run tests"
	@echo "  vet           Run go vet"
	@echo "  fmt           Run go fmt"
	@echo "  clean         Remove build artifacts"
	@echo "  install       Build and install using PREFIX (default ~/.local)"
