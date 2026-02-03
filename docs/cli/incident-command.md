# Incident Command

## Overview

The `incident` command creates and manages diagnostic bundles for support and troubleshooting.

## Subcommands

### Create Incident

```bash
# Create and upload incident
omniforge incident create --summary "Services not starting"

# Create without upload (offline)
omniforge incident create --summary "Issue description" --no-upload
```

### Upload Incident

```bash
omniforge incident upload ./incident-20250126-123456.zip
```

### List Incidents

```bash
omniforge incident list
```

## Flags

| Flag | Description |
|------|-------------|
| `--summary` | Description of the incident |
| `--no-upload` | Create bundle without uploading |

## Bundle Contents

An incident bundle includes:

| File | Contents |
|------|----------|
| `metadata.json` | System info, CLI version, timestamp |
| `health.json` | Health check results |
| `logs/*.log` | Service logs (redacted) |
| `system_info.json` | OS, architecture, hostname |
| `dangerous_mode.json` | Consent metadata (if applicable) |

## Bundle Creation

```
Creating incident bundle...
  Collecting system info...
  Running health checks...
  Gathering logs...
  Redacting secrets...
  Creating archive...

Incident bundle created: incident-20260128-123456.zip
Size: 2.5 MB

Uploading to OmniForge...
  Upload complete
  Incident ID: inc_abc123
```

## Metadata Structure

```json
{
  "created_at": "2026-01-28T12:00:00Z",
  "os": "darwin",
  "arch": "arm64",
  "cli_version": "1.0.0",
  "summary": "Services not starting",
  "dangerous_mode": false
}
```

## Health Check Snapshot

```json
{
  "overall": "degraded",
  "checks": [
    {"name": "docker", "ok": true, "message": "Running"},
    {"name": "keycloak", "ok": false, "message": "Not responding"}
  ]
}
```

## Log Redaction

Sensitive data is automatically redacted:

- JWT tokens
- API keys
- Passwords
- Connection strings
- Private keys

Redaction patterns defined in `internal/redact/patterns.go`.

## Upload Process

1. Request presigned S3 URL from server
2. Upload bundle to S3
3. Create incident record in database
4. Return incident ID

## List Output

```
Incidents:
  inc_abc123  2026-01-28  Services not starting
  inc_def456  2026-01-27  Keycloak failing
  inc_ghi789  2026-01-26  Docker issues
```

## Storage

- **Local**: `~/.omniforge/incidents/`
- **Server**: S3 bucket (omniforge-incidents)

## Implementation Files

| File | Purpose |
|------|---------|
| `cmd/spotlight/commands/root.go` | incident command |
| `internal/bundle/incident.go` | Bundle creation |
| `internal/bundle/collector.go` | Data collection |
| `internal/redact/redact.go` | Secret redaction |
| `internal/api/incidents.go` | Upload and list |
