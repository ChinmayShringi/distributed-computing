# Chat Command

## Overview

The `chat` command provides an interactive AI assistant for troubleshooting, system administration, and knowledge queries. It supports tool execution with approval workflows and integrates with the knowledge base.

## Usage

```bash
# Interactive chat session
omniforge chat

# Single message mode
omniforge chat -m "How do I restart Docker?"

# Disable color output
omniforge chat --no-color

# Dangerous mode (bypass safety controls)
omniforge chat --allow-dangerous
omniforge chat --ad

# Auto-confirm dangerous mode
omniforge chat --ad -y
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `-m` | | Single message mode (non-interactive) |
| `--no-color` | | Disable colored output |
| `--allow-dangerous` | `--ad` | Enable dangerous mode |
| `--yes` | `-y` | Auto-confirm dangerous mode consent |

## Interactive Commands

Within a chat session:

| Command | Description |
|---------|-------------|
| `!<command>` | Execute shell command directly |
| `clear` | Clear conversation history |
| `exit` | Exit chat session |
| `quit` | Exit chat session |
| `Ctrl+C` | Exit chat session |

## Features

### Tool Execution

The AI can request tool execution. In SAFE mode, each tool requires approval:

```
┌─ Tool Execution Request ─────────────────────────┐
│                                                  │
│  Tool: restart_service                           │
│  Risk: HIGH                                      │
│                                                  │
│  Arguments:                                      │
│    service: keycloak                             │
│                                                  │
│  Reason: Keycloak is not responding, restarting  │
│          may resolve the connectivity issue.     │
│                                                  │
│  [Y] Approve  [A] Always Allow  [F] Feedback     │
│  [N] Reject   [C] Cancel                         │
└──────────────────────────────────────────────────┘
```

### Approval Options

| Option | Key | Description |
|--------|-----|-------------|
| Approve | `Y` | Execute this time only |
| Always Allow | `A` | Create rule to always allow |
| Feedback | `F` | Provide alternative instructions |
| Reject | `N` | Skip this tool |
| Cancel | `C` | Cancel entire operation |

### Knowledge Base Integration

Chat automatically searches the knowledge base for relevant context:

- Documentation search via RAG
- Source attribution in responses
- Real-time document status

### UI Elements

- **ASCII header** with Unicode borders
- **Colored role prefixes**: You (cyan), Assistant (green), System (yellow)
- **Spinner** with elapsed time during thinking
- **Risk color coding**: LOW (green), MEDIUM (yellow), HIGH (red)

## Dangerous Mode

When enabled with `--allow-dangerous`:

1. Full-screen consent warning displayed
2. User must type exact phrase: `I UNDERSTAND AND ACCEPT THE RISK`
3. Tool allowlist bypassed
4. Schema validation bypassed
5. Raw bash execution allowed
6. No per-tool approval prompts

See [dangerous-mode.md](dangerous-mode.md) for full details.

## Architecture

### Message Flow

```
User Input
    │
    ▼
┌──────────────┐
│ Chat Client  │──────► Server API
└──────────────┘             │
    ▲                        ▼
    │               ┌────────────────┐
    │               │ AI Processing  │
    │               │ + RAG Context  │
    │               └────────────────┘
    │                        │
    │                        ▼
    │               ┌────────────────┐
    │               │ Tool Request?  │
    │               └────────────────┘
    │                   │      │
    │              Yes  │      │ No
    │                   ▼      │
    │           ┌───────────┐  │
    │           │ Approval  │  │
    │           │   Flow    │  │
    │           └───────────┘  │
    │                   │      │
    │                   ▼      ▼
    └───────────── Response ◄──┘
```

### Server-Side Tools

Executed on server, no client approval needed:
- `knowledge_search` - RAG search
- `knowledge_list` - List documents
- `knowledge_status` - Document status

### Client-Side Tools

Executed locally with approval:
- `bash` - Shell commands (dangerous mode only)
- `check_docker` - Docker status
- `check_hostagent_server` - HostAgent status
- `setup_hostagent_server` - Setup via artifact download
- `verify_hostagent_server` - Verification

## Configuration

### Approval Rules Storage

Rules are persisted in `~/.omniforge/approvals.json`:

```json
{
  "rules": [
    {
      "type": "prefix",
      "pattern": "docker ",
      "created_at": "2026-01-28T12:00:00Z",
      "usage_count": 5
    }
  ]
}
```

## Execution Budget (Loop Guards)

To prevent infinite tool execution loops, the chat command implements an execution budget system.

### Limits

| Limit | Default | Description |
|-------|---------|-------------|
| Max tool calls per turn | 6 | Maximum total tool calls per user message |
| Max repeated same tool | 2 | Maximum times same tool+args can be called |

### How It Works

1. **Per-Turn Budget**: Each user message starts a new budget
2. **Deduplication**: Tool calls are deduplicated by `tool_name + normalized_args`
3. **Blocking**: Exceeding limits blocks further tool execution
4. **Summary**: When budget exhausted, a summary is displayed

### Budget Exhausted Output

```
Execution budget exhausted (6 tool calls reached)

Tools executed this turn:
  1. check_hostagent_server -> {"installed":false}
  2. check_hostagent_server -> (blocked: duplicate blocked)
  3. check_docker -> {"running":true}
  ...

Suggested next steps:
  - Run `omniforge prereq docker --install` to install Docker
  - Run `omniforge debug installers` to debug script discovery
  - Run `omniforge doctor` for automated troubleshooting
```

### Implementation

The budget is implemented in `internal/chat/budget.go`:

```go
type ExecutionBudget struct {
    MaxToolCallsPerTurn int            // Default: 6
    MaxRepeatedSameTool int            // Default: 2
    toolExecutions      map[string]int // key: dedupe key
    executionLog        []ToolExecution
}
```

This prevents scenarios where the AI repeatedly calls the same tool without making progress.

## Error Handling

| Error | Cause | Solution |
|-------|-------|----------|
| "Not authenticated" | No valid session | Run `omniforge login` |
| "Tool not found" | Unknown tool requested | Check tool registry |
| "Tool not allowed" | Blocked by allowlist | Use dangerous mode or contact admin |
| "budget exhausted" | Too many tool calls | Follow suggested next steps |
| "duplicate blocked" | Same tool+args called twice | AI should try different approach |
| Connection error | Network issue | Check connectivity |

## Implementation Files

| File | Purpose |
|------|---------|
| `cmd/spotlight/commands/root.go` | Chat command and loop |
| `internal/chat/budget.go` | Execution budget (loop guards) |
| `internal/approval/approval.go` | Approval UI |
| `internal/approval/rules.go` | Rule persistence |
| `internal/ui/render.go` | UI styling |
| `internal/api/chat.go` | Chat API client |
