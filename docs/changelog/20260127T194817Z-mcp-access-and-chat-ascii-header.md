# Changelog: MCP Access, Knowledge Tools, and ASCII Chat Header

**Date:** 2026-01-27
**Type:** Feature / UX Enhancement

## Summary

Fixed chat to properly use knowledge base tools and MCP connectivity. Added Claude Code-style ASCII art header to the CLI chat interface. The assistant no longer claims inability to access systems when tools are available.

## Changes

### A) Chat Now Has Knowledge and MCP Tools

**New Server-Side Tools (executed on server, not client):**

| Tool | Description |
|------|-------------|
| `knowledge_search` | Search the knowledge base for relevant documentation |
| `knowledge_list` | List all documents in the knowledge base |
| `knowledge_status` | Get status of a specific document |
| `mcp_status` | Check MCP server connectivity |

**Updated System Prompt:**
- Explicitly tells the assistant it HAS access to knowledge tools and MCP status
- Instructs it to NEVER claim inability when tools are available
- Provides clear examples of wrong vs right behavior

### B) New MCP Status Endpoint

Added `GET /v1/mcp/status` endpoint that returns:
```json
{
  "enabled": true/false,
  "reachable": true/false,
  "addr": "localhost:8080",
  "error": "optional error message",
  "how_to_enable": "Set environment variables..."
}
```

Environment variables:
- `MCP_ENABLED=true` - Enable MCP checking
- `MCP_KNOWLEDGE_ADDR=host:port` - MCP server address (default: localhost:8080)

### C) ASCII Art Chat Header

New Claude Code-style header with Unicode art:
```
╭──────────────────────────────────────────────────────────╮
│                                                          │
│                   Welcome back admin!                    │
│                                                          │
│                       * ▐▛███▜▌ *                        │
│                      * ▝▜█████▛▘ *                       │
│                       *  ▘▘ ▝▝  *                        │
│                                                          │
│                      OmniForge Chat                      │
│                         v0.1.0                           │
│                 http://50.17.127.28:3000                 │
│                                                          │
╰──────────────────────────────────────────────────────────╯
```

## Files Changed

| File | Changes |
|------|---------|
| `server/cmd/server/main.go` | Added `/v1/mcp/status` endpoint, added `net` import |
| `server/internal/openai/planner.go` | Updated system prompt, added 4 knowledge/mcp tools, added server-side tool execution loop, added `executeServerTool()` function |
| `internal/ui/render.go` | Added ASCII art header with Unicode logo, added `formatCenteredLine()` helper |

## How to Test

### 1. Build and Deploy

```bash
# Build server
cd server && make server

# Build CLI
cd .. && go build -o omniforge ./cmd/spotlight

# Deploy server to EC2 (or run locally)
./build/server
```

### 2. Test ASCII Header

```bash
./omniforge chat
```

Expected: See the new ASCII art header with Unicode logo.

### 3. Test MCP Status

```
You: can you access the mcp server?
```

Expected: Assistant calls `mcp_status` tool and reports:
- If MCP_ENABLED=true and reachable: "MCP server is enabled and reachable at localhost:8080"
- If MCP_ENABLED=false: "MCP is not enabled. Set MCP_ENABLED=true to enable."
- If unreachable: "MCP server at localhost:8080 is not reachable: <error>"

### 4. Test Knowledge Search

```
You: search the knowledge base for docker troubleshooting
```

Expected: Assistant calls `knowledge_search` tool and returns results from the knowledge base.

### 5. Test Knowledge List

```
You: what documents are in the knowledge base?
```

Expected: Assistant calls `knowledge_list` tool and lists available documents.

## Expected Behavior

**Before (broken):**
```
User: can you access the mcp server?
Assistant: I don't have direct access to external servers or systems like an MCP server...
```

**After (fixed):**
```
User: can you access the mcp server?
Assistant: Let me check the MCP server status.
[Calls mcp_status tool]
The MCP server status is: enabled=false. MCP is not currently enabled. To enable it, set the environment variables MCP_ENABLED=true and MCP_KNOWLEDGE_ADDR=host:port.
```
