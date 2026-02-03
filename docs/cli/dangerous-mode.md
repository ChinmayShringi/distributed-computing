# Dangerous Mode

## Overview

Dangerous Mode is an opt-in execution mode that allows users to bypass safety controls for advanced troubleshooting scenarios. It enables:
- Execution of commands outside the standard tool allowlist
- Raw shell command execution
- Schema validation bypass
- Full system access with explicit user consent

## Usage

### CLI Flags (Global)

The dangerous mode flags are **global persistent flags** defined at the root command level. They appear in `omniforge --help` and are inherited by all subcommands.

```bash
# Long form
omniforge chat --allow-dangerous
omniforge doctor --allow-dangerous

# Short form
omniforge chat --ad
omniforge doctor --ad

# Auto-confirm (skip typing consent phrase)
omniforge chat --allow-dangerous --yes
omniforge chat --ad -y

# Verify flag resolution
omniforge --allow-dangerous debug flags
```

| Flag | Description |
|------|-------------|
| `--allow-dangerous` | Enable dangerous mode (global) |
| `--ad` | Alias for `--allow-dangerous` (global) |
| `-y, --yes` | Auto-confirm consent (global) |

### Verify Flags

Use `debug flags` to verify dangerous mode is enabled:

```bash
$ omniforge --allow-dangerous debug flags
Resolved Flag Values:
  --verbose:         false
  --config:          ""
  --allow-dangerous: true
  --ad:              false
  --yes:             false
  Dangerous Mode:    true
```

## Consent Flow

When invoked with `--allow-dangerous` (without `-y`), a full-screen warning appears:

1. Terminal is cleared with ANSI escape sequence
2. Red warning banner displayed with "!" borders
3. Lists what dangerous mode disables:
   - Tool allowlist
   - Argument schema validation
   - Raw bash execution blocking
4. Lists potential consequences (in RED):
   - DATA LOSS
   - SYSTEM CORRUPTION
   - SECURITY VULNERABILITIES
   - UNINTENDED SYSTEM MODIFICATIONS
5. Safety note (in YELLOW): "All actions still logged for audit purposes"
6. User must type exact phrase: `I UNDERSTAND AND ACCEPT THE RISK`
7. Exact match required - no fuzzy matching, no partial matches
8. Mistyped phrase aborts the command immediately

## Architecture

### ExecutionMode Enum

```go
type ExecutionMode int

const (
    ModeSafe      ExecutionMode = 0  // Default - enforces all safety controls
    ModeDangerous ExecutionMode = 1  // Bypasses safety controls
)
```

### ModeContext Struct

```go
type ModeContext struct {
    Mode             ExecutionMode
    ConsentTimestamp time.Time      // When user consented
    ConsentHash      string         // SHA-256 tamper-proof hash
    UserUID          string         // Current user's UID
    CLIVersion       string         // CLI version at time of consent
    OS               string         // Operating system
    Arch             string         // CPU architecture
}
```

### Consent Hash Computation

SHA-256 hash computed from:
- Required phrase (`I UNDERSTAND AND ACCEPT THE RISK`)
- Consent timestamp (RFC3339Nano format)
- User UID
- CLI version

Creates a tamper-proof audit trail that can be verified server-side.

## Tool Execution Changes

| Aspect | SAFE Mode | DANGEROUS Mode |
|--------|-----------|----------------|
| Tool allowlist | Enforced | Bypassed |
| Schema validation | Enforced | Bypassed |
| Raw bash execution | Blocked | Allowed |
| Approval prompts | Per-tool | None (pre-consented) |
| Tool execution method | `Execute()` | `ExecuteWithMode()` |

### Chat Mode Behavior

**In SAFE Mode:**
- Each tool execution triggers `approval.PromptApproval()`
- User can: approve, always-allow, reject, or cancel
- Rejected commands send feedback back to AI

**In DANGEROUS Mode:**
- Skips all per-tool approval prompts
- Validates tool exists in registry OR is "bash"
- Executes directly without additional confirmation

### Bash Command Support

