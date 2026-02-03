# Changelog: CLI Artifact Download Command

**Date:** 2026-01-27
**Type:** Feature Addition

## Summary

Added a new CLI command `omniforge artifact download --latest` that downloads release artifacts from the backend via presigned URL, verifies SHA256 checksum, extracts the tarball, and optionally runs installation scripts with user confirmation prompts.

## Changes Overview

### New Files

| File | Description |
|------|-------------|
| `internal/artifact/download.go` | Download functionality with progress and SHA256 verification |
| `internal/artifact/extract.go` | Tar.gz extraction with path security |
| `internal/artifact/install.go` | Script execution with permission prompts |

### Modified Files

| File | Changes |
|------|---------|
| `cmd/spotlight/commands/root.go` | Added `artifact` command with `download` subcommand |
| `internal/api/artifacts.go` | Added `GetLatestArtifact()` method and `LatestArtifactResponse` type |

## Usage

### Basic Download

```bash
# Login first
omniforge login

# Download latest release
omniforge artifact download --latest
```

### Download with Installation

```bash
# Download and run install scripts (with prompts)
omniforge artifact download --latest --install

# Download and auto-approve all prompts
omniforge artifact download --latest --install -y
```

## Command Details

### `omniforge artifact download`

| Flag | Description |
|------|-------------|
| `--latest` | Download the latest release (required) |
| `--install` | Run prereq and install scripts after extraction |
| `-y, --yes` | Auto-approve script execution prompts |

### Flow

```
1. Authenticate with server (requires login)
2. GET /v1/artifact/latest → {version, asset_name, sha256, url}
3. Download tar.gz to ~/.spotlight/cache/<version>/<asset_name>
4. Verify SHA256 checksum (fail if mismatch)
5. Extract to ~/.spotlight/releases/<version>/
6. Read manifest.json (fail with helpful error if missing)
7. If --install:
   a. Detect OS (linux/darwin) and distro (ubuntu/debian/etc)
   b. Find appropriate script in manifest
   c. Prompt user before each script execution
   d. Execute scripts with environment variables
```

### Directory Structure

```
~/.spotlight/
├── cache/
│   └── v1.0.0/
│       └── hostagent-server.tar.gz
└── releases/
    └── v1.0.0/
        ├── manifest.json
        ├── scripts/
        │   ├── prereq-linux.sh
        │   ├── prereq-darwin.sh
        │   ├── install-linux.sh
        │   └── install-darwin.sh
        └── ... (other release files)
```

### Manifest Format

The tarball must contain a `manifest.json` at its root:

```json
{
  "version": "1.0.0",
  "name": "hostagent-server",
  "description": "OMNI Host Agent Server",
  "prereq": {
    "linux": {
      "path": "scripts/prereq-linux.sh",
      "description": "Install Docker and dependencies",
      "require_root": true
    },
    "darwin": {
      "path": "scripts/prereq-darwin.sh",
      "description": "Install Docker and dependencies"
    }
  },
  "install": {
    "ubuntu": {
      "path": "scripts/install-ubuntu.sh",
      "description": "Install on Ubuntu",
      "require_root": true
    },
    "linux": {
      "path": "scripts/install-linux.sh",
      "description": "Generic Linux install",
      "require_root": true
    },
    "darwin": {
      "path": "scripts/install-darwin.sh",
      "description": "Install on macOS"
    }
  }
}
```

### Error Handling

**Missing manifest.json:**
```
Error: manifest.json not found at /path/to/release/manifest.json.
The tarball should contain a manifest.json file at its root with the following structure:

{
  "version": "1.0.0",
  "name": "hostagent-server",
  ...
}
```

**SHA256 mismatch:**
```
Error: SHA256 mismatch: expected abc123..., got def456...
```

**Script not found:**
```
Error: Install script not found: scripts/install-linux.sh (expected at /path/to/release/scripts/install-linux.sh)
```

## Environment Variables

Scripts receive these environment variables:

