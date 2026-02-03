# Bundle Command

## Overview

The `bundle` command manages offline installation bundles (planned feature).

## Usage

```bash
# Create offline bundle (not yet implemented)
omniforge bundle create
```

## Planned Features

### Create Bundle

Creates a self-contained installation package:

- CLI binary for target platform
- Installer scripts
- Docker images (exported)
- Configuration templates

### Bundle Contents

```
offline-bundle/
├── omniforge-linux-amd64
├── scripts/
│   └── installers/
├── images/
│   ├── hostagent-server.tar
│   ├── keycloak.tar
│   └── ...
└── manifest.json
```

### Install from Bundle

```bash
# Extract and install from offline bundle
omniforge install --offline --bundle offline-bundle.tar.gz
```

## Current Status

This feature is planned but not yet implemented. The CLI currently supports:

- Installing from existing tarball (`omniforge install --tarball`)
- Downloading from server (requires authentication)

## Implementation Files

| File | Purpose |
|------|---------|
| `cmd/spotlight/commands/root.go` | bundle command stub |
