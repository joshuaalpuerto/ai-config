#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
source "$SCRIPT_DIR/transpile.sh"

echo ""
echo "Building ai-config..."
echo ""

transpile_all

echo ""
echo "Done."