| Variable | Description |
|----------|-------------|
| `SPOTLIGHT_VERSION` | Release version from manifest |
| `SPOTLIGHT_PLATFORM` | Platform (linux/darwin) |
| `SPOTLIGHT_ARCH` | Architecture (amd64/arm64) |
| `SPOTLIGHT_DISTRO` | Linux distro (ubuntu/debian/etc) |
| `SPOTLIGHT_EXTRACT_PATH` | Path to extracted release |

## Manual Test Procedure

### Ubuntu

```bash
# 1. Build the CLI
cd /path/to/spotlight-omni-cli
go build -o omniforge ./cmd/spotlight

# 2. Set up test environment
export PATH="$PWD:$PATH"

# 3. Login
omniforge login -u testuser -p testpass

# 4. Test download (no install)
omniforge artifact download --latest

# Expected output:
# Platform: linux/ubuntu 22.04 (amd64)
# Fetching latest release info...
#   Version: v0.1.0
#   Asset:   hostagent-server.tar.gz
#   SHA256:  abc123...
# Downloading hostagent-server.tar.gz...
#   Progress: 100.0% (12345678 / 12345678 bytes)
#   Downloaded to: ~/.spotlight/cache/v0.1.0/hostagent-server.tar.gz
#   SHA256: abc123...
# Extracting to ~/.spotlight/releases/v0.1.0/...
#   Extracted to: ~/.spotlight/releases/v0.1.0
# Reading manifest.json...
#   Name:    hostagent-server
#   Version: v0.1.0
# Available scripts:
#   - Prerequisite script
#   - Install script
# To run installation scripts, use: omniforge artifact download --latest --install

# 5. Verify files
ls -la ~/.spotlight/cache/v0.1.0/
ls -la ~/.spotlight/releases/v0.1.0/
cat ~/.spotlight/releases/v0.1.0/manifest.json

# 6. Test with install (will prompt)
omniforge artifact download --latest --install

# 7. Test with auto-approve
omniforge artifact download --latest --install -y
```

### macOS

```bash
# 1. Build the CLI
cd /path/to/spotlight-omni-cli
go build -o omniforge ./cmd/spotlight

# 2. Set up test environment
export PATH="$PWD:$PATH"

# 3. Login
omniforge login -u testuser -p testpass

# 4. Test download
omniforge artifact download --latest

# Expected output:
# Platform: darwin (arm64)
# Fetching latest release info...
# ...

# 5. Verify files
ls -la ~/.spotlight/cache/v0.1.0/
ls -la ~/.spotlight/releases/v0.1.0/

# 6. Test with install
omniforge artifact download --latest --install
```

### Testing Without Server

For local testing without the backend:

```bash
# Create a mock release
mkdir -p /tmp/test-release/scripts
cat > /tmp/test-release/manifest.json << 'EOF'
{
  "version": "1.0.0",
  "name": "test-release",
  "prereq": {
    "linux": {"path": "scripts/prereq.sh"},
    "darwin": {"path": "scripts/prereq.sh"}
  },
  "install": {
    "linux": {"path": "scripts/install.sh"},
    "darwin": {"path": "scripts/install.sh"}
  }
}
EOF

cat > /tmp/test-release/scripts/prereq.sh << 'EOF'
#!/bin/bash
echo "Running prereq script"
echo "Platform: $SPOTLIGHT_PLATFORM"
echo "Version: $SPOTLIGHT_VERSION"
EOF

cat > /tmp/test-release/scripts/install.sh << 'EOF'
#!/bin/bash
echo "Running install script"
echo "Extract path: $SPOTLIGHT_EXTRACT_PATH"
EOF

chmod +x /tmp/test-release/scripts/*.sh

# Create tarball
cd /tmp && tar -czvf test-release.tar.gz test-release/

# Host locally and test (requires local server modifications)
```

## Security Considerations

1. **SHA256 Verification** - Downloads are verified against server-provided hash
2. **Path Traversal Prevention** - Extraction rejects paths containing `..`
3. **Script Execution Prompts** - User must explicitly approve each script
4. **Sudo Handling** - Scripts requiring root use `sudo` (prompts for password)
5. **Environment Isolation** - Scripts run with controlled environment variables
