#!/bin/bash
# Smoke test for chat API
# Requires web server running: make web (default port 8080)

set -e

WEB_URL="${WEB_URL:-http://localhost:8080}"

echo "=== Chat API Smoke Test ==="
echo "Web server URL: $WEB_URL"
echo

# Test 1: Health check
echo "--- Test 1: Health Check ---"
echo "GET $WEB_URL/api/chat/health"
HEALTH=$(curl -s "$WEB_URL/api/chat/health")
echo "Response:"
echo "$HEALTH" | jq .
echo

# Check if health endpoint returned valid JSON
if ! echo "$HEALTH" | jq . > /dev/null 2>&1; then
    echo "ERROR: Invalid JSON from health endpoint"
    exit 1
fi

# Check provider field
PROVIDER=$(echo "$HEALTH" | jq -r '.provider // "none"')
OK=$(echo "$HEALTH" | jq -r '.ok // false')
echo "Provider: $PROVIDER"
echo "OK: $OK"
echo

if [ "$OK" != "true" ]; then
    echo "WARNING: Chat runtime is not available."
    echo "This is expected if Ollama/LM Studio is not running."
    echo
    echo "To set up Ollama:"
    echo "  brew install ollama"
    echo "  ollama serve"
    echo "  ollama pull llama2"
    echo
    echo "Skipping chat test (runtime unavailable)"
    exit 0
fi

# Test 2: Send a chat message
echo "--- Test 2: Chat Request ---"
echo "POST $WEB_URL/api/chat"
CHAT_RESPONSE=$(curl -s -X POST "$WEB_URL/api/chat" \
    -H "Content-Type: application/json" \
    -d '{"messages":[{"role":"user","content":"Say hello in exactly 3 words."}]}')

echo "Response:"
echo "$CHAT_RESPONSE" | jq .
echo

# Check if chat returned valid JSON
if ! echo "$CHAT_RESPONSE" | jq . > /dev/null 2>&1; then
    echo "ERROR: Invalid JSON from chat endpoint"
    exit 1
fi

# Check for reply field
REPLY=$(echo "$CHAT_RESPONSE" | jq -r '.reply // empty')
if [ -n "$REPLY" ]; then
    echo "Chat reply received: OK"
    echo "Reply: $REPLY"
else
    # Check for error
    ERROR=$(echo "$CHAT_RESPONSE" | jq -r '.error // empty')
    if [ -n "$ERROR" ]; then
        echo "Chat error: $ERROR"
        exit 1
    fi
    echo "WARNING: No reply in response"
fi

echo
echo "=== Smoke test completed ==="
