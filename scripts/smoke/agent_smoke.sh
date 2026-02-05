#!/bin/bash
# Agent Smoke Test
# Tests the /api/agent endpoint with tool calling
#
# Prerequisites:
# 1. Start gRPC server: make server
# 2. Start web server with LM Studio: CHAT_PROVIDER=openai CHAT_BASE_URL=http://localhost:1234 make web
# 3. Have LM Studio running with a model loaded

set -e

WEB_ADDR="${WEB_ADDR:-localhost:8080}"
BASE_URL="http://${WEB_ADDR}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "========================================="
echo "Agent Smoke Test"
echo "========================================="
echo "Web server: ${BASE_URL}"
echo ""

# Test 1: Health check
echo -e "${YELLOW}Test 1: Agent Health Check${NC}"
HEALTH=$(curl -s "${BASE_URL}/api/agent/health")
echo "Response: ${HEALTH}"

if echo "$HEALTH" | grep -q '"ok":true'; then
    echo -e "${GREEN}PASS: Agent is healthy${NC}"
else
    echo -e "${RED}WARN: Agent may not be fully configured${NC}"
    echo "This is expected if LM Studio is not running"
fi
echo ""

# Test 2: Simple agent request (echo mode works without LLM)
echo -e "${YELLOW}Test 2: Simple Agent Request${NC}"
RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
    -H "Content-Type: application/json" \
    -d '{"message": "list all devices in the mesh"}')

echo "Response:"
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"

if echo "$RESPONSE" | grep -q '"reply"'; then
    echo -e "${GREEN}PASS: Got agent response${NC}"
else
    echo -e "${RED}FAIL: No reply in response${NC}"
fi
echo ""

# Test 3: Check tool calls were made
echo -e "${YELLOW}Test 3: Verify Tool Calling${NC}"
TOOL_CALLS=$(echo "$RESPONSE" | jq '.tool_calls // []' 2>/dev/null)
ITERATIONS=$(echo "$RESPONSE" | jq '.iterations // 0' 2>/dev/null)

echo "Iterations: ${ITERATIONS}"
echo "Tool calls: ${TOOL_CALLS}"

if [ "$ITERATIONS" -gt 0 ]; then
    echo -e "${GREEN}PASS: Agent completed in ${ITERATIONS} iteration(s)${NC}"
else
    echo -e "${YELLOW}INFO: Agent completed without tool calls (may be using echo provider)${NC}"
fi
echo ""

# Test 4: Test with specific command request
echo -e "${YELLOW}Test 4: Shell Command Request${NC}"
RESPONSE=$(curl -s -X POST "${BASE_URL}/api/agent" \
    -H "Content-Type: application/json" \
    -d '{"message": "run df -h on any available device and show me disk usage"}')

echo "Response:"
echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"

REPLY=$(echo "$RESPONSE" | jq -r '.reply // ""' 2>/dev/null)
if [ -n "$REPLY" ]; then
    echo -e "${GREEN}PASS: Got reply for shell command request${NC}"
else
    echo -e "${YELLOW}WARN: Empty reply${NC}"
fi
echo ""

# Summary
echo "========================================="
echo "Smoke Test Complete"
echo "========================================="
echo ""
echo "To run with a real LLM:"
echo "  1. Start LM Studio and load a model (e.g., qwen3-vl-8b)"
echo "  2. Start server: make server"
echo "  3. Start web: CHAT_PROVIDER=openai CHAT_BASE_URL=http://localhost:1234 CHAT_MODEL=qwen3-vl-8b make web"
echo "  4. Run this script again"
echo ""
echo "Example curl request:"
echo "  curl -X POST http://localhost:8080/api/agent \\"
echo "    -H 'Content-Type: application/json' \\"
echo "    -d '{\"message\": \"show me disk usage on any device\"}'"
