# Changelog: Remove Lambda Code and Dead Code Cleanup

**Date:** 2026-01-27
**Type:** Cleanup / Debt Reduction

## Summary

Removed all Lambda-related code after migration to EC2. The EC2 server (`server/cmd/server/main.go`) fully implements all API routes that were previously handled by Lambda functions, making the Lambda code completely dead.

## Background

The project originally deployed API handlers as AWS Lambda functions behind API Gateway. After migrating to a single EC2-hosted HTTP server, the Lambda handlers became redundant. This cleanup removes that dead code.

## Files Deleted

### Lambda Handlers (7 files)
| File | Purpose | Replaced By |
|------|---------|-------------|
| `server/cmd/lambda/auth/main.go` | Authentication | `handleLogin`, `handleRefresh` in server |
| `server/cmd/lambda/allowlist/main.go` | Tool allowlist | `handleAllowlist` in server |
| `server/cmd/lambda/plan/main.go` | Plan generation | `handlePlan` in server |
| `server/cmd/lambda/step/main.go` | Step result recording | `handleStepResult` in server |
| `server/cmd/lambda/incidents/main.go` | Incident management | `handleIncidents` in server |
| `server/cmd/lambda/artifacts/main.go` | Artifact downloads | `handleArtifacts*` in server |
| `server/cmd/lambda/migrate/main.go` | DB migration | Never deployed (not in LAMBDA_FUNCS) |

### Empty Directories (2 directories)
| Directory | Reason |
|-----------|--------|
| `internal/app/` | Empty, never used |
| `internal/cache/` | Empty, never used |

## Files Modified

### server/Makefile
Removed Lambda build/deploy targets:
- `LAMBDA_FUNCS` variable
- `build` target (Lambda zip builds)
- `deploy` target (Lambda deployment)
- `create-functions` target (Lambda creation)

Changed `all` target from building Lambdas to building EC2 binaries:
```makefile
all: server mcp-knowledge seed-knowledge
```

### server/go.mod / go.sum
After `go mod tidy`, removed unused dependency:
- `github.com/aws/aws-lambda-go v1.47.0`

Kept AWS SDK packages (still used by EC2 server):
- `github.com/aws/aws-sdk-go-v2/*` (S3, Secrets Manager)

## Verification Results

```
Tests:
  - Root module: ok (github.com/spotlight-omni/spotlight-cli/internal/approval)
  - Server module: ok (github.com/spotlight-omni/spotlight-cli/server/internal/knowledge)

Builds:
  - server: SUCCESS
  - mcp-knowledge: SUCCESS
  - seed-knowledge: SUCCESS

Lambda References:
  - aws-lambda-go in code: NONE
  - APIGateway events in code: NONE
  - Lambda directory: DELETED
```

## How to Verify

```bash
# 1. Check no Lambda code exists
ls server/cmd/
# Should show: mcp-knowledge, seed-knowledge, server (NO lambda/)

# 2. Check no Lambda dependencies
grep "aws-lambda-go" server/go.mod
# Should return nothing

# 3. Run tests
go test ./...
cd server && go test ./...

# 4. Build all binaries
cd server && make all
# Should build: server, mcp-knowledge, seed-knowledge

# 5. Verify EC2 server still works
cd server && make dev-db-up
make server-run
# Test: curl http://localhost:3000/health
# Expected: {"status":"ok"}
```

## Impact

- **Code reduction**: ~1500 lines of Lambda handler code removed
- **Dependencies reduced**: aws-lambda-go package removed
- **Build simplified**: `make all` now builds only EC2 binaries
- **No breaking changes**: EC2 server already implements all routes
