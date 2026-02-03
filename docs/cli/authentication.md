# Authentication

## Overview

OmniForge CLI requires authentication to access cloud features like AI-assisted diagnostics, artifact downloads, and incident uploads. Authentication uses JWT tokens stored locally.

## Usage

### Login

```bash
# Interactive login (prompts for credentials)
omniforge login

# Non-interactive login (for scripts)
omniforge login -u USERNAME -p PASSWORD
omniforge login --username USERNAME --password PASSWORD
```

### Logout

```bash
omniforge logout
```

### Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--username` | `-u` | Username for authentication |
| `--password` | `-p` | Password for authentication |

## Behavior

### Login Flow

1. Check if already authenticated - if so, notify user and exit
2. Get username from flag or prompt interactively
3. Get password from flag or prompt (hidden input using terminal)
4. Send credentials to server API `/v1/auth/login`
5. Receive JWT tokens (access + refresh)
6. Store tokens in config file

### Token Storage

Tokens are stored in `~/.omniforge/config.json`:

```json
{
  "auth": {
    "username": "user@example.com",
    "access_token": "eyJ...",
    "refresh_token": "eyJ...",
    "expires_at": "2026-01-28T12:15:00Z"
  }
}
```

### Token Lifecycle

- **Access Token**: Short-lived (15 minutes), used for API requests
- **Refresh Token**: Long-lived (7 days), used to obtain new access tokens
- **Auto-refresh**: API client automatically refreshes tokens when expired

### Session Check

```go
cfg.IsAuthenticated()  // Returns true if valid tokens exist
```

## Authentication Required For

| Feature | Requires Auth |
|---------|---------------|
| `omniforge doctor` (online) | Yes |
| `omniforge doctor --offline` | No |
| `omniforge chat` | Yes |
| `omniforge artifact download` | Yes |
| `omniforge incident create` | Yes (for upload) |
| `omniforge incident list` | Yes |
| `omniforge prereq` | No |
| `omniforge install --tarball` | No |

## Error Handling

| Error | Cause | Solution |
|-------|-------|----------|
| "login failed" | Invalid credentials | Check username/password |
| "already logged in" | Session exists | Run `omniforge logout` first |
| "token expired" | Session timeout | Re-run `omniforge login` |

## Security

- Passwords are never stored locally
- Password input is hidden (not echoed to terminal)
- Tokens are stored with user-only file permissions
- `logout` command clears all stored credentials

## Implementation Files

| File | Purpose |
|------|---------|
| `cmd/spotlight/commands/root.go` | Login/logout commands |
| `internal/config/config.go` | Token storage and management |
| `internal/api/client.go` | API client with auth |
