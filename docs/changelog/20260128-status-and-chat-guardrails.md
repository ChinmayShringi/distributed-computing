# Status and Chat Guardrails - 2026-01-28

## Summary

This release adds loop guards to prevent infinite tool execution in chat mode, makes `--allow-dangerous` a global flag, and expands `omniforge status` with comprehensive connectivity checks.

## Changes

### Chat Loop Guards (Critical Bug Fix)

**Problem:** The chat command could get stuck in infinite loops when the AI repeatedly called the same tool (e.g., `check_hostagent_server`) with identical arguments.

**Solution:**
- Added per-turn execution budget: max 6 tool calls per user message
- Added duplicate detection: same tool+args blocked after 2 executions
- When budget exhausted, displays summary of executed tools and suggests next actions
- Blocked tool calls are recorded and reported to the AI for better decision-making

**Files:**
- `internal/chat/budget.go` (new) - ExecutionBudget tracker with deduplication

### Global Flags

**Problem:** `--allow-dangerous` was defined on individual commands (chat, doctor), not visible in `omniforge --help`.

**Solution:**
- Moved `--allow-dangerous`, `--ad`, `--yes/-y` to root persistent flags
- These flags now appear in `omniforge --help` output
- Added `omniforge debug flags` command to print resolved flag values for debugging

**Files:**
- `cmd/spotlight/commands/root.go` - Added persistent flags
- `cmd/spotlight/commands/debug.go` - Added debugFlagsCmd

### Status Command Expansion

**Problem:** `omniforge status` only showed local Docker service status. Users needed visibility into connectivity to all key components.

**Solution:**
- Added connectivity checks for:
  - Auth: Token validity
  - Server: API health and `/v1/server/status`
  - Artifacts: Manifest endpoint reachability
  - MCP: Knowledge server connectivity (via server `/v1/mcp/status`)
  - Docker: Daemon accessibility via Unix socket
  - Hostagent: Installation, port 8998 accessibility, compose services
- Added `--json` flag for machine-readable output
- Per-check timeout (5 seconds) prevents hanging
- Shows "UNKNOWN (not configured)" when endpoints not set
- Help text documents config keys used

**Files:**
- `internal/netcheck/netcheck.go` (new) - HTTP/TCP/Unix socket check utilities
- `internal/status/status.go` (new) - Status runner and check implementations
- `cmd/spotlight/commands/root.go` - Expanded statusCmd

### Intent Mapping Improvements

**Problem:** Inconsistent handling of user intent phrases like "setup hostagent", "setup spotlight".

**Solution:**
- Added synonym mappings: "hostagent", "host agent", "hostagent-server", "host-agent" all map to hostagent server setup
- Added "setup spotlight", "setup omni" mappings to full OMNI stack setup
- Added consent requirement: AI must ask before running install operations
- Added loop avoidance instructions in system prompt

**Files:**
- `server/internal/openai/planner.go` - Updated system prompt with intent mappings and consent requirements

## Manual Test Steps (Ubuntu)

### Test 1: Help shows --allow-dangerous
```bash
omniforge --help
# Should show: --allow-dangerous, --ad, --yes in global flags section
```

### Test 2: debug flags command
```bash
omniforge --allow-dangerous debug flags
# Should show:
#   --allow-dangerous: true
#   Dangerous Mode:    true
```

### Test 3: Status with connectivity
```bash
omniforge status
# Should show categorized checks with OK/FAIL/UNKNOWN
# Shows UNKNOWN for unset endpoints

omniforge status --json
# Should output valid JSON with all check results
```

### Test 4: Chat no longer loops
```bash
omniforge chat
> setup hostagent
# Should NOT loop on check_hostagent_server indefinitely
# Should hit budget limit after ~6 tool calls and show summary
# CTRL+C should cancel promptly
```

### Test 5: Dangerous mode in chat
```bash
omniforge chat --allow-dangerous
# Should prompt for consent phrase
# After consent, dangerous mode should be active
```

## Configuration Keys Used

The status command checks the following configuration:

| Config Key | Default | Description |
|------------|---------|-------------|
| `server_url` | `http://50.17.127.28:3000` | API server base URL |
| `omni_install_path` | `/opt/hostagent-server` | Local hostagent installation path |
| `MCP_KNOWLEDGE_ADDR` (env) | `localhost:8080` | MCP server address (checked server-side) |
| `auth.access_token` | (none) | Authentication token |

## Breaking Changes

None. This release is backwards compatible.

## Deprecations

None.
