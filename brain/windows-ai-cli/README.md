# Windows AI CLI

A Windows-only C#/.NET CLI tool that wraps Windows AI APIs for plan generation and text processing. Designed to be called from Go for AI-powered task orchestration.

## Requirements

- Windows 10/11 (x64)
- .NET 8.0 SDK
- For AI features: Windows 11 24H2+ (build 26100+)

## Build

```bash
# Debug build
dotnet build

# Release build
dotnet build -c Release

# Output location
# bin/Release/net8.0-windows10.0.22621.0/win-x64/WindowsAiCli.exe
```

## Commands

### capabilities

Check Windows AI availability and supported features.

```bash
WindowsAiCli.exe capabilities
```

**Output:**
```json
{"ok":true,"features":["plan","summarize"],"notes":"Windows AI not available (fallback mode). Windows build 22631 detected. Windows AI requires build 26100+ (24H2)"}
```

### plan

Generate an execution plan from JSON input.

```bash
WindowsAiCli.exe plan --in request.json --format json
```

**Input file (request.json):**
```json
{
  "text": "collect system info from all devices",
  "max_workers": 2,
  "devices": [
    {"device_id": "dev-1", "device_name": "PC-1", "has_npu": true, "has_gpu": true},
    {"device_id": "dev-2", "device_name": "PC-2", "has_npu": false, "has_gpu": true},
    {"device_id": "dev-3", "device_name": "PC-3", "has_npu": false, "has_gpu": false}
  ]
}
```

**Output:**
```json
{
  "ok": true,
  "plan": {
    "groups": [
      {
        "index": 0,
        "tasks": [
          {"task_id": "uuid-1", "kind": "SYSINFO", "input": "collect_status", "target_device_id": "dev-1"},
          {"task_id": "uuid-2", "kind": "SYSINFO", "input": "collect_status", "target_device_id": "dev-2"}
        ]
      }
    ]
  },
  "reduce": {"kind": "CONCAT"}
}
```

Note: With `max_workers: 2`, only the first 2 devices are included in the plan.

### summarize

Summarize input text.

```bash
WindowsAiCli.exe summarize --text "Long text here..." --format json
```

**Output:**
```json
{"ok":true,"summary":"Long text here...","used_ai":false}
```

When text exceeds 200 characters, it's truncated with "...":
```json
{"ok":true,"summary":"This is a very long text that will be truncated at a word boundary...","used_ai":false}
```

## Error Handling

All errors return JSON with `ok: false`:

```json
{"ok":false,"error":"Input file not specified or does not exist. Use --in <path>"}
```

## JSON Schema

### PlanRequest

```json
{
  "text": "string - job description",
  "max_workers": "int - limit number of devices (0 = unlimited)",
  "devices": [
    {
      "device_id": "string - unique device ID",
      "device_name": "string - human-readable name",
      "has_npu": "bool - NPU capability",
      "has_gpu": "bool - GPU capability",
      "has_cpu": "bool - CPU capability (default true)",
      "grpc_addr": "string - device gRPC address"
    }
  ]
}
```

### Plan (output)

```json
{
  "groups": [
    {
      "index": "int - execution order (0 = first)",
      "tasks": [
        {
          "task_id": "string - unique task ID",
          "kind": "string - task type (SYSINFO, ECHO, etc.)",
          "input": "string - task parameters",
          "target_device_id": "string - assigned device ID"
        }
      ]
    }
  ]
}
```

### ReduceSpec

```json
{
  "kind": "string - reduce operation (CONCAT)"
}
```

## Integration with Go

Set environment variables and run the server:

```bash
# Windows PowerShell
$env:WINDOWS_AI_CLI_PATH = "C:\path\to\WindowsAiCli.exe"
$env:USE_WINDOWS_AI_PLANNER = "true"

# Run the orchestrator server
make server
```

The Go integration (`internal/brain`) will call this CLI when:
1. `USE_WINDOWS_AI_PLANNER=true`
2. `WINDOWS_AI_CLI_PATH` points to the executable
3. No plan is provided in the job request
4. Running on Windows

## Design Philosophy

**Fallback-first**: The CLI always generates a valid deterministic plan, even when Windows AI APIs are unavailable. AI features enhance but never block plan generation.

**Minimal and stable**: Only essential operations are exposed. The CLI focuses on:
- Plan generation with device-aware task distribution
- Text summarization for result processing

## Future Enhancements

When Windows 11 24H2+ is widely available:
- Use Windows AI for intelligent task routing based on device capabilities
- Use Windows AI for semantic summarization of job results
- Use Windows AI for natural language job specification parsing
