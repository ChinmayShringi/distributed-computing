# Tool Registry

## Overview

The Tool Registry is the central system for managing CLI tools that can be executed during remediation and chat sessions. It handles tool registration, allowlist enforcement, and mode-aware execution.

## Architecture

```go
type Registry struct {
    tools     map[string]Tool  // Registered tools
    allowlist []api.Tool       // Server-provided allowlist
}

type Tool interface {
    Name() string
    Description() string
    ArgsSchema() json.RawMessage
    IsDangerous() bool
    Run(ctx context.Context, args map[string]interface{}) (*ToolResult, error)
}

type ToolResult struct {
    OK         bool
    ExitCode   int
    StdoutTail string
    StderrTail string
    Message    string
}
```

## Registered Tools

### Health Check Tools

| Tool | Description | Dangerous |
|------|-------------|-----------|
| `health_check` | Run comprehensive OMNI health checks | No |
| `check_docker` | Check Docker health (structured JSON) | No |
| `check_compose` | Check Docker Compose availability | No |
| `check_python` | Check Python 3.9+ installation | No |
| `check_git` | Check Git installation | No |
| `check_ports` | Check required ports (4443, 8443, 8998) | No |
| `check_network` | Check network connectivity | No |
| `check_disk_space` | Check available disk space | No |
| `check_memory` | Check available memory | No |

### Service Management Tools

| Tool | Description | Dangerous |
|------|-------------|-----------|
| `restart_docker` | Restart Docker daemon | Yes |
| `restart_service` | Restart specific OMNI service | Yes |
| `view_logs` | View service logs | No |

### HostAgent Server Tools

| Tool | Description | Dangerous |
|------|-------------|-----------|
| `setup_hostagent_server` | Complete setup: download artifact → install → start → verify | Yes |
| `verify_hostagent_server` | Verify hostagent-server is running | No |

The `setup_hostagent_server` tool performs the following steps:
1. Check Docker is installed and running
2. Download artifact from server using AWS S3 presigned URL
3. Verify SHA256 hash
4. Extract artifact
5. Run install script from manifest.json
6. Start services via docker compose
7. Verify health

### Utility Tools

| Tool | Description | Dangerous |
|------|-------------|-----------|
| `get_system_time` | Get current system date/time | No |

## Execution Methods

### Standard Execution

```go
// Enforces allowlist
result, err := registry.Execute(ctx, "check_docker", args)
```

### Mode-Aware Execution

```go
// In SAFE mode: enforces allowlist
// In DANGEROUS mode: bypasses allowlist
result, err := registry.ExecuteWithMode(ctx, "check_docker", args, modeCtx)
```

## Allowlist System

### Server-Provided Allowlist

The server provides a list of allowed tools:

```go
type api.Tool struct {
    ToolName  string
    Enabled   bool
    ArgsSchema json.RawMessage
    Dangerous bool
}
```

### Allowlist Behavior

| Scenario | Allowlist | Tool Allowed |
|----------|-----------|--------------|
| No allowlist set | nil | All registered tools allowed |
| Allowlist set, tool enabled | tool in list | Allowed |
| Allowlist set, tool disabled | tool in list, enabled=false | Not allowed |
| Allowlist set, tool missing | tool not in list | Not allowed |

### SAFE vs DANGEROUS Mode

| Mode | Allowlist Enforcement |
|------|----------------------|
| SAFE | Enforced - only allowed tools can run |
| DANGEROUS | Bypassed - any registered tool can run |

## Creating a New Tool

### Tool Implementation

```go
type MyTool struct{}

func (t *MyTool) Name() string        { return "my_tool" }
func (t *MyTool) Description() string { return "Does something useful" }
func (t *MyTool) ArgsSchema() json.RawMessage {
    return json.RawMessage(`{
        "type": "object",
        "properties": {
            "param1": {"type": "string"}
        }
    }`)
}
func (t *MyTool) IsDangerous() bool   { return false }

func (t *MyTool) Run(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
    // Implementation
    return &ToolResult{OK: true, Message: "Success"}, nil
}
```

### Registration

```go
func (r *Registry) registerBuiltinTools() {
    // ... existing tools ...
    r.Register(&MyTool{})
}
```

## Tool Result Format

```go
type ToolResult struct {
    OK         bool   `json:"ok"`          // Success flag
    ExitCode   int    `json:"exit_code"`   // Process exit code
    StdoutTail string `json:"stdout_tail"` // Last N lines of stdout
    StderrTail string `json:"stderr_tail"` // Last N lines of stderr
    Message    string `json:"message"`     // Human-readable message
}
```

## Error Handling

| Error | Cause |
|-------|-------|
| "tool not found" | Tool not registered in registry |
| "tool not allowed" | Tool blocked by allowlist (SAFE mode) |
| Execution error | Tool's Run() returned an error |

## Implementation Files

| File | Purpose |
|------|---------|
| `internal/tools/registry.go` | Registry and all tool definitions |
| `internal/artifact/download.go` | Artifact download with verification |
| `internal/artifact/extract.go` | Tar.gz extraction |
| `internal/artifact/install.go` | Manifest-based installation |
