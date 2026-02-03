# Service Lifecycle

## Overview

The `up`, `down`, and `status` commands manage the OMNI service lifecycle using Docker Compose.

## Commands

### Start Services

```bash
omniforge up
```

Starts all OMNI services via `docker-compose up -d`.

### Stop Services

```bash
# Stop services (preserve data)
omniforge down

# Stop and remove volumes
omniforge down --volumes
```

### Check Status

```bash
omniforge status
```

## Services

| Service | Container | Description |
|---------|-----------|-------------|
| hostagent-server | hostagent-server | Core host agent |
| Keycloak | keycloak | Authentication server |
| Gateway | omni-gateway | API gateway |
| PostgreSQL | gateway-db | Database |

## Status Output

```
OMNI Service Status
-------------------
  ✓ hostagent-server: running (healthy)
  ✓ keycloak: running (healthy)
  ✓ omni-gateway: running (healthy)
  ✓ gateway-db: running (healthy)
```

When not installed:
```
OMNI is not installed at /opt/hostagent-server
```

## Volume Management

With `--volumes` flag, data is permanently removed:

- PostgreSQL database
- Keycloak realm data
- Gateway configurations

**Warning:** This is destructive and cannot be undone.

## Controller Architecture

```go
type Controller struct {
    config      *config.Config
    InstallPath string
}

func (c *Controller) Up(ctx context.Context) error
func (c *Controller) Down(ctx context.Context, removeVolumes bool) error
func (c *Controller) Status(ctx context.Context) (*Status, error)
func (c *Controller) IsInstalled() bool
func (c *Controller) RequireInstalled() error
```

## Docker Compose Operations

### Up
```bash
docker compose -f /opt/hostagent-server/docker-compose.yml up -d
```

### Down
```bash
docker compose -f /opt/hostagent-server/docker-compose.yml down
```

### Down with Volumes
```bash
docker compose -f /opt/hostagent-server/docker-compose.yml down -v
```

## Status Structure

```go
type Status struct {
    Installed bool
    Services  []ServiceStatus
}

type ServiceStatus struct {
    Name    string
    Running bool
    Health  string // healthy, unhealthy, starting
}
```

## Implementation Files

| File | Purpose |
|------|---------|
| `cmd/spotlight/commands/root.go` | up/down/status commands |
| `internal/omni/controller.go` | Lifecycle management |
| `internal/omni/compose.go` | Docker Compose operations |
