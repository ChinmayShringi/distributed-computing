# Privilege Elevation

## Overview

The `elevate` package handles root/administrator privilege requirements for system-level operations.

## Functions

### RequireRoot

Ensures the CLI runs with root privileges:

```go
if err := elevate.RequireRoot(); err != nil {
    return err
}
```

### Behavior

| Current User | Action |
|--------------|--------|
| Root (UID 0) | Continue |
| Non-root | Exit with instruction |

## Usage in Commands

Commands requiring root:

```go
var installCmd = &cobra.Command{
    RunE: func(cmd *cobra.Command, args []string) error {
        // Require root for installation
        if err := elevate.RequireRoot(); err != nil {
            return err
        }
        // Continue with install...
    },
}
```

## Commands Requiring Root

| Command | Reason |
|---------|--------|
| `omniforge install` | System-level installation |
| `omniforge prereq --install` | Package installation |
| `omniforge up` | Docker operations |
| `omniforge down` | Docker operations |

## Error Message

```
This command requires root privileges.
Please run with sudo:
    sudo omniforge install --tarball hostagent-server.tar.gz
```

## Platform-Specific

### Unix (macOS, Linux)

```go
func RequireRoot() error {
    if os.Getuid() != 0 {
        return fmt.Errorf("requires root privileges")
    }
    return nil
}
```

### Windows

Administrator check via Windows API (not fully implemented).

## Implementation Files

| File | Purpose |
|------|---------|
| `internal/elevate/elevate.go` | Main elevation logic |
| `internal/elevate/unix.go` | Unix implementation |
