# Redaction System

## Overview

The redaction system automatically removes sensitive data from logs and incident bundles before upload. It protects secrets while maintaining debugging information.

## Redacted Patterns

| Pattern | Example |
|---------|---------|
| JWT tokens | `eyJhbGciOiJIUzI1NiIs...` → `[REDACTED:JWT]` |
| API keys | `api_key=abc123` → `api_key=[REDACTED]` |
| Passwords | `password=secret` → `password=[REDACTED]` |
| Bearer tokens | `Authorization: Bearer xxx` → `Authorization: Bearer [REDACTED]` |
| Connection strings | `postgres://user:pass@host` → `postgres://[REDACTED]@host` |
| Private keys | `-----BEGIN RSA PRIVATE KEY-----` → `[REDACTED:PRIVATE_KEY]` |
| AWS credentials | `AKIA...` → `[REDACTED:AWS_KEY]` |

## Usage

### Redact String

```go
redacted := redact.Redact(input)
```

### Redact File

```go
redact.RedactFile(inputPath, outputPath)
```

### Redact Logs

```go
redact.RedactLogs(logEntries)
```

## Pattern Definitions

```go
var patterns = []Pattern{
    {
        Name:    "jwt",
        Regex:   `eyJ[A-Za-z0-9_-]*\.eyJ[A-Za-z0-9_-]*\.[A-Za-z0-9_-]*`,
        Replace: "[REDACTED:JWT]",
    },
    {
        Name:    "password",
        Regex:   `(?i)(password|passwd|pwd)\s*[=:]\s*\S+`,
        Replace: "$1=[REDACTED]",
    },
    {
        Name:    "bearer",
        Regex:   `(?i)bearer\s+\S+`,
        Replace: "Bearer [REDACTED]",
    },
    // ... more patterns
}
```

## Application Points

| Location | Redaction Applied |
|----------|------------------|
| Incident bundles | All logs and configs |
| Log uploads | Service logs |
| Error messages | Before display |
| API responses | Sensitive fields |

## Incident Bundle Flow

```
Collect Logs
    │
    ▼
┌───────────────┐
│ Apply Redact  │
│ Patterns      │
└───────┬───────┘
        │
        ▼
┌───────────────┐
│ Create ZIP    │
└───────┬───────┘
        │
        ▼
Upload to Server
```

## Testing Redaction

```go
func TestRedaction(t *testing.T) {
    input := "password=secret123"
    output := redact.Redact(input)
    assert.Equal(t, "password=[REDACTED]", output)
    assert.NotContains(t, output, "secret123")
}
```

## Custom Patterns

Add custom patterns in `internal/redact/patterns.go`:

```go
func init() {
    patterns = append(patterns, Pattern{
        Name:    "custom_secret",
        Regex:   `CUSTOM_KEY_\w+`,
        Replace: "[REDACTED:CUSTOM]",
    })
}
```

## Security Considerations

- Patterns are applied in order
- Case-insensitive matching for common keywords
- Multi-line support for keys and certificates
- Preserves structure for debugging

## Implementation Files

| File | Purpose |
|------|---------|
| `internal/redact/redact.go` | Main redaction logic |
| `internal/redact/patterns.go` | Pattern definitions |
| `internal/bundle/incident.go` | Applies to bundles |
