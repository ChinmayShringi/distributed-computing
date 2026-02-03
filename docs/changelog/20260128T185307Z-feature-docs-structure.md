# Changelog: Feature Documentation Structure

**Date:** 2026-01-28
**Type:** Documentation

## Summary

Introduced a feature documentation structure where detailed living docs live in `docs/<area>/` and changelog entries are short audit logs that link to them.

## Motivation

Having detailed documentation in changelog entries led to:
- Long, hard-to-scan changelog files
- Duplicated information across README and changelog
- No single source of truth for feature behavior

The new structure separates concerns:
- **Changelog** (`docs/changelog/`) - Append-only audit log of changes
- **Feature Docs** (`docs/<area>/`) - Living documentation maintained over time

## Changes

### New Feature Docs Created
- `docs/cli/dangerous-mode.md` - Dangerous mode architecture and usage
- `docs/cli/docker-script-installers.md` - Docker installer implementation
- `docs/cli/hostagent-script-installer.md` - HostAgent installer implementation

### Changelog Entries Shortened
- `20260128-allow-dangerous-mode.md` - Reduced from 175 to 35 lines
- `20260128T120000Z-docker-script-installers.md` - Reduced from 594 to 51 lines

### README Updated
- Added "Documentation" section explaining the structure
- Simplified "Dangerous Mode" section with link to feature doc

## Documentation Areas

| Area | Description |
|------|-------------|
| `docs/cli/` | CLI features and behavior |
| `docs/server/` | Server-side features |
| `docs/frontend/` | Frontend features |
| `docs/mcp/` | MCP server features |

## Conventions

**Changelog entries** should:
- Be short (30-60 lines)
- Include Date, Type, Summary, Changes
- Link to feature docs for details

**Feature docs** should:
- Be the single source of truth
- Be updated when features change
- Include Overview, Usage, Architecture, Testing sections
