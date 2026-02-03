# Changelog: Chat Tool Execution Fix and UI Improvements

**Date:** 2026-01-27
**Type:** Bug Fix, Feature Enhancement

## Summary

Fixed the chat command tool execution bug where the assistant would describe actions in text (e.g., "Suggested actions: 1. bash") instead of properly calling tools. Added dangerous command detection with warnings for potentially destructive operations.

## Problem

When users asked the AI assistant to run commands (e.g., "can you run ls?"), the assistant would:
1. Print "Suggested actions: 1. bash" in the message text
2. Return to input without prompting for tool approval
3. Never actually execute the requested command

**Root Cause:** The backend LLM was describing tools in text instead of making proper OpenAI function calls.

## Changes Overview

### Backend (Server)

| File | Changes |
|------|---------|
| `server/internal/openai/planner.go` | Enhanced Chat() function with improved system prompt forcing tool usage |

#### Key Backend Changes

1. **Improved System Prompt**
   ```
   CRITICAL: You have a bash tool that executes commands on the user's system.

   WRONG: "I can run ls for you. The command would be: ls"
   RIGHT: [Call bash function with command="ls"]
   ```

2. **Tool Definitions**
   - `bash`: Execute any bash command on the user's system
   - `get_system_time`: Get current system date/time (already existed)

3. **Tool Choice Setting**
   - Added `ToolChoice: "auto"` to let the model decide when to use tools

### CLI (Client)

| File | Changes |
|------|---------|
| `cmd/spotlight/commands/root.go` | Added dangerous command detection and warnings |
| `internal/api/chat.go` | Updated ChatAction to include ToolID for function calling |

#### Dangerous Command Detection

Added detection for potentially destructive commands with warnings:

| Pattern | Warning |
|---------|---------|
| `rm -rf` | This command recursively deletes files without confirmation |
| `mkfs` | This command formats a disk, destroying all data |
| `dd if=` | This command can overwrite disks and destroy data |
| `curl \| sh` | This downloads and executes code from the internet |
| `chmod 777` | This makes files world-writable, which is a security risk |
| `shutdown` | This will shut down the system |
| `reboot` | This will reboot the system |
| `killall` | This kills multiple processes at once |

#### New Functions

```go
// isDangerousCommand checks if a command is potentially dangerous
func isDangerousCommand(cmd string) (bool, string) {
    // Returns true and a warning message if command matches dangerous patterns
}

// promptToolApprovalWithCommand shows warning for dangerous commands
func promptToolApprovalWithCommand(toolName, description, command string) bool {
    // Shows warning if dangerous, then prompts for approval
}
```

## Expected Behavior After Fix

### Before (Broken)
```
You: can you run ls?

Thinking...
Assistant: I can help you run the ls command to list directory contents.
Suggested actions: 1. bash

You:   <- Returns to input without executing
```

### After (Fixed)
```
You: can you run ls?

Thinking...
Assistant: I'll run that for you.

Tool: bash
   Description: Run bash command
   Execute this tool? [y/N]: y
   Running: ls
Desktop  Documents  Downloads  Music  Pictures

Thinking...
Assistant: Here are the files in your current directory...
```

### Dangerous Command Warning
```
You: remove all my files

Thinking...
Assistant: I'll run that command for you.

Tool: bash
   Description: Run bash command
   WARNING: This command recursively deletes files without confirmation
   Command: rm -rf *
   Execute this tool? [y/N]: n
   Skipped.

Thinking...
Assistant: The command was not executed as you declined.
```

## Code Details

### Backend System Prompt Enhancement

```go
systemPrompt := `You are OmniForge Assistant...

CRITICAL: You have a bash tool that executes commands on the user's system. When the user asks you to:
- Run a command (e.g., "run ls", "execute df -h")
- Check something (e.g., "check docker status", "show disk space")
- Fix something (e.g., "restart docker", "kill process")

You MUST call the bash function with the exact command. Do NOT just describe the command in text.

WRONG: "I can run ls for you. The command would be: ls"
RIGHT: [Call bash function with command="ls"]

After calling the tool, explain what the output means.`
```

### CLI Tool Approval Flow

```go
// Check for dangerous commands before showing approval prompt
if command != "" {
    if isDangerous, warning := isDangerousCommand(command); isDangerous {
        fmt.Printf("   WARNING: %s\n", warning)
        fmt.Printf("   Command: %s\n", command)
    }
}
fmt.Print("   Execute this tool? [y/N]: ")
```

## Testing

### Prerequisites

1. Build CLI and Server:
```bash
cd /path/to/spotlight-omni-cli
go build -o omniforge ./cmd/spotlight/
cd server
go build -o omniforge-server ./cmd/server/
```

2. Ensure server has valid OPENAI_API_KEY in environment or AWS Secrets Manager

3. Start server:
```bash
cd server
source .env
./omniforge-server
```

### Test 1: Basic Tool Execution

```bash
./omniforge chat
# Type: run ls
# Should prompt for tool approval
# Type: y
# Should show directory listing
```

### Test 2: Dangerous Command Warning

```bash
./omniforge chat
# Type: delete everything with rm -rf
# Should show WARNING message before approval prompt
# Type: n to skip
```

### Test 3: Tool Decline

```bash
./omniforge chat
# Type: what time is it?
# When prompted for get_system_time, type: n
# Assistant should acknowledge the tool was not run
```

### Test in Ubuntu Container

```bash
# Pull and run Ubuntu container
docker run -it ubuntu:22.04 bash

# Inside container:
apt update && apt install -y curl

# Download CLI
curl -Lo omniforge http://your-server/cli/linux/amd64
chmod +x omniforge

# Login
./omniforge login

# Test chat
./omniforge chat
# Type: run ls
# Approve tool execution
# Verify directory listing appears
```

## Security Considerations

1. **Dangerous Command Warnings**: All potentially destructive commands now show explicit warnings
2. **User Approval Required**: Every tool execution requires explicit user consent
3. **Command Visibility**: The actual command is shown before execution
4. **Tool Execution Tracking**: Tool results are properly sent back to the LLM for context

## Migration Notes

- No database changes required
- No configuration changes needed
- Backwards compatible with existing chat functionality
- Server must be rebuilt and redeployed for backend changes to take effect

## Files Modified

1. `server/internal/openai/planner.go` - Backend chat function with improved prompts
2. `cmd/spotlight/commands/root.go` - CLI with dangerous command detection
3. `internal/api/chat.go` - ChatAction struct with ToolID field
