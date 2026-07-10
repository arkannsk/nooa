#!/usr/bin/env bash
# gen-client.sh — Generate an HTTP client for a given elval-integration example.
#
# Usage: make gen-client EXAMPLE=01_basic_types
#
# Steps:
#   1. Build the example binary
#   2. Start it in background
#   3. Download openapi.json from the running server
#   4. Stop the server
#   5. Run clientgen on the downloaded spec
#   6. Output goes to examples/elval-integration/clients/<EXAMPLE>/client.go

set -euo pipefail

ROOT="$(cd "$(dirname "$0")/.." && pwd)"
EXAMPLE="${EXAMPLE:?set EXAMPLE env var, e.g. 01_basic_types}"
EXAMPLE_DIR="$ROOT/examples/elval-integration/$EXAMPLE"
CLIENT_DIR="$ROOT/examples/elval-integration/clients/$EXAMPLE"
SPEC_FILE="$CLIENT_DIR/openapi.json"
CLIENT_FILE="$CLIENT_DIR/client.go"

if [ ! -d "$EXAMPLE_DIR" ]; then
    echo "ERROR: example directory not found: $EXAMPLE_DIR"
    exit 1
fi

# Determine package path
PKG_PATH="github.com/arkannsk/nooa/examples/elval-integration/$EXAMPLE"

# Create output directory
mkdir -p "$CLIENT_DIR"

echo "Building example: $EXAMPLE"
(cd "$EXAMPLE_DIR" && go build -o /tmp/nooa_example_bin .)

echo "Starting server..."
/tmp/nooa_example_bin &
SERVER_PID=$!
trap "kill $SERVER_PID 2>/dev/null; wait $SERVER_PID 2>/dev/null" EXIT

# Wait for server to be ready
echo "Waiting for server on port 9090..."
for i in $(seq 1 30); do
    if curl -sf http://localhost:9090/openapi.json > /dev/null 2>&1; then
        break
    fi
    sleep 0.5
done

echo "Downloading OpenAPI spec..."
curl -sf http://localhost:9090/openapi.json -o "$SPEC_FILE"
echo "Spec saved to: $SPEC_FILE"

# Stop server
kill $SERVER_PID 2>/dev/null
wait $SERVER_PID 2>/dev/null || true
trap - EXIT

echo "Generating client..."
go run "$ROOT/cmd/clientgen" \
    -spec "$SPEC_FILE" \
    -out "$CLIENT_FILE" \
    -base "github.com/arkannsk/nooa/examples" \
    -models-dir "models" \
    -model-map "models=github.com/arkannsk/nooa/examples/elval-integration/models"

echo "Client generated: $CLIENT_FILE"
echo "Done!"
