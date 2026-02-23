#!/bin/sh
set -e

# Resolve symlinks (supports pre-commit hook symlink from .git/hooks/)
if [ -L "$0" ]; then
    SCRIPT_DIR="$(cd "$(dirname "$0")" && cd "$(dirname "$(readlink "$(basename "$0")")")" && pwd)"
else
    SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
fi

# Change to repository root for consistent working directory
cd "$SCRIPT_DIR/.."

echo "[1/2] Running lint..."
"$SCRIPT_DIR/checks/lint.sh"

echo "[2/2] Running tests..."
"$SCRIPT_DIR/checks/test.sh"

echo ""
echo "All checks passed."
