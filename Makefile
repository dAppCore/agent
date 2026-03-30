
# ── core-agent binary ──────────────────────────────────

BINARY_NAME=core-agent
CMD_PATH=./cmd/core-agent
MODULE_PATH=dappco.re/go/agent

# Default LDFLAGS to empty
LDFLAGS = ""

# If VERSION is set, inject into binary
ifdef VERSION
	LDFLAGS = -ldflags "-X '$(MODULE_PATH).Version=$(VERSION)'"
endif

.PHONY: build install agent-dev test coverage

build:
	@echo "Building $(BINARY_NAME)..."
	@go build $(LDFLAGS) -o $(BINARY_NAME) $(CMD_PATH)

install:
	@echo "Installing $(BINARY_NAME)..."
	@go install $(LDFLAGS) $(CMD_PATH)

agent-dev: build
	@./$(BINARY_NAME) version

test:
	@echo "Running tests..."
	@go test ./...

coverage:
	@echo "Generating coverage report..."
	@go test -coverprofile=coverage.out ./...
	@echo "Coverage: coverage.out"
