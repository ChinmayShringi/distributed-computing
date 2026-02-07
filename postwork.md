# Post-Work Checklist

Read this file AFTER completing any task. These steps catch regressions and keep the project clean.

## 1. Code Quality

```bash
gofmt -l .              # Must produce no output (all files formatted)
go vet ./...            # Static analysis
go build ./...          # Full build including all binaries
```

Fix any issues before considering the task done.

## 2. Tests

```bash
make test               # Run all tests
```

All tests must pass. If you added new functionality, consider whether it needs tests.

## 3. Cross-Platform Build

If you changed Go code (especially `cmd/server`), verify Windows cross-compilation:

```bash
GOOS=windows GOARCH=amd64 go build -o dist/server-windows.exe ./cmd/server
```

If you changed platform-specific code (`_windows.go` / `_stub.go`), verify both build tags compile.

## 4. Deploy and Test on Windows (if applicable)

If changes affect the server binary and the Windows machine (10.20.38.80) is reachable:

```bash
# Quick connectivity check
ssh sshuser@10.20.38.80 "echo ok"

# Full deploy
./deploy-windows.sh
```

If not reachable, note it for the user rather than skipping silently.

## 5. C# CLI (if applicable)

If you changed `brain/windows-ai-cli/`:

```bash
# Build on Windows via SSH
ssh sshuser@10.20.38.80 "cd C:\Users\sshuser.Batman\windows-ai-cli && dotnet build -c Release"

# Test commands
ssh sshuser@10.20.38.80 "C:\Users\sshuser.Batman\windows-ai-cli\bin\Release\net8.0-windows10.0.22621.0\win-x64\WindowsAiCli.exe capabilities"
```

## 6. Local Integration Test

If you changed server/web endpoints, run a quick smoke test:

```bash
# Start server (terminal 1)
go run ./cmd/server

# Start web (terminal 2)
go run ./cmd/web

# Test key endpoints
curl -s http://localhost:8080/api/devices | python3 -m json.tool
curl -s -X POST http://localhost:8080/api/plan \
  -H "Content-Type: application/json" \
  -d '{"text":"collect status","max_workers":0}' | python3 -m json.tool
```

Verify the response structure matches what the UI expects.

## 7. Update Documentation

If you added or changed any of the following, the docs must be updated:

| Change | Docs to Update |
|---|---|
| New/changed gRPC RPC | `docs/features/grpc.md` |
| New/changed REST endpoint | `docs/features/web-ui.md` |
| New/changed proto fields | `docs/features/grpc.md`, `docs/features/device-registry.md` |
| Job/plan behavior | `docs/features/jobs.md` |
| New package or env var | `CLAUDE.md` |
| New binary or build step | `CLAUDE.md`, `README.md` |
| Windows AI CLI changes | `brain/windows-ai-cli/README.md` |
| New feature area | `docs/features/README.md` (feature index) |
| Project structure changes | `README.md` (project tree) |

## 8. Clean Up

- Kill any test servers you started (`lsof -ti:50051 | xargs kill`, `lsof -ti:8080 | xargs kill`)
- Remove any temp files created during testing
- Verify `git diff` shows only intentional changes (no debug prints, no commented-out code)
