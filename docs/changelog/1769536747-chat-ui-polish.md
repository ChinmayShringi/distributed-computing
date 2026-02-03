# Changelog: Chat UI Polish

**Date:** 2026-01-27T18:49:53Z
**Type:** Enhancement

## Summary

Upgraded `omniforge chat` CLI to a polished terminal experience with styled header, colored role prefixes, spinners, and improved tool approval action cards.

## Features

### Welcome Header Panel
- Boxed header panel on chat start with username, server URL, and workspace
- Styled with Unicode box drawing characters
- Cyan color theme for borders

### Colored Role Prefixes
- "You:" in bold green
- "Assistant:" in bold blue
- System messages in dim text

### Spinner Animation
- Braille spinner animation during API calls
- Shows elapsed time after 2 seconds
- Reusable `ui.Spinner` component

### Tool Approval Action Card
- Styled action approval card with yellow border
- Shows command, working directory, sudo status, and risk level
- Risk level color-coded (green=LOW, yellow=MEDIUM, red=HIGH)
- Clear numbered options menu

### Color Control
- `--no-color` flag to disable colored output
- Respects `NO_COLOR` environment variable (https://no-color.org/)
- Automatic fallback for non-TTY environments

## Files Changed

| File | Type | Description |
|------|------|-------------|
| `internal/ui/theme.go` | New | Color codes, box characters, TTY detection |
| `internal/ui/render.go` | New | Header panel and message rendering |
| `internal/ui/spinner.go` | New | Reusable spinner component |
| `internal/ui/actioncard.go` | New | Tool approval action card |
| `cmd/spotlight/commands/root.go` | Updated | Integrated UI package, added --no-color flag |
| `internal/approval/approval.go` | Updated | Use new action card UI |

## How to Test

```bash
# Build the CLI
go build -o omniforge ./cmd/spotlight

# Test interactive chat
./omniforge chat

# Verify:
# 1. Boxed header appears with user/server info
# 2. "You:" prompt is green
# 3. "Assistant:" responses are blue
# 4. Spinner animates during API call
# 5. Type a command that triggers tool approval
# 6. Action card appears with styled border

# Test --no-color flag
./omniforge chat --no-color
# Verify: No colors, plain text output

# Test NO_COLOR env var
NO_COLOR=1 ./omniforge chat
# Verify: Same as --no-color

# Test non-interactive (single message)
./omniforge chat -m "what time is it"
```

## Docker/Ubuntu Container Test

```bash
# Run Ubuntu container
docker run -it --rm ubuntu:22.04 bash

# Inside container:
apt update && apt install -y curl

# Download binary
curl -sSL http://YOUR_SERVER/cli/linux/amd64 -o omniforge
chmod +x omniforge

# Login
./omniforge login -u admin

# Test chat
./omniforge chat

# Expected:
# - Header renders correctly with box characters
# - Colors work in terminal
# - Spinner animation works
# - Action cards display properly
```

## SSH Compatibility

The UI is SSH-compatible:
- Uses standard ANSI escape codes
- Box characters are Unicode (widely supported)
- Graceful fallback to plain text for non-TTY
- No cursor movement beyond line clearing
