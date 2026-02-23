#!/bin/sh
set -e

if ! command -v golangci-lint >/dev/null 2>&1; then
    echo "ERROR: golangci-lint is not installed."
    echo ""
    echo "Install via:"
    echo "  go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@latest"
    echo "  or: brew install golangci-lint"
    exit 1
fi

golangci-lint run ./...
