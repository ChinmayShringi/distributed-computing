# Changelog: Interactive Approval UI and Feedback Loop

**Date:** 2026-01-27
**Type:** Feature Enhancement

## Summary

Implemented an interactive approval UI for tool execution in `omniforge chat`, similar to Claude Code. Users can now approve, always-allow, provide feedback, or cancel tool actions. The system persists "always allow" rules and supports a feedback loop where users can redirect the assistant's approach.

## Features

### 1. Interactive Approval Prompt

When the assistant proposes a tool action, users see:

```
┌─────────────────────────────────────────────────────────────
│ Tool: bash
├─────────────────────────────────────────────────────────────
│ Command: ls -la /tmp
│ Working Directory: /home/user/project
│ Needs sudo: no
│ Risk: LOW (read-only / safe)
├─────────────────────────────────────────────────────────────
│ Rationale: Execute shell command
└─────────────────────────────────────────────────────────────

Do you want to proceed?
  1) Yes (run once)
  2) Yes, and always allow this command
  3) No, and tell the assistant what to do differently
  4) Cancel / skip this action

Choice [1-4]:
```

### 2. Risk Assessment

Commands are automatically assessed for risk level:

| Risk Level | Description | Examples |
|------------|-------------|----------|
| LOW | Read-only / safe operations | `ls`, `cat`, `echo`, `date` |
| MEDIUM | System modifications | `sudo`, `rm`, `chmod`, `apt install` |
| HIGH | Potentially destructive | `rm -rf`, `mkfs`, `dd`, `curl \| bash` |

### 3. "Always Allow" Rules

When users choose option 2, they can select a scope:

```
Choose what to allow:
  1) Commands starting with 'ls'
  2) This exact command
  3) Commands operating in '/tmp'

Scope [1]:
```

Rules are persisted in `~/.omniforge/approvals.json`:

```json
{
  "rules": [
    {
      "type": "prefix",
      "pattern": "docker",
      "tool": "bash",
      "created_at": "2026-01-27T12:00:00Z",
      "used_count": 5
    },
    {
      "type": "path",
      "pattern": "/tmp",
      "tool": "bash",
      "created_at": "2026-01-27T12:00:00Z",
      "used_count": 3
    }
  ]
}
```

### 4. Feedback Loop (Option 3)

When users reject an action and provide feedback:
1. User types alternative instructions
2. Message sent to assistant: "I did not approve the action 'bash: rm file.txt'. Do this differently: use /tmp instead"
3. Assistant proposes a new approach
4. Loop continues until user approves or cancels

### 5. Improved CTRL+C Handling

- First CTRL+C: Cancels current request, shows "Press CTRL+C again to force exit..."
- Second CTRL+C: Immediately exits the program

## Files Changed

| File | Changes |
|------|---------|
| `internal/approval/approval.go` | New - Approval UI, risk assessment, action analysis |
| `internal/approval/rules.go` | New - Rules persistence and matching logic |
| `internal/approval/rules_test.go` | New - Unit tests for rule matching |
| `cmd/spotlight/commands/root.go` | Updated - Integrated approval UI, fixed CTRL+C |
| `.github/workflows/deploy.yml` | Updated - Fixed server binary deployment |

## Config File Schema

`~/.omniforge/approvals.json`:

```json
{
  "rules": [
    {
      "type": "prefix|exact|path",
      "pattern": "string",
      "tool": "bash|get_system_time|...",
      "created_at": "ISO8601 timestamp",
      "used_count": 0
    }
  ]
}
```

## Rule Types

| Type | Description | Example Pattern |
|------|-------------|-----------------|
| `prefix` | Match commands starting with pattern | `docker`, `git`, `ls` |
| `exact` | Match exact command string | `cat /etc/hosts` |
| `path` | Match commands operating in directory | `/tmp`, `~/Documents` |

## How to Test

### Test 1: Basic Approval Flow

```bash
./omniforge chat
# Type: create a test file in /tmp
# Should show approval prompt with 4 options
# Choose 1 to run once
```

### Test 2: Always Allow

```bash
./omniforge chat
# Type: list files in /tmp
# Choose 2 to always allow
# Select scope (e.g., "Commands starting with 'ls'")
# Next time, same command auto-approves with "[Auto-approved by saved rule]"
```

### Test 3: Feedback Loop

```bash
./omniforge chat
# Type: create a file in ~/Documents
# When prompted, choose 3
# Type: do it in /tmp instead
# Assistant should propose /tmp alternative
```

### Test 4: CTRL+C Handling

```bash
./omniforge chat
# Type: hello
# While "Thinking...", press CTRL+C once
# Should show "Press CTRL+C again to force exit..."
# Press CTRL+C again
# Should immediately exit
```

### Test 5: Risk Assessment

```bash
./omniforge chat
# Type: delete all files with rm -rf
# Should show Risk: HIGH (potentially destructive)
# Choose 4 to cancel
```

## Demo Transcript

```
You: create a test file called hello.txt in /tmp

⠹ Thinking...
Assistant: I'll create that file for you.

┌─────────────────────────────────────────────────────────────
│ Tool: bash
├─────────────────────────────────────────────────────────────
│ Command: echo "Hello, World!" > /tmp/hello.txt
│ Working Directory: /home/ubuntu
│ Needs sudo: no
│ Risk: LOW (read-only / safe)
├─────────────────────────────────────────────────────────────
│ Rationale: Execute shell command
└─────────────────────────────────────────────────────────────

Do you want to proceed?
  1) Yes (run once)
  2) Yes, and always allow this command
  3) No, and tell the assistant what to do differently
  4) Cancel / skip this action

Choice [1-4]: 2

Choose what to allow:
  1) Commands starting with 'echo'
  2) This exact command
  3) Commands operating in '/tmp'

Scope [1]: 3
   Rule saved for future commands.
   Running: echo "Hello, World!" > /tmp/hello.txt

⠹ Thinking...
Assistant: I've created the file /tmp/hello.txt with the content "Hello, World!".

You: now cat that file

⠹ Thinking...
Assistant: I'll read the file for you.

[Auto-approved by saved rule]
   Running: cat /tmp/hello.txt
   Output:
Hello, World!

⠹ Thinking...
Assistant: The file contains: "Hello, World!"
```

## Security Considerations

1. **Risk Labels**: High-risk commands are clearly marked
2. **No Default Allow**: Each command requires explicit approval
3. **Scoped Rules**: Users choose what scope to allow, preventing overly broad permissions
4. **Feedback Loop**: Users can redirect dangerous operations to safer alternatives
5. **Persistent Rules**: Stored with restricted permissions (0600)

## Migration Notes

- No database changes required
- New config file `~/.omniforge/approvals.json` created on first use
- Backwards compatible with existing chat functionality
- Old simple approval prompts replaced with interactive UI
