# Doctor Command

## Overview

The `doctor` command provides AI-assisted diagnostics and automatic remediation for OMNI infrastructure. It runs health checks, identifies issues, and executes fix steps automatically.

## Usage

```bash
# Online mode (AI-assisted, requires authentication)
omniforge doctor

# Offline mode (deterministic remediation ladder)
omniforge doctor --offline

# With dangerous mode (bypass safety controls)
omniforge doctor --allow-dangerous
omniforge doctor --ad

# Auto-confirm dangerous mode
omniforge doctor --ad -y
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--offline` | | Use deterministic remediation without server |
| `--allow-dangerous` | `--ad` | Enable dangerous mode |
| `--yes` | `-y` | Auto-confirm dangerous mode consent |

## How It Works

### Online Mode (Default)

1. **Run health checks** on all OMNI services
2. **Send diagnostics** to server for AI analysis
3. **Receive remediation plan** with specific fix steps
4. **Execute each step** automatically (with approval for dangerous tools)
5. **Re-check health** and iterate until resolved

```
Running diagnostics...
Overall Health: degraded
  ✓ docker_daemon: Docker daemon is running
  ✗ keycloak: Not responding on HTTPS

Requesting remediation plan from server...
Plan: Fix Keycloak HTTPS connectivity
Steps: 3

  [1] restart_service: Restart Keycloak container
      Result: OK
  [2] check_ports: Verify port 4443 is accessible
      Result: OK
  [3] health_check: Re-run health checks
      Result: OK

Overall Health: healthy
Remediation successful!
```

### Offline Mode

Uses a deterministic 8-step remediation ladder:

1. Check Docker daemon
2. Restart Docker if needed
3. Check required ports
4. Check disk space
5. Check memory
6. Restart services
7. View recent logs
8. Re-run health checks

No server connection required.

## Architecture

### Components

```
Doctor
├── API Client (server communication)
├── OMNI Controller (health checks)
├── Tool Registry (remediation tools)
└── Mode Context (safe/dangerous)
```

### Remediation Loop

```
┌─────────────────┐
│  Health Check   │
└────────┬────────┘
         │
    ┌────▼────┐
    │ All OK? │───Yes──► Done
    └────┬────┘
         │ No
    ┌────▼────────────┐
    │ Request Plan    │
    │ (from server)   │
    └────────┬────────┘
             │
    ┌────────▼────────┐
    │ Execute Steps   │
    │ (with approval) │
    └────────┬────────┘
             │
    └────────┘ (loop up to 5 iterations)
```

### Constants

| Constant | Value | Description |
|----------|-------|-------------|
| `MaxIterations` | 5 | Maximum remediation cycles |
| `MaxStepsPerPlan` | 25 | Maximum steps per plan |

## Plan Request

Sent to server for AI analysis:

```go
type PlanRequest struct {
    HealthCheckOutput string       // Summary of failed checks
    SystemInfo        string       // OS, arch, platform
    LogSnippets       string       // Last 50 lines of logs
    PreviousSteps     []StepResult // Results from prior attempts
    ExecutionMode     string       // "SAFE" or "DANGEROUS"
}
```

## Plan Response

Received from server:

```go
type Plan struct {
    RunID    string     // Unique identifier
    Goal     string     // What the plan aims to fix
    Steps    []PlanStep // Ordered remediation steps
    StopCond string     // Condition to stop
    MaxSteps int        // Maximum steps
}

type PlanStep struct {
    Tool string                 // Tool to execute
    Args map[string]interface{} // Tool arguments
    Why  string                 // Explanation
}
```

## Tool Approval

In SAFE mode, dangerous tools require confirmation:

```
[1] restart_docker: Restart the Docker daemon
    This tool is marked as dangerous. Run it? [y/N]:
```

In DANGEROUS mode, all tools execute without prompts (user already consented at session start).

## Troubleshooting

| Issue | Solution |
|-------|----------|
| "Not authenticated" | Run `omniforge login` |
| "Failed to fetch allowlist" | Check network, falls back to offline |
| "Remediation could not fully resolve" | Create incident report |
| Stuck in loop | Max 5 iterations, then stops |

## Implementation Files

| File | Purpose |
|------|---------|
| `internal/doctor/doctor.go` | Main doctor implementation |
| `internal/doctor/ladder.go` | Offline remediation ladder |
| `internal/omni/health.go` | Health check logic |
| `internal/tools/registry.go` | Tool execution |
