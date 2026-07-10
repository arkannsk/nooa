#!/usr/bin/env bash
# gen-client.sh — Generate an HTTP client for a given elval-integration example.
#
# Usage: make gen-client EXAMPLE=01_basic_types
#
# Steps:
#   1. Parse main.go AST to extract routes, types and imports
#   2. Generate client code using Go templates
#   3. Output goes to examples/elval-integration/clients/<EXAMPLE>/client.go

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
EXAMPLE="${EXAMPLE:?set EXAMPLE env var, e.g. 01_basic_types}"
EXAMPLE_DIR="$ROOT/examples/elval-integration/$EXAMPLE"
CLIENT_DIR="$ROOT/examples/elval-integration/clients/$EXAMPLE"
CLIENT_FILE="$CLIENT_DIR/client.go"

if [ ! -d "$EXAMPLE_DIR" ]; then
    echo "ERROR: example directory not found: $EXAMPLE_DIR"
    exit 1
fi

# Create output directory
mkdir -p "$CLIENT_DIR"

echo "Generating client for: $EXAMPLE"
go run "$ROOT/cmd/clientgen" \
    -pkg "$EXAMPLE_DIR" \
    -out "$CLIENT_FILE"

echo "Client generated: $CLIENT_FILE"
echo "Done!"
