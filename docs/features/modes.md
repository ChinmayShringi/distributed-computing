# Execution Modes

EdgeCLI operates in two modes: Safe Mode (default) and Dangerous Mode. This provides security controls while allowing power users to bypass restrictions when needed.

## Safe Mode (Default)

Safe mode enforces security controls to prevent accidental damage.

### Features

- **Command Allowlist**: Only whitelisted commands can execute
- **Schema Validation**: Arguments validated against tool schemas
- **Approval Workflows**: Interactive confirmation for dangerous operations
- **Audit Logging**: All commands logged

### Allowed Commands

Default allowlist: `ls`, `cat`, `pwd`

```go
// internal/allowlist/allowlist.go
var defaultAllowed = []string{"ls", "cat", "pwd"}
```

### Validation Flow

```
Command Request
      ↓
Allowlist Check ──→ Rejected if not allowed
      ↓
Schema Validation ──→ Rejected if invalid args
      ↓
Approval Check ──→ User confirmation if dangerous
      ↓
Execute
```

## Dangerous Mode

Dangerous mode removes all restrictions. Use with caution.

### Enabling

```bash
go run ./cmd/edgecli --allow-dangerous
# or
go run ./cmd/edgecli --ad
```

### Confirmation Required

Before entering dangerous mode, users must type the exact phrase:

```
I UNDERSTAND AND ACCEPT THE RISK
```

A full-screen warning is displayed explaining the risks.

### Features

- No command restrictions
- No schema validation
- Direct command execution
- AI can execute ANY command
- Still logged for audit

## Implementation

### Mode Context (`internal/mode/`)

```go
type ModeContext struct {
    IsDangerous bool
    ConsentGiven bool
}

func (m *ModeContext) RequireConsent() error {
    if m.IsDangerous && !m.ConsentGiven {
        return ErrConsentRequired
    }
    return nil
}
```

### Consent Flow (`internal/mode/consent.go`)

```go
func PromptDangerousConsent() (bool, error) {
    // Display warning
    ui.RenderWarning(dangerousWarningText)

    // Prompt for confirmation phrase
    fmt.Print("Type the confirmation phrase: ")
    input := readLine()

    return input == "I UNDERSTAND AND ACCEPT THE RISK", nil
}
```

### Allowlist Validation (`internal/allowlist/`)

```go
func ValidateCommand(cmd string, args []string) (*CommandSpec, error) {
    if !isAllowed(cmd) {
        return nil, fmt.Errorf("command not allowed: %s", cmd)
    }

    return &CommandSpec{
        Executable: cmd,
        Args:       args,
    }, nil
}
```

## Approval Workflows

For certain operations in safe mode, interactive approval is required.

### Approval Rules (`internal/approval/`)

```go
type ApprovalRule struct {
    Pattern     string
    Description string
    Level       ApprovalLevel
}

const (
    ApprovalNone     ApprovalLevel = iota
    ApprovalSimple   // Single confirmation
    ApprovalExplicit // Type command to confirm
)
```

### Example Flow

```
Tool wants to delete file
      ↓
Check approval rules
      ↓
Match: "delete" requires ApprovalExplicit
      ↓
Prompt: "Type 'delete /path/to/file' to confirm"
      ↓
User types command
      ↓
Execute if matches
```

## Configuration

### Adding Allowed Commands

Modify `internal/allowlist/allowlist.go`:

```go
var defaultAllowed = []string{
    "ls", "cat", "pwd",
    "echo",  // Add new command
    "date",
}
```

### Adding Approval Rules

Modify `internal/approval/rules.go`:

```go
var defaultRules = []ApprovalRule{
    {Pattern: "delete", Level: ApprovalExplicit},
    {Pattern: "remove", Level: ApprovalExplicit},
    {Pattern: "write",  Level: ApprovalSimple},
}
```

## Security Considerations

### Safe Mode

- Limits blast radius of mistakes
- Prevents accidental data loss
- Auditable command history
- Suitable for production use

### Dangerous Mode

- Only use for development/testing
- Never enable on shared systems
- Always review what AI executed
- Consider using on isolated VMs

## CLI Flags

| Flag | Description |
|------|-------------|
| `--allow-dangerous` / `--ad` | Enable dangerous mode |
| `--yes` / `-y` | Auto-approve simple confirmations |
| `--no-confirm` | Skip all confirmations (dangerous mode only) |
