# Command Execution

## Overview

The `exec` package provides safe command execution with output capture, timeouts, and size limits.

## Runner

```go
runner := cmdexec.NewRunner()
result := runner.Run(ctx, "docker", "info")
```

## Result Structure

```go
type Result struct {
    ExitCode int
    Stdout   *bytes.Buffer
    Stderr   *bytes.Buffer
    Error    error
}

// Get tail of output
stdout := result.StdoutTail(50) // Last 50 lines
stderr := result.StderrTail(50)
```

## Options

### Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

result := runner.Run(ctx, "long-running-command")
```

### Environment

```go
runner := cmdexec.NewRunner()
runner.Env = append(os.Environ(), "KEY=value")
result := runner.Run(ctx, "command")
```

### Working Directory

```go
runner := cmdexec.NewRunner()
runner.Dir = "/path/to/dir"
result := runner.Run(ctx, "command")
```

## Output Limits

Output is capped to prevent memory issues:

```go
const MaxOutputBytes = 1024 * 1024 // 1MB
```

## Error Handling

```go
result := runner.Run(ctx, "command")

if result.Error != nil {
    // Command failed to start
    return result.Error
}

if result.ExitCode != 0 {
    // Command exited with error
    return fmt.Errorf("exit code %d: %s", result.ExitCode, result.StderrTail(10))
}
```

## Tool Integration

Used by tool implementations:

```go
func (t *CheckDockerTool) Run(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
    runner := cmdexec.NewRunner()
    result := runner.Run(ctx, "docker", "info")

    return &ToolResult{
        OK:         result.ExitCode == 0,
        ExitCode:   result.ExitCode,
        StdoutTail: result.StdoutTail(50),
        StderrTail: result.StderrTail(50),
    }, nil
}
```

## Security

- Commands are not run through a shell (prevents injection)
- Arguments passed as separate strings
- Environment controlled explicitly

## Implementation Files

| File | Purpose |
|------|---------|
| `internal/exec/runner.go` | Command runner |
| `internal/exec/result.go` | Result handling |
