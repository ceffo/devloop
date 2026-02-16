#!/bin/bash
# Convenience script to run Modal's dev-loop.sh on the devloop project
# Usage: ./scripts/dev-loop-run.sh [wave] [start_task] [end_task]

set -e

DEVLOOP_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
MODAL_DIR="$HOME/code/modal"
MODAL_DEV_LOOP="$MODAL_DIR/scripts/dev-loop.sh"

if [ ! -f "$MODAL_DEV_LOOP" ]; then
    echo "Error: Modal dev-loop.sh not found at: $MODAL_DEV_LOOP"
    echo "Please ensure the Modal project exists at $MODAL_DIR"
    exit 1
fi

# Change to devloop directory
cd "$DEVLOOP_DIR"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Running dev-loop.sh on devloop project"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Project:    devloop"
echo "Directory:  $DEVLOOP_DIR"
echo "Tasks file: docs/TASKS.md"
echo "Wave:       ${1:-all}"
echo ""

# Run Modal's dev-loop.sh from devloop directory
"$MODAL_DEV_LOOP" "$@"
