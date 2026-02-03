# API Client

## Overview

The `api` package provides the HTTP client for communicating with the OmniForge server.

## Client Creation

```go
cfg, _ := config.Load()
client := api.NewClient(cfg)
```

## Authentication

### Login

```go
err := client.Login(ctx, username, password)
```

Stores tokens in config on success.

### Token Refresh

```go
err := client.RefreshToken(ctx)
```

Called automatically when access token expires.

### Check Auth

```go
if client.IsAuthenticated() {
    // Make authenticated requests
}
```

## Endpoints

### Allowlist

```go
allowlist, err := client.GetAllowlist(ctx)
```

### Plan

```go
plan, err := client.GetPlan(ctx, &PlanRequest{
    HealthCheckOutput: summary,
    SystemInfo:        sysInfo,
    ExecutionMode:     "SAFE",
})
```

### Step Result

```go
err := client.ReportStepResult(ctx, &StepResult{
    RunID:     plan.RunID,
    StepIndex: 0,
    Tool:      "check_docker",
    OK:        true,
})
```

### Artifacts

```go
manifest, err := client.GetManifest(ctx, platform, arch, "latest")
err := client.DownloadArtifact(ctx, artifactID, outputPath)
```

### Incidents

```go
incident, err := client.CreateIncident(ctx, summary, metadata)
incidents, err := client.ListIncidents(ctx, limit, offset)
```

### Chat

```go
response, err := client.Chat(ctx, messages, systemContext)
```

## Request/Response Types

### PlanRequest

```go
type PlanRequest struct {
    HealthCheckOutput string
    SystemInfo        string
    LogSnippets       string
    PreviousSteps     []StepResult
    ExecutionMode     string
}
```

### Plan

```go
type Plan struct {
    RunID    string
    Goal     string
    Steps    []PlanStep
    StopCond string
    MaxSteps int
}
```

## Error Handling

```go
response, err := client.GetPlan(ctx, req)
if err != nil {
    if apiErr, ok := err.(*api.Error); ok {
        switch apiErr.StatusCode {
        case 401:
            // Handle unauthorized
        case 500:
            // Handle server error
        }
    }
}
```

## Implementation Files

| File | Purpose |
|------|---------|
| `internal/api/client.go` | Main client |
| `internal/api/auth.go` | Authentication |
| `internal/api/plan.go` | Planning API |
| `internal/api/chat.go` | Chat API |
| `internal/api/artifacts.go` | Artifact API |
| `internal/api/incidents.go` | Incident API |
