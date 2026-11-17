BINARY  := txoko
PKG     := github.com/LifeOfAitor/txoko
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X $(PKG)/internal/version.Version=$(VERSION) \
	-X $(PKG)/internal/version.Commit=$(COMMIT) \
	-X $(PKG)/internal/version.Date=$(DATE)

.DEFAULT_GOAL := help

.PHONY: help build run test cover lint fmt check clean demo

help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"} /^[a-zA-Z_-]+:.*##/ {printf "  \033[32m%-8s\033[0m %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build ./bin/txoko with version info
	go build -trimpath -ldflags "$(LDFLAGS)" -o bin/$(BINARY) ./cmd/$(BINARY)

run: build ## Build and run txoko
	./bin/$(BINARY)

test: ## Run unit tests with the race detector
	go test -race ./...

cover: ## Run tests and open the coverage report
	go test -race -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | tail -1
	go tool cover -html=coverage.out

lint: ## Run golangci-lint
	golangci-lint run

fmt: ## Format code
	golangci-lint fmt

check: fmt lint test ## Format, lint and test (run before committing)

demo: build ## Record the README images with VHS (brew install vhs)
	@for tape in docs/tapes/*.tape; do \
		case $$tape in */settings.tape) continue;; esac; \
		echo "  recording $$tape"; vhs $$tape >/dev/null || exit 1; \
	done
	@rm -f docs/images/season-*.gif docs/images/once.gif docs/images/tmux.gif
	@tmux -L txoko-readme kill-server 2>/dev/null || true

clean: ## Remove build artifacts
	rm -rf bin coverage.out
