.DEFAULT_GOAL := help

BINARY_DIR := bin
BINARY := $(BINARY_DIR)/tracedelta
GO ?= go
GOFMT ?= gofmt

.PHONY: help build test test-race test-fuzz-smoke test-release-smoke fmt fmt-check vet check run-example clean

help: ## Show available targets.
	@awk 'BEGIN {FS = ":.*## "; printf "TraceDelta development targets:\n\n"} /^[a-zA-Z0-9_-]+:.*## / {printf "  %-14s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the TraceDelta CLI.
	@mkdir -p $(BINARY_DIR)
	$(GO) build -trimpath -o $(BINARY) ./cmd/tracedelta

test: ## Run the full Go test suite.
	$(GO) test ./...

test-race: ## Run tests with the race detector.
	$(GO) test -race ./...

test-fuzz-smoke: ## Run bounded parser, matcher, and reporter fuzz targets.
	bash ./scripts/fuzz-smoke.sh

test-release-smoke: ## Cross-build and verify the five-target release bundle.
	bash ./scripts/build-release.smoke.sh

fmt: ## Format all Go source files.
	find . -type f -name '*.go' ! -path './vendor/*' -exec $(GOFMT) -w {} +

fmt-check: ## Fail if any Go source file is not gofmt-formatted.
	@unformatted="$$(find . -type f -name '*.go' ! -path './vendor/*' -exec $(GOFMT) -l {} + | sort)"; \
	if [ -n "$$unformatted" ]; then \
		printf 'The following files require gofmt:\n%s\n' "$$unformatted"; \
		exit 1; \
	fi

vet: ## Run Go static analysis.
	$(GO) vet ./...

check: fmt-check test vet build ## Run all essential local checks.

run-example: build ## Run the sample comparison and verify its expected exit code (1).
	@set +e; \
	./$(BINARY) compare --baseline testdata/baseline.json --candidate testdata/candidate.json; \
	status=$$?; \
	set -e; \
	if [ "$$status" -ne 1 ]; then \
		printf 'Expected example exit code 1, got %s\n' "$$status" >&2; \
		exit 1; \
	fi; \
	printf 'Example produced the expected exit code 1.\n'

clean: ## Remove repository-local build and test artifacts.
	$(GO) clean
	rm -rf -- ./$(BINARY_DIR) ./coverage.out ./coverage.txt ./reports ./tracedelta ./tracedelta.exe
