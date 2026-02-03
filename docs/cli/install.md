# Install Command

## Overview

The `install` command installs OMNI infrastructure including hostagent-server and required services.

## Usage

```bash
# Install from local tarball
omniforge install --tarball hostagent-server.tar.gz

# Install with auto-download (requires authentication)
omniforge install

# Offline installation
omniforge install --offline --tarball hostagent-server.tar.gz

# Force reinstall
omniforge install --force --tarball hostagent-server.tar.gz
```

### Flags

| Flag | Description |
|------|-------------|
| `--tarball` | Path to hostagent-server.tar.gz |
| `--offline` | Install from local bundle only |
| `--force` | Force reinstall even if already installed |

## Installation Process

1. **Check root privileges** - Requires `sudo`
2. **Detect platform** - macOS, Ubuntu/Debian
3. **Check prerequisites** - Docker, Python, Git, ports
4. **Locate tarball** - From flag, assets, or download
5. **Install OMNI** - Extract and configure
6. **Start services** - Run `docker-compose up`
7. **Verify health** - Run health checks

## Tarball Discovery

If `--tarball` not specified:

1. Check `assets/hostagent-server/hostagent-server.tar.gz`
2. Check relative to executable path
3. Download from server (if authenticated)

## Download from Server

When authenticated and tarball not found locally:

```
Downloading hostagent-server from OmniForge...
  Downloading hostagent-server.tar.gz (150.5 MB)...
  Download complete
```

Includes SHA256 verification.

## Installation Path

Default: `/opt/hostagent-server/`

Contents:
- `docker-compose.yml`
- Service configurations
- Data directories

## Output

```
Platform: darwin/arm64 (macOS)

Checking prerequisites...
  ✓ docker: Docker 24.0.7
  ✓ docker_compose: Docker Compose v2.21.0
  ✓ python: Python 3.11.4
  ✓ ports: All ports available

Installing OMNI from hostagent-server.tar.gz...
  Extracting...
  Loading Docker images...
  Setting permissions...

Starting OMNI services...
  hostagent-server: started
  keycloak: started
  gateway: started
  postgres: started

Waiting for services to become healthy.......
Overall Health: ok
---
  ✓ docker_daemon: Docker daemon is running
  ✓ postgres: PostgreSQL is running and healthy
  ✓ keycloak: Keycloak is running and OMNI realm is configured
  ✓ hostagent_server: Host Agent Server is running and listening on port 8998
  ✓ omni_gateway: OMNI Gateway is running and listening on port 8443

OMNI installed and running successfully!
```

The install command waits up to 90 seconds for all services to become healthy before reporting final status. Progress dots indicate waiting.

## Already Installed

```
OMNI is already installed at /opt/hostagent-server
Use 'omniforge down' to stop services, or reinstall with --force
```

## Error Handling

| Error | Cause | Solution |
|-------|-------|----------|
| "prerequisites not satisfied" | Missing Docker/Python | Run `omniforge prereq --install` |
| "tarball not found" | No local tarball | Provide `--tarball` or login to download |
| "not authenticated" | No valid session | Run `omniforge login` |
| "already installed" | OMNI exists | Use `--force` flag |

## Implementation Files

| File | Purpose |
|------|---------|
| `cmd/spotlight/commands/root.go` | install command |
| `internal/omni/controller.go` | Installation logic |
| `internal/omni/install.go` | Tarball extraction |
| `internal/api/artifacts.go` | Download from server |