In dangerous mode, the AI can emit raw bash commands:
```json
{"tool": "bash", "args": {"command": "systemctl restart docker"}, "why": "Restart Docker daemon"}
```

## Server Integration

### PlanRequest Changes

The `ExecutionMode` field is passed from client to server:
```go
type PlanRequest struct {
    // ... other fields
    ExecutionMode string `json:"execution_mode"` // "SAFE" or "DANGEROUS"
}
```

### System Prompt Modifications

When `ExecutionMode == "DANGEROUS"`, the planner appends:

```
DANGEROUS MODE ENABLED:
- You may use the "bash" tool to execute raw shell commands
- Schema validation is disabled on the client
- Allowlist enforcement is bypassed
- The user has explicitly consented to this mode
- You have full system access - be effective but careful
- All actions are still logged for audit purposes
```

**Important:** The server NEVER executes commands - it only generates plans. Execution is ALWAYS local on the client.

## Audit Logging

### Session Logs

All dangerous mode sessions include:
- `DANGEROUS_MODE=true` marker in logs
- Full command transcript with timestamps
- SHA-256 consent hash for tamper verification

### Incident Bundles

When creating an incident during a dangerous mode session:

**IncidentMetadata:**
```go
type IncidentMetadata struct {
    // ... other fields
    DangerousMode bool `json:"dangerous_mode"`
}
```

**dangerous_mode.json file created in bundle:**
```json
{
  "enabled": true,
  "consent_timestamp": "2026-01-28T12:00:00Z",
  "consent_hash": "abc123...",
  "user_uid": "501",
  "cli_version": "0.3.0",
  "os": "darwin",
  "arch": "arm64"
}
```

## Security Invariants

1. **Backend Never Executes** - Server only generates plans, never runs commands
2. **Local Execution Only** - Commands execute on client machine only
3. **Secrets Never Logged** - `redact.go` continues to apply secret filtering
4. **Exact Phrase Matching** - No fuzzy matching prevents accidental activation
5. **No AI-Initiated Activation** - Only user CLI flag can enable; AI cannot suggest it
6. **Full Audit Trail** - Complete transcript + consent metadata in logs and bundles
7. **Irreversible Session Consent** - Once phrase is typed, mode is active for entire session

## Testing

### Test Consent Flow

```bash
./omniforge chat --allow-dangerous
```
- Full-screen warning should appear
- Mistyped phrase should abort
- Correct phrase should start session with `[DANGEROUS MODE ENABLED]` banner

### Test Execution Without Approval

```bash
./omniforge chat --allow-dangerous
# After consent:
You: run ls -la
```
- Command should execute WITHOUT per-command approval prompt

### Test Doctor Mode

```bash
./omniforge doctor --allow-dangerous
```
- Consent flow should appear
- Dangerous mode indicator shown
- Tool execution without per-tool confirmation

### Test Incident Bundle

```bash
./omniforge chat --allow-dangerous
# Create incident during session
```
- Bundle should contain `dangerous_mode.json` with full consent metadata

### Test Auto-Confirm

```bash
./omniforge chat --ad -y
```
- Dangerous mode should enable WITHOUT prompting for confirmation

## When to Use

**Appropriate use cases:**
- Troubleshooting issues requiring commands outside the standard tool allowlist
- Advanced system administration tasks
- Debugging edge cases not covered by built-in tools

**Never use when:**
- You don't understand what commands the AI might run
- On production systems without proper backups
- If you're uncomfortable with potential system modifications

## Implementation Files

| File | Purpose |
|------|---------|
| `internal/mode/mode.go` | ExecutionMode enum and ModeContext |
| `internal/mode/consent.go` | Consent prompt UI |
| `cmd/spotlight/commands/root.go` | CLI flag integration |
| `internal/tools/registry.go` | `ExecuteWithMode()` method |
| `internal/doctor/doctor.go` | `NewDoctorWithMode()` |
| `internal/bundle/incident.go` | `CreateWithMode()` + dangerous_mode.json |
| `internal/ui/render.go` | `RenderDangerousModeIndicator()` |
| `server/internal/openai/planner.go` | System prompt adjustments |
