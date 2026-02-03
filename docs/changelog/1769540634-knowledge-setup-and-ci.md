# Changelog: Knowledge Stack Setup and CI/CD Updates

**Date:** 2026-01-27
**Type:** Infrastructure
**Timestamp:** 1769540634

## Summary

Added comprehensive local development setup, verification scripts, and CI/CD pipeline updates for the pgvector RAG knowledge stack. This enables one-command local development with Postgres+pgvector, full verification scripts, and CI that builds all server binaries.

## Features

### 1. Local Development Infrastructure
- Docker Compose configuration for Postgres 15 with pgvector (`docker-compose.dev.yml`)
- New Make targets for streamlined local development:
  - `dev-db-up` - Start local Postgres with pgvector
  - `dev-db-down` - Stop database
  - `dev-db-reset` - Reset database with fresh data
  - `migrate-knowledge-dev` - Run pgvector migration
  - `server-run` - Build and run HTTP server
  - `mcp-knowledge-run` - Run MCP knowledge server
  - `verify-knowledge` - Run full verification script

### 2. Verification Script
- Full end-to-end verification of knowledge stack (`verify_knowledge_stack.sh`)
- Tests database, migrations, server, auth, seeding, search, chat, and MCP
- Colored PASS/FAIL summary output
- Prerequisite checks (docker, psql, curl, jq)

### 3. PDF Upload Script
- Uploads PDF documents from docs/ (`upload_and_ingest_docs.sh`)
- Supports `Getting Started.pdf` and `Hardware Specs.pdf`
- Handles admin gating gracefully (403 errors)
- Polls for indexing completion with timeout
- Handles duplicate detection (409 responses)

### 4. MCP Server Demo
- Example script demonstrating MCP knowledge server usage (`mcp_knowledge_demo.sh`)
- Shows initialize, tools/list, and tools/call patterns
- Includes usage instructions for interactive querying

### 5. CI/CD Pipeline
- New `ci.yml` workflow with:
  - Unit tests for CLI and server modules
  - Build artifacts: server, mcp-knowledge, seed-knowledge, CLI (linux amd64/arm64)
  - Go module caching for faster builds
  - Smoke test job with Postgres container
  - Integration tests gated behind `workflow_dispatch` + `run_integration` input
  - No real OpenAI keys required for PR CI
- Updated `deploy.yml` to build and deploy mcp-knowledge and seed-knowledge

### 6. Documentation
- Comprehensive RAG setup guide (`docs/knowledge-rag.md`)
- Environment variables reference
- API endpoint documentation
- MCP server usage guide
- Troubleshooting section
- Ubuntu container testing commands

## Files Changed

| File | Type | Description |
|------|------|-------------|
| `server/docker-compose.dev.yml` | New | Postgres 15 + pgvector for local dev |
| `server/Makefile` | Updated | Added dev-db, server-run, verify targets |
| `server/scripts/verify_knowledge_stack.sh` | New | Full verification script |
| `server/scripts/upload_and_ingest_docs.sh` | New | PDF upload helper |
| `server/examples/mcp_knowledge_demo.sh` | New | MCP demo script |
| `.github/workflows/ci.yml` | New | Tests, builds, smoke test workflow |
| `.github/workflows/deploy.yml` | Updated | Added mcp-knowledge, seed-knowledge builds |
| `docs/knowledge-rag.md` | New | Setup and usage documentation |

## Verification Results

Verified on local development environment:

```
=== Verification Summary ===

PASSED:
  - Docker installed
  - psql installed
  - curl installed
  - jq installed
  - Database connection
  - pgvector extension installed
  - Knowledge tables created
  - HNSW vector index created
  - Server binary built
  - MCP knowledge binary built
  - Seed knowledge binary built
```

## How to Test

### Quick Start (Local Development)

```bash
# 1. Start the dev database
cd server
make dev-db-up

# 2. Run migrations
make migrate-knowledge-dev

# 3. Start the server (in terminal 1)
export OPENAI_API_KEY=sk-your-key
make server-run

# 4. Run verification (in terminal 2)
make verify-knowledge
```

### Full Verification Script

```bash
cd server
export OPENAI_API_KEY=sk-your-key
export ADMIN_USER=admin
export ADMIN_PASS=yourpassword
./scripts/verify_knowledge_stack.sh
```

### Upload PDFs

```bash
# Get JWT token
TOKEN=$(curl -s -X POST http://localhost:3000/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"yourpass"}' | jq -r '.access_token')

# Upload PDFs
./scripts/upload_and_ingest_docs.sh http://localhost:3000 $TOKEN
```

### Test MCP Server

```bash
export OMNIFORGE_JWT_TOKEN=$TOKEN
./examples/mcp_knowledge_demo.sh
```

### Ubuntu Container CLI Test

```bash
# Start Ubuntu container
docker run -it --rm --network host ubuntu:22.04 bash

# Inside container
apt update && apt install -y curl jq

# Download CLI (from your server)
curl -o omniforge http://localhost:3000/cli/linux/amd64
chmod +x omniforge

# Login
./omniforge login -u admin -s http://localhost:3000

# Test chat
./omniforge chat
# Type: "How do I fix Docker permission issues?"
# Verify: Response should include RAG context from knowledge base

# Test tool approval UI
# Type: run ls
# Choose option 1
# Verify: Command output displayed

# Test --no-color flag
./omniforge chat --no-color
# Verify: No ANSI color codes in output
```

## CI/CD Verification

The CI workflow verifies:
1. Unit tests pass (CLI and server)
2. All binaries build successfully:
   - `omniforge-linux-amd64`
   - `omniforge-linux-arm64`
   - `server`
   - `mcp-knowledge`
   - `seed-knowledge`
3. Smoke test with Postgres container:
   - pgvector extension installed
   - Knowledge tables created
   - HNSW index created
   - Server health check passes

To trigger integration tests:
```bash
gh workflow run ci.yml -f run_integration=true
```

## Environment Variables Reference

### Required for Production
```bash
OPENAI_API_KEY=sk-...           # OpenAI API key for embeddings
KNOWLEDGE_ADMIN_USERS=admin     # Comma-separated admin usernames
DATABASE_URL=postgresql://...   # Or individual DB_* vars
```

### Development Defaults
```bash
DB_HOST=localhost
DB_PORT=5432
DB_USER=omniforge
DB_PASSWORD=devpassword123
DB_NAME=omniforge
KNOWLEDGE_ADMIN_USERS=admin
PORT=3000
```

## Migration Notes

The pgvector migration (`002_knowledge_pgvector.sql`) requires:
1. Initial migration (`001_initial.sql`) to be run first (for uuid-ossp extension)
2. PostgreSQL 15+ with pgvector extension available
3. For RDS: PostgreSQL 15.2+ or Aurora 15.3+ with pgvector in parameter group
