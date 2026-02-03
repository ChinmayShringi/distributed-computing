# Version Command

## Overview

The `version` command displays CLI version, build, and platform information.

## Usage

```bash
omniforge version
```

## Output

```
OmniForge CLI
  Version:  1.0.0
  Commit:   abc123def
  Platform: darwin/arm64 (macOS 14.0)
```

## Build Information

Set at compile time via ldflags:

```makefile
VERSION ?= $(shell git describe --tags --always)
COMMIT ?= $(shell git rev-parse --short HEAD)

build:
    go build -ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT)" ...
```

## Variables

| Variable | Description |
|----------|-------------|
| `Version` | Git tag or "dev" |
| `Commit` | Short commit hash |

## Platform Detection

Uses `osdetect.Detect()`:

```go
info, err := osdetect.Detect()
if err == nil {
    fmt.Printf("Platform: %s\n", info)
}
```

## Implementation Files

| File | Purpose |
|------|---------|
| `cmd/spotlight/commands/root.go` | version command |
| `internal/osdetect/detect.go` | Platform detection |
