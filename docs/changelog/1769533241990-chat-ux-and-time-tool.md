# Changelog: Chat UX Improvements and System Time Tool

**Date:** 2026-01-27
**Type:** Feature Enhancement

## Summary

Improved the CLI chat experience with better progress indicators, request timeout handling, and clean CTRL+C cancellation. Added a new `get_system_time` tool that allows the AI assistant to retrieve the current system date/time with user approval.

## Changes Overview

### Modified Files

| File | Changes |
|------|---------|
| `cmd/spotlight/commands/root.go` | Enhanced chat command with timeout, spinner, CTRL+C handling, and tool approval prompts |
| `internal/tools/registry.go` | Added `GetSystemTimeTool` for retrieving system date/time |

### New Features

#### 1. Improved Spinner with Elapsed Time

The spinner now shows elapsed time after 2 seconds of waiting:
```
⠹ Thinking... (5s)
```

#### 2. Request Timeout (60 seconds)

Chat requests now timeout after 60 seconds with a clear error message:
```
Error: request timed out after 1m0s. Check server connectivity or retry. Use --debug for more info
```

#### 3. Clean CTRL+C Cancellation

Pressing CTRL+C during a pending request now:
- Cancels the in-flight request
- Prints "Interrupted. Cancelling request..."
- Exits gracefully

#### 4. Tool Execution with Approval Prompts

Before executing any tool (including bash commands), the user is prompted:
```
🔧 Tool: get_system_time
   Description: Get the current system date and time
   Execute this tool? [y/N]:
```

#### 5. New `get_system_time` Tool

A safe tool for retrieving the current system date/time:

| Property | Value |
|----------|-------|
| Name | `get_system_time` |
| Dangerous | `false` |
| Args | `format` (optional string) |

**Behavior by platform:**
- **Linux/macOS:** Runs `date` command (with optional format like `+%Y-%m-%d`)
- **Windows:** Runs `powershell -Command Get-Date`

## Code Details

### Spinner State Tracking

```go
type spinnerState struct {
    message string
    elapsed time.Duration
    done    bool
    mu      sync.Mutex
}
```

### Chat with Timeout

```go
func chatWithTimeout(ctx context.Context, client *api.Client,
    messages []api.ChatMessage, systemContext string) (*api.ChatResponse, error) {
    timeoutCtx, cancel := context.WithTimeout(ctx, chatTimeout)
    defer cancel()
    // ... timeout handling
}
```

### Tool Approval Prompt

```go
func promptToolApproval(toolName, description string) bool {
    fmt.Printf("\n🔧 Tool: %s\n", toolName)
    fmt.Printf("   Description: %s\n", description)
    fmt.Print("   Execute this tool? [y/N]: ")
    // ... read user input
}
```

## Testing

### Prerequisites

1. Build the CLI:
```bash
go build -o omniforge ./cmd/spotlight
```

2. Login (if not already):
```bash
./omniforge login
```

### Test 1: Observe Spinner

```bash
./omniforge chat
# Type: hello
# Observe: spinner shows "⠹ Thinking..."
# After 2 seconds, shows "⠹ Thinking... (3s)"
```

**Expected behavior:**
```
You: hello

⠹ Thinking... (3s)
Assistant: Hello! How can I help you today?
```

### Test 2: Ask for System Date

```bash
./omniforge chat
# Type: what day is it today?
```

**Expected interaction:**
```
You: what day is it today?

⠹ Thinking...
Assistant: I can check the current date for you. Let me run a command to get it.

🔧 Tool: get_system_time
   Description: Get the current system date and time
   Execute this tool? [y/N]: y
   Executing get_system_time...
   Result: Mon Jan 27 12:34:56 EST 2026

⠹ Thinking...
Assistant: Today is Monday, January 27, 2026.
```

### Test 3: Decline Tool Execution

```bash
./omniforge chat
# Type: what time is it?
# When prompted, type: n
```

**Expected behavior:**
```
You: what time is it?

⠹ Thinking...
Assistant: I can check the current time for you.

🔧 Tool: get_system_time
   Description: Get the current system date and time
   Execute this tool? [y/N]: n
   Skipped.

⠹ Thinking...
Assistant: I cannot determine the current time without running a local command.
Would you like me to try again?
```

### Test 4: CTRL+C Cancellation

```bash
./omniforge chat
# Type: tell me a long story
# Press CTRL+C while "Thinking..."
```

**Expected behavior:**
```
You: tell me a long story

⠹ Thinking... (2s)
^C
Interrupted. Cancelling request...
```

### Test 5: Request Timeout

To test timeout (requires slow/unresponsive server):
```bash
# With server that takes >60s to respond
./omniforge chat
# Type: hello
# Wait 60 seconds
```

**Expected behavior:**
```
You: hello

⠹ Thinking... (60s)
Error: request timed out after 1m0s. Check server connectivity or retry.
```

### Test 6: Single Message Mode

```bash
./omniforge chat -m "what is the current date?"
# Approve tool execution when prompted
```

## Terminal Demo

### Before (old behavior)
```
You: hello
⠹ Thinking...        <- Static spinner, no elapsed time
Assistant: Hello!

You: what day is it?
⠹ Thinking...
> Running: date       <- No approval prompt, runs immediately
Mon Jan 27 12:34:56 EST 2026
```

### After (new behavior)
```
You: hello
⠹ Thinking... (3s)   <- Shows elapsed time after 2s
Assistant: Hello!

You: what day is it?
⠹ Thinking... (2s)
Assistant: Let me check the current date.

🔧 Tool: get_system_time
   Description: Get the current system date and time
   Execute this tool? [y/N]: y     <- Approval prompt
   Executing get_system_time...
   Result: Mon Jan 27 12:34:56 EST 2026

⠹ Thinking...
Assistant: Today is Monday, January 27, 2026.
```

## Security Considerations

1. **Tool Approval Required:** All tool executions require explicit user approval
2. **Safe Time Tool:** `get_system_time` is marked as non-dangerous and only runs `date` or `Get-Date`
3. **Request Cancellation:** Users can cancel pending requests with CTRL+C
4. **Timeout Protection:** Requests timeout after 60 seconds to prevent hanging

## Migration Notes

- No database changes required
- No configuration changes needed
- Backwards compatible with existing chat functionality
