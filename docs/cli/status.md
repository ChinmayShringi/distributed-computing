# Status Command

## Overview

The `status` command provides comprehensive visibility into OMNI service status and connectivity to all key components. It performs health checks on local services, server API, artifacts endpoint, and Docker.

## Usage

```bash
# Human-readable status report
omniforge status

# JSON output for machine parsing
omniforge status --json
```

### Flags

| Flag | Description |
|------|-------------|
| `--json` | Output as JSON for machine parsing |
| `-h, --help` | Help for status |

## Checks Performed

The status command runs the following checks:

### Auth
- **auth_token**: Validates authentication token validity and expiration

### Server
- **server_health**: GET request to `{server_url}/health`
- **server_status**: GET request to `{server_url}/v1/server/status` (authenticated)

### Artifacts
- **artifacts_manifest**: GET request to `{server_url}/v1/artifacts/manifest` (authenticated)

### Docker
- **docker_daemon**: Checks Unix socket `/var/run/docker.sock` accessibility

### Hostagent
- **hostagent_installed**: Checks for `docker-compose.yml` at install path
- **hostagent_port**: TCP connection to `localhost:8998`
- **compose_services**: Docker Compose service status via `omni.Controller`

## Output Formats

### Human-Readable (Default)

```
OmniForge Status Report
========================
Timestamp: 2026-01-28T10:30:00Z

Auth
  ✓ auth_token           user: admin@example.com, expires: 2026-01-29

Server
  ✓ server_health        http://50.17.127.28:3000/health (45ms)
  ✓ server_status        reachable (23ms)

Artifacts
  ✓ artifacts_manifest   reachable (120ms)

Docker
  ✓ docker_daemon        /var/run/docker.sock accessible (5ms)

Hostagent
  ✓ hostagent_installed  /opt/hostagent-server/docker-compose.yml
  ✓ hostagent_port       port 8998 listening (15ms)
  ✓ compose_services     4/4 services running

Summary: 8 OK, 0 FAILED, 0 UNKNOWN
```

### Status Icons

| Icon | Status | Meaning |
|------|--------|---------|
| ✓ | OK | Check passed |
| ✗ | FAIL | Check failed |
| ⚠ | UNKNOWN | Check could not be performed (missing config, not installed) |

### JSON Output

```bash
omniforge status --json
```

```json
{
  "timestamp": "2026-01-28T10:30:00Z",
  "checks": [
    {
      "name": "auth_token",
      "category": "auth",
      "status": "ok",
      "message": "user: admin@example.com, expires: 2026-01-29",
      "latency_ms": 0,
      "config_key": "auth.access_token"
    },
    {
      "name": "server_health",
      "category": "server",
      "status": "ok",
      "message": "http://50.17.127.28:3000/health (45ms)",
      "latency_ms": 45,
      "config_key": "server_url"
    }
  ],
  "summary": {
    "total": 8,
    "ok": 8,
    "failed": 0,
    "unknown": 0
  }
}
```

## Configuration Keys

The status command reads configuration from these sources:

| Config Key | Source | Default | Description |
|------------|--------|---------|-------------|
| `server_url` | `~/.omniforge/config.json` | `http://50.17.127.28:3000` | API server base URL |
| `omni_install_path` | `~/.omniforge/config.json` | `/opt/hostagent-server` | Local hostagent installation path |
| `auth.access_token` | `~/.omniforge/config.json` | (none) | Authentication token |

## Timeout Behavior

- Each check has a **5 second timeout**
- Overall command has a **30 second timeout**
- Checks that timeout are reported as FAIL with timeout error message
- The command never hangs indefinitely

## UNKNOWN Status

A check returns UNKNOWN when:

1. **Not Installed**: Local component is not present
   ```
   Hostagent
     ⚠ hostagent_installed  not installed
                            /opt/hostagent-server/docker-compose.yml not found
   ```

2. **Not Authenticated**: Check requires auth but user is not logged in
   ```
   Server
     ⚠ server_status        requires authentication
   ```

## Error Handling

| Error | Cause | Solution |
|-------|-------|----------|
| "token expired" | Auth token has expired | Run `omniforge login` |
| "401 Unauthorized" | Invalid or expired token | Run `omniforge login` |
| "connection refused" | Service not running | Start the service or check address |
| "timeout" | Service not responding | Check network connectivity |
| "not installed" | Component not present | Run `omniforge install` |

## Use Cases

### Quick Health Check
```bash
omniforge status
```
Shows overview of all component connectivity.

### CI/CD Integration
```bash
omniforge status --json | jq '.summary.failed'
```
Returns number of failed checks for automated monitoring.

### Troubleshooting
```bash
omniforge status
# If issues found:
omniforge doctor
```
Use status to identify issues, then doctor to fix them.

## Implementation Files

| File | Purpose |
|------|---------|
| `cmd/spotlight/commands/root.go` | Status command definition |
| `internal/status/status.go` | Status runner and checks |
| `internal/netcheck/netcheck.go` | HTTP/TCP check utilities |
| `internal/config/config.go` | Configuration loading |
| `internal/omni/controller.go` | Docker Compose status |
