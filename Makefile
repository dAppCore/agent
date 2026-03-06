# Host UK Developer Workspace
# Run `make setup` to bootstrap your environment

CORE_REPO := github.com/host-uk/core
CORE_VERSION := latest
INSTALL_DIR := $(HOME)/.local/bin

.PHONY: all setup install-deps install-go install-core doctor clean help

all: help

help:
	@echo "Host UK Developer Workspace"
	@echo ""
	@echo "Usage:"
	@echo "  make setup        Full setup (deps + core + clone repos)"
	@echo "  make install-deps Install system dependencies (go, gh, etc)"
	@echo "  make install-core Build and install core CLI"
	@echo "  make doctor       Check environment health"
	@echo "  make clone        Clone all repos into packages/"
	@echo "  make clean        Remove built artifacts"
	@echo ""
	@echo "Quick start:"
	@echo "  make setup"

setup: install-deps install-core doctor clone
	@echo ""
	@echo "Setup complete! Run 'core health' to verify."

install-deps:
	@echo "Installing dependencies..."
	@./scripts/install-deps.sh

install-go:
	@echo "Installing Go..."
	@./scripts/install-go.sh

install-core:
	@echo "Installing core CLI..."
	@./scripts/install-core.sh

doctor:
	@core doctor || echo "Run 'make install-core' first if core is not found"

clone:
	@core setup || echo "Run 'make install-core' first if core is not found"

clean:
	@rm -rf ./build
	@echo "Cleaned build artifacts"
