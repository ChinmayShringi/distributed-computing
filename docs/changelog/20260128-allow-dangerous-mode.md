# Changelog: Dangerous Mode (`--allow-dangerous` Flag)

**Date:** 2026-01-28
**Type:** Feature

## Summary

Added `--allow-dangerous` CLI flag to `omniforge chat` and `omniforge doctor` commands. This flag enables an explicit "dangerous mode" that allows the agent to execute ANY local command without allowlist restrictions, with strong guardrails and irreversible user consent.

## Motivation

Power users and advanced troubleshooting scenarios sometimes require commands outside the standard tool allowlist. This feature provides a deliberate opt-in mechanism for unrestricted execution while maintaining full audit trails.

## Changes

- New `internal/mode/` package with ExecutionMode enum and ModeContext
- Full-screen consent warning with exact phrase requirement
- `--allow-dangerous` and `--ad` flags for chat/doctor commands
- `-y/--yes` flag for auto-confirm (scripting use)
- `ExecuteWithMode()` method in tool registry bypasses allowlist
- Server planner includes DANGEROUS mode system prompt
- Incident bundles include `dangerous_mode.json` audit file

## Files Changed

- `internal/mode/mode.go`, `internal/mode/consent.go` (NEW)
- `cmd/spotlight/commands/root.go`
- `internal/tools/registry.go`
- `internal/doctor/doctor.go`
- `internal/bundle/incident.go`
- `server/internal/openai/planner.go`

## Feature Documentation

See [dangerous-mode.md](../cli/dangerous-mode.md) for detailed architecture, security invariants, and usage guide.
