# Incident Bundles

## Overview

Incident bundles are diagnostic archives containing system state, logs, and health information for support troubleshooting.

## Bundle Structure

```
incident-20260128-123456.zip
├── metadata.json       # Incident metadata
├── health.json         # Health check results
├── system_info.json    # System information
├── logs/
│   ├── hostagent.log   # Service logs (redacted)
│   ├── keycloak.log
│   ├── gateway.log
│   └── postgres.log
└── dangerous_mode.json # If applicable
```

## Metadata

```json
{
  "id": "inc_abc123",
  "created_at": "2026-01-28T12:00:00Z",
  "os": "darwin",
  "arch": "arm64",
  "cli_version": "1.0.0",
  "summary": "Services not starting after Docker update",
  "dangerous_mode": false
}
```

## System Info

```json
{
  "hostname": "dev-machine",
  "os": "darwin",
  "arch": "arm64",
  "platform": "macOS 14.0",
  "docker_version": "24.0.7",
  "compose_version": "2.21.0",
  "python_version": "3.11.4"
}
```

## Health Snapshot

```json
{
  "overall": "degraded",
  "timestamp": "2026-01-28T12:00:00Z",
  "checks": [
    {"name": "docker_daemon", "ok": true, "message": "Running"},
    {"name": "keycloak", "ok": false, "message": "Connection refused"},
    {"name": "hostagent", "ok": true, "message": "Port 8998 reachable"},
    {"name": "gateway", "ok": true, "message": "Port 8443 reachable"}
  ]
}
```

## Dangerous Mode Audit

When created during a dangerous mode session:

```json
{
  "enabled": true,
  "consent_timestamp": "2026-01-28T11:55:00Z",
  "consent_hash": "sha256:abc123...",
  "user_uid": "501",
  "cli_version": "1.0.0",
  "os": "darwin",
  "arch": "arm64"
}
```

## Bundle Types

| Type | Created When |
|------|--------------|
| `missing_scripts` | Installer scripts not found |
| `setup_failed` | Setup script execution failed |
| `run_failed` | Run script execution failed |
| `verify_failed` | Verification failed |
| `manual` | User-initiated via `incident create` |

## Automatic Bundle Creation

Bundles are automatically created on failure:

```go
bundle, err := incident.Create(ctx, incident.BundleTypeSetupFailed, "hostagent-server setup failed")
```

## Storage

| Location | Purpose |
|----------|---------|
| `~/.omniforge/incidents/` | Local storage |
| S3 (omniforge-incidents) | Server storage |

## Auto-Cleanup

Local bundles are automatically cleaned up:
- Keeps last 10 incidents
- Oldest removed first
- Configured in `internal/incident/cleanup.go`

## Bundle Creation API

```go
type IncidentBundle struct {
    ID            string
    Timestamp     time.Time
    Type          BundleType
    Description   string
    Tool          string
    OS            string
    Arch          string
    Distro        string
    SearchedPaths []string
    Environment   map[string]string
    DockerStatus  *DockerStatus
    ErrorMessage  string
    ExitCode      int
    Logs          []LogEntry
}

func Create(ctx context.Context, bundleType BundleType, description string) (*IncidentBundle, error)
func CreateWithMode(ctx context.Context, summary string, modeCtx *mode.ModeContext) (string, error)
```

## Implementation Files

| File | Purpose |
|------|---------|
| `internal/bundle/incident.go` | Bundle creation |
| `internal/incident/types.go` | Bundle types |
| `internal/incident/bundle.go` | Core bundle logic |
| `internal/incident/collector.go` | Data collection |
