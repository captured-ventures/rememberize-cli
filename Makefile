.PHONY: build test lint install-dev clean release-snapshot

GIT_SHA := $(shell git rev-parse --short HEAD 2>/dev/null || echo unknown)
VERSION ?= dev-$(GIT_SHA)
BIN := bin/rememberize
INSTALL_DIR ?= $(HOME)/.local/bin

build:
	CGO_ENABLED=0 go build -o $(BIN) -ldflags "-s -w -X main.version=$(VERSION)" ./cmd/rememberize

test:
	go test -race ./...

lint:
	golangci-lint run

install-dev: build
	mkdir -p $(INSTALL_DIR)
	cp $(BIN) $(INSTALL_DIR)/rememberize
	@echo "Installed to $(INSTALL_DIR)/rememberize"

clean:
	rm -rf bin/ dist/

release-snapshot:
	goreleaser release --snapshot --clean
