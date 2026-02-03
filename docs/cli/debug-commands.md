# Debug Commands

## Overview

Debug commands provide diagnostic information for troubleshooting CLI issues.

## Available Commands

### Debug Flags

Prints resolved global flag values for debugging:

```bash
# Show current flag values
omniforge debug flags

# Verify dangerous mode is enabled
omniforge --allow-dangerous debug flags

# Verify verbose mode
omniforge -v debug flags
```

#### Output

```
Resolved Flag Values:
  --verbose:         false
  --config:          ""
  --allow-dangerous: true
  --ad:              false
  --yes:             false
  Dangerous Mode:    true
```

This is useful for verifying that global flags like `--allow-dangerous` are correctly parsed and inherited by subcommands.

## Implementation Files

| File | Purpose |
|------|---------|
| `cmd/spotlight/commands/debug.go` | Debug commands |
