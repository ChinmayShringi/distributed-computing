# Pre-Work Checklist

Read this file BEFORE starting any task. These steps ensure you understand the current state and avoid breaking changes.

## 1. Understand the Request

- Clarify ambiguous requirements with the user before writing code.
- Identify which components are affected (server, web, client, brain, proto).
- Determine if this is a new feature, bug fix, or refactor.

## 2. Check Repository State

```bash
git status              # Any uncommitted changes?
git log --oneline -5    # Recent commits for context
go build ./...          # Does the codebase currently compile?
make test               # Do existing tests pass?
```

If the build or tests are broken before you start, flag it to the user immediately.

## 3. Read Relevant Code

Do not propose changes to files you haven't read. For any task:

- Read the files you plan to modify.
- Read their callers/callees to understand impact.
- Check for platform-specific files (build tags: `_windows.go`, `_stub.go`).

## 4. Check for Cross-Cutting Concerns

Changes often ripple across layers. Use this map:

| If you change... | Also check/update... |
|---|---|
| `proto/orchestrator.proto` | Run `make proto`, update `cmd/server/main.go` handlers, `cmd/web/main.go` handlers, `cmd/web/index.html` UI |
| `internal/brain/brain.go` (types/signatures) | `internal/brain/brain_windows.go`, `internal/brain/brain_stub.go`, `cmd/server/main.go` call sites |
| `internal/jobs/jobs.go` (exported API) | `cmd/server/main.go` (SubmitJob, PreviewPlan) |
| `cmd/server/main.go` (new RPC) | `cmd/web/main.go` (new REST endpoint), `cmd/web/index.html` (new UI), `docs/features/grpc.md` |
| `cmd/web/main.go` (new endpoint) | `cmd/web/index.html` (JS to call it), `docs/features/web-ui.md` |
| Device registration fields | `proto/orchestrator.proto`, `cmd/server/main.go` (getSelfDeviceInfo), `cmd/web/main.go` (DeviceResponse), `cmd/web/index.html` (display) |

## 5. Identify Test Strategy

- Are there existing tests for the area you're changing? (`go test -v ./internal/...`)
- Will you need to test on Windows? (SSH access: `sshuser@10.20.38.80`, pass: `root`)
- Will you need to run both server + web to verify? (ports 50051 + 8080)

## 6. Plan Before Coding

For non-trivial changes (3+ files, new feature, architectural change):

- Write a plan listing files to modify and the change in each.
- Identify the execution order (e.g., proto first, then Go, then UI).
- Consider what can be done in parallel vs. what has sequential dependencies.
