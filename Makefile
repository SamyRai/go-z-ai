# Common development tasks for go-z-ai.
# Mirrors the commands documented in CONTRIBUTING.md so contributors have one
# entry point. All targets are phony (no file outputs).

GO ?= go
GOLANGCI_LINT ?= golangci-lint
GOVULNCHECK ?= govulncheck
MARKDOWNLINT ?= markdownlint-cli2
LYCHEE ?= lychee
MD_GLOBS := README*.md docs/**/*.md CONTRIBUTING.md examples/README.md .github/**/*.md
# LYCHEE_FLAGS mirrors the args used in .github/workflows/ci.yml so a local
# `make docs-lint` and CI agree. Exclude pkg.go.dev / docs.z.ai — both are
# sometimes flaky under link-checker user agents.
LYCHEE_FLAGS := --no-progress --max-cache-age 30d --max-concurrency 10 --exclude 'pkg\.go\.dev' --exclude 'docs\.z\.ai'

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

.PHONY: docs-lint
docs-lint: ## Lint markdown: structure (markdownlint-cli2) + links (lychee).
	$(MARKDOWNLINT) $(MD_GLOBS)
	$(LYCHEE) $(LYCHEE_FLAGS) $(MD_GLOBS)

.PHONY: docs-lint-links
docs-lint-links: ## Check markdown links only (local + external).
	$(LYCHEE) $(LYCHEE_FLAGS) --cache $(MD_GLOBS)

.PHONY: docs-fix
docs-fix: ## Auto-fix markdownlint-cli2 issues.
	$(MARKDOWNLINT) --fix $(MD_GLOBS)

.PHONY: ci-local
ci-local: fmt-check vet lint test vuln docs-lint ## Run the full CI-equivalent check locally.

.PHONY: clean
clean: ## Remove built binary, coverage artifacts, and generated site.
	rm -f zai-client sitegen cover.out coverage.txt coverage.html
	rm -rf site

# ─── Static site generation ────────────────────────────────────────────
# The site generator renders the project's markdown docs + dynamic GitHub
# data into static HTML at ./site. See docs/en/site-generation.md.

SITEGEN ?= go run ./cmd/sitegen
SITE_OUT ?= site

.PHONY: site
site: ## Generate the static HTML site into ./site.
	$(SITEGEN) -out $(SITE_OUT)

.PHONY: site-offline
site-offline: ## Generate the site without GitHub API calls (sandbox / no network).
	$(SITEGEN) -out $(SITE_OUT) -offline

.PHONY: site-serve
site-serve: ## Generate site and serve on http://localhost:8000.
	$(SITEGEN) -out $(SITE_OUT)
	@echo "Serving $(SITE_OUT) at http://localhost:8000 — Ctrl-C to stop"
	@cd $(SITE_OUT) && python3 -m http.server 8000

.PHONY: site-clean
site-clean: ## Remove the generated site.
	rm -rf $(SITE_OUT)
