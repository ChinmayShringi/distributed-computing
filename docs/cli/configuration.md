# Configuration

## Overview

OmniForge CLI stores configuration in `~/.omniforge/` including authentication tokens, settings, and cached data.

## Directory Structure

```
~/.omniforge/
├── config.json         # Main configuration
├── approvals.json      # Approval rules
├── logs/               # CLI logs
├── cache/              # Downloaded artifacts
├── incidents/          # Local incident bundles
└── scripts/
    └── installers/     # Cached installer scripts
```

## Config File

`~/.omniforge/config.json`:

```json
{
  "auth": {
    "username": "user@example.com",
    "access_token": "eyJ...",
    "refresh_token": "eyJ...",
    "expires_at": "2026-01-28T12:15:00Z"
  },
  "api_url": "https://api.omniforge.io",
  "install_path": "/opt/hostagent-server"
}
```

## Configuration Fields

| Field | Type | Description |
|-------|------|-------------|
| `auth.username` | string | Authenticated username |
| `auth.access_token` | string | JWT access token |
| `auth.refresh_token` | string | JWT refresh token |
| `auth.expires_at` | string | Token expiration (RFC3339) |
| `api_url` | string | Server API URL |
| `install_path` | string | OMNI installation path |

## Loading Configuration

```go
cfg, err := config.Load()
if err != nil {
    // Handle error
}

if cfg.IsAuthenticated() {
    fmt.Printf("Logged in as %s\n", cfg.Auth.Username)
}
```

## Config API

```go
type Config struct {
    Auth        *AuthConfig
    APIUrl      string
    InstallPath string
}

func Load() (*Config, error)
func (c *Config) Save() error
func (c *Config) IsAuthenticated() bool
func (c *Config) ClearAuth() error
```

## Environment Variables

| Variable | Description | Default |
|----------|-------------|---------|
| `OMNIFORGE_API_URL` | Override API URL | From config |
| `OMNIFORGE_BUILD_MODE` | dev or release | auto-detect |
| `SCRIPTS_ROOT` | Override scripts directory | auto-detect |

## Global Flags

```bash
omniforge --config /path/to/config.json <command>
omniforge -v <command>  # Verbose output
```

## File Permissions

Configuration files are created with restricted permissions:
- Config file: `0600` (user read/write only)
- Directories: `0700` (user access only)

## Default Values

| Setting | Default |
|---------|---------|
| API URL | https://api.omniforge.io |
| Install Path | /opt/hostagent-server |
| Log Dir | ~/.omniforge/logs |
| Cache Dir | ~/.omniforge/cache |

## Implementation Files

| File | Purpose |
|------|---------|
| `internal/config/config.go` | Configuration management |
| `internal/config/auth.go` | Auth token handling |
| `internal/config/paths.go` | Path resolution |
