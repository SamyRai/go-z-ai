# Common development tasks for go-z-ai.
# Mirrors the commands documented in CONTRIBUTING.md so contributors have one
# entry point. All targets are phony (no file outputs).

GO ?= go
GOLANGCI_LINT ?= golangci-lint
GOVULNCHECK ?= govulncheck

.PHONY: help
help: ## Show this help.
	@awk 'BEGIN {FS = ":.*##"; printf "Usage:\n  make <target>\n\nTargets:\n"} \
	/^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-14s\033[0m %s\n", $$1, $$2 }' $(MAKEFILE_LIST)

.PHONY: build
build: ## Build all packages and the CLI binary.
	$(GO) build ./...
	$(GO) build -o zai-client .

.PHONY: vet
vet: ## Run go vet.
	$(GO) vet ./...

.PHONY: test
test: ## Run tests with the race detector.
	$(GO) test -race ./...

.PHONY: test-cover
test-cover: ## Run tests with coverage, write cover.out.
	$(GO) test -race -coverprofile=cover.out -covermode=atomic ./...
	@$(GO) tool cover -func=cover.out | tail -1

.PHONY: fmt
fmt: ## Format all Go source.
	$(GO) fmt ./...

.PHONY: fmt-check
fmt-check: ## Fail if any file is not gofmt-formatted.
	@diff=$$(gofmt -l .); \
	if [ -n "$$diff" ]; then echo "Unformatted files:\n$$diff"; exit 1; fi

.PHONY: lint
lint: ## Run golangci-lint.
	$(GOLANGCI_LINT) run ./...

.PHONY: vuln
vuln: ## Run govulncheck.
	$(GOVULNCHECK) ./...

.PHONY: tidy
tidy: ## Run go mod tidy.
	$(GO) mod tidy

.PHONY: ci-local
ci-local: fmt-check vet lint test vuln ## Run the full CI-equivalent check locally.

.PHONY: clean
clean: ## Remove built binary and coverage artifacts.
	rm -f zai-client cover.out coverage.txt coverage.html
