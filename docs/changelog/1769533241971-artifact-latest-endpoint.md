# Changelog: GitHub Release to S3 Artifact Endpoint

**Date:** 2026-01-27
**Type:** Feature Addition

## Summary

Implemented a new backend endpoint `GET /v1/artifact/latest` that fetches the latest GitHub release tarball, uploads it to S3, and returns a presigned download URL. This enables secure, authenticated access to release artifacts without exposing GitHub PAT to clients.

## Changes Overview

### New Files

| File | Description |
|------|-------------|
| `server/internal/config/config.go` | Configuration management via environment variables |
| `server/scripts/test-artifact-latest.sh` | Validation script for testing the endpoint |
| `server/.env` | Environment configuration template |

### Modified Files

| File | Changes |
|------|---------|
| `server/cmd/server/main.go` | Added route and handler for `/v1/artifact/latest` |
| `server/cmd/lambda/artifacts/main.go` | Added Lambda handler for `/v1/artifact/latest` |
| `server/internal/artifacts/github.go` | Added asset selection and SHA256 extraction functions |
| `server/internal/storage/s3.go` | Added streaming upload with hash computation |

## Technical Details

### New Endpoint

```
GET /v1/artifact/latest
Authorization: Bearer <jwt-token>
```

**Response:**
```json
{
  "version": "v1.2.3",
  "asset_name": "hostagent-server-linux-amd64.tar.gz",
  "sha256": "abc123def456...",
  "url": "https://bucket.s3.amazonaws.com/releases/v1.2.3/...",
  "expires_in": 600
}
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `GITHUB_OWNER` | `spotlight-omni` | GitHub organization/owner |
| `GITHUB_REPO` | `omni-releases` | GitHub repository name |
| `GITHUB_PAT` | *(from Secrets Manager)* | GitHub Personal Access Token |
| `S3_BUCKET` | `omniforge-artifacts-840864614419` | S3 bucket for artifacts |
| `S3_PREFIX` | `releases/` | S3 key prefix |
| `PRESIGN_TTL_SECONDS` | `600` | Presigned URL expiry (seconds) |

### Flow Diagram

```
Client                    Server                    GitHub                S3
  |                         |                         |                    |
  |-- GET /artifact/latest->|                         |                    |
  |                         |-- Get latest release -->|                    |
  |                         |<-- Release metadata ----|                    |
  |                         |                         |                    |
  |                         |-- Check if exists ----->|                    |
  |                         |<-- exists: true/false --|                    |
  |                         |                         |                    |
  |                         |  (if not exists)        |                    |
  |                         |-- Download asset ------>|                    |
  |                         |<-- Asset bytes ---------|                    |
  |                         |                         |                    |
  |                         |-- Upload + SHA256 --------------------->|    |
  |                         |<-- Upload complete ---------------------|    |
  |                         |                         |                    |
  |                         |-- Generate presigned URL ------------->|    |
  |                         |<-- Presigned URL -----------------------|    |
  |                         |                         |                    |
  |<-- JSON response -------|                         |                    |
```

### Key Functions Added

#### `server/internal/config/config.go`
- `Load()` - Reads configuration from environment variables with defaults

#### `server/internal/artifacts/github.go`
- `InitGitHubWithPAT(pat string)` - Initialize GitHub client with explicit PAT
- `SelectHostagentAsset(release *Release) (*Asset, error)` - Find hostagent-server tar.gz
- `GetSHA256FromSums(ctx, owner, repo, release, filename) (string, error)` - Extract checksum from SHA256SUMS file

#### `server/internal/storage/s3.go`
- `ReleaseKey(prefix, version, filename) string` - Generate S3 key for releases
- `UploadWithHash(ctx, bucket, key, body, contentType) (*UploadResult, error)` - Upload while computing SHA256
- `ObjectExists(ctx, bucket, key) (bool, error)` - Check if object exists in S3

## Testing

### Local Testing

```bash
# Start the server
cd server
source .env
go run ./cmd/server

# In another terminal, get a JWT token
TOKEN=$(curl -s -X POST http://localhost:3000/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"password"}' | jq -r '.access_token')

# Test the endpoint
export TOKEN
./scripts/test-artifact-latest.sh
```

### Expected Output

```
=== Testing GET /v1/artifact/latest ===
Server: http://localhost:3000

1. Calling endpoint...
   HTTP Status: 200
   Response:
   {
     "version": "v0.1.0",
     "asset_name": "hostagent-server.tar.gz",
     "sha256": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
     "url": "https://omniforge-artifacts-840864614419.s3.amazonaws.com/...",
     "expires_in": 600
   }

2. Verifying presigned URL with HEAD request...
   Presigned URL is valid (HTTP 200)

=== Test completed successfully ===
```

## Security Considerations

1. **JWT Authentication Required** - Endpoint requires valid Bearer token
2. **GitHub PAT Not Exposed** - Server handles GitHub auth, clients never see PAT
3. **Presigned URLs Expire** - Default 10-minute expiry limits exposure window
4. **S3 Caching** - Assets uploaded once, subsequent requests serve from S3

## Migration Notes

- No database migrations required
- Existing S3 bucket infrastructure is reused
- GitHub PAT can be set via:
  - `GITHUB_PAT` environment variable (preferred for local dev)
  - AWS Secrets Manager `omniforge/github-pat` (production)

## Related Issues

- Implements artifact distribution for CLI `install` command
- Supports offline bundle preparation workflow
