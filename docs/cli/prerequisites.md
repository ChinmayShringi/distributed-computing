# Prerequisites

## Overview

The `prereq` command checks and optionally installs all prerequisites required for running OMNI infrastructure.

## Usage

```bash
# Check prerequisites only
omniforge prereq

# Check and install missing prerequisites
omniforge prereq --install
```

### Flags

| Flag | Description |
|------|-------------|
| `--install` | Install missing prerequisites |

## Required Prerequisites

| Prerequisite | macOS | Ubuntu/Debian | Purpose |
|--------------|-------|---------------|---------|
| Docker | Homebrew (cask) | apt | Container runtime |
| Docker Compose | Bundled with Docker | Plugin | Service orchestration |
| Python 3.9+ | Homebrew | apt | Scripts and tools |
| Git | Homebrew/Xcode | apt | Version control |

### Required Ports

| Port | Service |
|------|---------|
| 4443 | Keycloak HTTPS |
| 8443 | Gateway API |
| 8998 | gRPC server |

## Output

```
Platform: darwin/arm64 (macOS)

Prerequisites:
  ✓ docker: Docker 24.0.7
  ✓ docker_compose: Docker Compose v2.21.0
  ✓ python: Python 3.11.4
  ✗ ports: Port 4443 in use

Some prerequisites are missing. Run with --install to install them.
```

## Installation Process

With `--install` flag:

1. Requires root privileges (`sudo`)
2. Iterates through failed checks
3. Runs platform-specific installer
4. Reports success/failure for each

```
Installing missing prerequisites...
Installing docker...
  Done
Installing python...
  Done
```

## Platform Detection

The CLI detects the platform using `osdetect`:

| Platform | Detection Method |
|----------|-----------------|
| macOS | `runtime.GOOS == "darwin"` |
| Linux (Ubuntu) | `/etc/os-release` |
| Linux (Debian) | `/etc/os-release` |
| Windows | Detected but not fully supported |

## Prerequisite Registry

```go
type Prereq interface {
    Name() string
    Check(ctx context.Context) (*Result, error)
    Install(ctx context.Context) error
}

type Registry struct {
    prereqs map[string]Prereq
}
```

## Implementation Files

| File | Purpose |
|------|---------|
| `cmd/spotlight/commands/root.go` | prereq command |
| `internal/prereq/registry.go` | Prerequisite registry |
| `internal/prereq/docker.go` | Docker prerequisite |
| `internal/prereq/python.go` | Python prerequisite |
| `internal/prereq/git.go` | Git prerequisite |
| `internal/prereq/ports.go` | Port availability |
| `internal/osdetect/detect.go` | Platform detection |
