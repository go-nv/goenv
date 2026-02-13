#!/usr/bin/env bash
# Cross-platform build wrapper for Unix systems
# Usage: ./build.sh [task] [options]

set -e

TASK="${1:-build}"
shift || true

exec go run scripts/build-tool/main.go -task="$TASK" "$@"
