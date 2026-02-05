#!/bin/bash
# Smoke test for qai-hub doctor command
# Verifies qai-hub CLI integration is working

set -e

echo "=== QAI Hub Doctor Smoke Test ==="
echo

# Run doctor command
echo "Running: go run ./cmd/edgecli qaihub doctor --json"
OUTPUT=$(go run ./cmd/edgecli qaihub doctor --json 2>&1)

echo "Output:"
echo "$OUTPUT" | head -20
echo

# Check if output is valid JSON
if echo "$OUTPUT" | jq . > /dev/null 2>&1; then
    echo "JSON output: VALID"
else
    echo "JSON output: INVALID"
    echo "Full output:"
    echo "$OUTPUT"
    exit 1
fi

# Check for qai_hub_found field
QAI_HUB_FOUND=$(echo "$OUTPUT" | jq -r '.qai_hub_found // false')
echo "qai_hub_found: $QAI_HUB_FOUND"

if [ "$QAI_HUB_FOUND" = "true" ]; then
    echo
    echo "qai-hub CLI is available."
    VERSION=$(echo "$OUTPUT" | jq -r '.qai_hub_version // "unknown"')
    echo "Version: $VERSION"
else
    echo
    echo "qai-hub CLI is NOT available."
    echo "This is expected if qai-hub is not installed."
    echo "Install with: pip install qai-hub"
fi

echo
echo "=== Smoke test completed ==="
