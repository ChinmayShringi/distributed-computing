# Logs Command

## Overview

The `logs` command displays service logs from OMNI containers.

## Usage

```bash
# View all service logs
omniforge logs

# Follow logs in real-time
omniforge logs -f

# Specify number of lines
omniforge logs -n 50

# View specific service logs
omniforge logs keycloak
omniforge logs hostagent
omniforge logs gateway
omniforge logs postgres
```

## Flags

| Flag | Short | Description | Default |
|------|-------|-------------|---------|
| `--follow` | `-f` | Follow log output | false |
| `--lines` | `-n` | Number of lines to show | 100 |

## Services

| Service | Container |
|---------|-----------|
| keycloak | keycloak |
| hostagent | hostagent-server |
| gateway | omni-gateway |
| postgres | gateway-db |
| all | All containers |

## Implementation

Uses `docker compose logs`:

```bash
# All logs
docker compose -f /opt/hostagent-server/docker-compose.yml logs --tail 100

# Follow mode
docker compose -f /opt/hostagent-server/docker-compose.yml logs -f

# Specific service
docker compose -f /opt/hostagent-server/docker-compose.yml logs keycloak --tail 50
```

## Implementation Files

| File | Purpose |
|------|---------|
| `cmd/spotlight/commands/root.go` | logs command |
| `internal/omni/controller.go` | Log retrieval |
