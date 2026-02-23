.PHONY: setup check build help

help: ## Show available targets
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "  %-15s %s\n", $$1, $$2}'

setup: ## Install pre-commit hook and check tool dependencies
	@if [ ! -d .git ]; then echo "ERROR: .git directory not found. Run from repository root."; exit 1; fi
	@echo "Installing pre-commit hook..."
	@ln -sf ../../scripts/pre-commit-checks.sh .git/hooks/pre-commit
	@echo "Pre-commit hook installed."
	@echo ""
	@echo "Checking tool dependencies..."
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "  golangci-lint: OK ($$(golangci-lint --version 2>&1 | head -1))"; \
	else \
		echo "  golangci-lint: MISSING"; \
		echo "    Install: go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest"; \
		echo "    Or:      brew install golangci-lint"; \
	fi

check: ## Run all quality checks (lint + test)
	@scripts/pre-commit-checks.sh

build: ## Build the wachat binary
	go build -o wachat ./cmd/
