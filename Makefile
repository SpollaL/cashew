VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build test lint clean help

build: ## Compile both binaries (version injected from git tag)
	go build $(LDFLAGS) -o cashew ./cmd/cashew
	go build $(LDFLAGS) -o cashew-server ./cmd/server

test: ## Run all tests
	go test ./...

lint: ## Run golangci-lint (requires golangci-lint v2)
	golangci-lint run ./...

clean: ## Remove compiled binaries
	rm -f cashew cashew-server

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | awk 'BEGIN {FS = ":.*##"}; {printf "  %-10s %s\n", $$1, $$2}'
