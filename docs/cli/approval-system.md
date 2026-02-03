# Approval System

## Overview

The Approval System provides interactive confirmation for tool execution during chat sessions. It assesses risk, displays tool details, and allows users to create persistent rules for frequently-used commands.

## Features

- **Risk Assessment**: Categorizes tools as LOW, MEDIUM, or HIGH risk
- **Interactive UI**: Clear display of tool, arguments, and reasoning
- **Persistent Rules**: Save "always allow" rules for convenience
- **Feedback Loop**: Provide alternative instructions when rejecting
- **Dangerous Command Detection**: Warns about destructive operations

## Usage

When a tool execution is requested:

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

## Approval Options

| Option | Key | Description |
|--------|-----|-------------|
| **Approve** | `Y` | Execute this tool once |
| **Always Allow** | `A` | Create rule to auto-approve in future |
| **Feedback** | `F` | Reject and provide alternative instructions |
| **Reject** | `N` | Skip this tool execution |
| **Cancel** | `C` | Cancel the entire operation |

## Risk Levels

| Level | Color | Criteria |
|-------|-------|----------|
| LOW | Green | Read-only checks (check_docker, check_ports) |
| MEDIUM | Yellow | Non-destructive modifications |
| HIGH | Red | Dangerous tools (restart_*, install_*) |

## Dangerous Command Detection

The system detects dangerous patterns in commands:

| Pattern | Warning |
|---------|---------|
| `rm -rf` | Recursive delete |
| `mkfs` | Filesystem format |
| `dd` | Low-level disk write |
| `curl\|sh` | Remote code execution |
| `chmod 777` | Insecure permissions |
| `shutdown`, `reboot` | System control |
| `killall` | Process termination |

## Persistent Rules

### Rule Types

| Type | Description | Example |
|------|-------------|---------|
| `prefix` | Match command prefix | `docker ` matches all docker commands |
| `exact` | Match exact command | `docker ps` only |
| `path` | Match within directory | `/home/user/project/` |

### Rule Storage

Rules are stored in `~/.omniforge/approvals.json`:

```json
{
  "rules": [
    {
      "type": "prefix",
      "pattern": "docker ",
      "created_at": "2026-01-28T12:00:00Z",
      "usage_count": 15
    },
    {
      "type": "exact",
      "pattern": "check_docker",
      "created_at": "2026-01-28T12:05:00Z",
      "usage_count": 3
    }
  ]
}
```

### Creating Rules

When selecting "Always Allow", the system suggests rule scopes:

```
Choose rule scope:
  [1] Command prefix: "docker "
  [2] Exact command: "docker ps"
  [3] Current directory: "/home/user/project/"
```

## Feedback Loop

When selecting "Feedback", the user can provide alternative instructions:

```
[F] Feedback selected
Enter alternative instructions for the AI:
> Instead of restarting, check the logs first
```

This feedback is sent back to the AI to adjust its approach.

## Architecture

```go
type ApprovalPrompt struct {
    Tool       string
    Args       map[string]interface{}
    Why        string
    RiskLevel  RiskLevel
}

type Decision int
const (
    DecisionYes Decision = iota      // Approve once
    DecisionAlwaysAllow              // Create rule
    DecisionNo                       // Reject
    DecisionCancel                   // Cancel operation
    DecisionFeedback                 // Provide feedback
)

type Rule struct {
    Type       RuleType  // prefix, exact, path
    Pattern    string    // Match pattern
    CreatedAt  time.Time
    UsageCount int
}
```

## Bypass in Dangerous Mode

When dangerous mode is enabled (`--allow-dangerous`):

1. User provides consent at session start
2. All per-tool approvals are skipped
3. Tools execute immediately
4. Full audit logging still applies

## Implementation Files

| File | Purpose |
|------|---------|
| `internal/approval/approval.go` | Approval prompt UI |
| `internal/approval/rules.go` | Rule storage and matching |
| `internal/approval/risk.go` | Risk assessment |
| `internal/approval/dangerous.go` | Dangerous pattern detection |
