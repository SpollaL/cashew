VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

.PHONY: build test lint clean

build:
	go build $(LDFLAGS) -o cashew ./cmd/cashew
	go build $(LDFLAGS) -o cashew-server ./cmd/server

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -f cashew cashew-server
