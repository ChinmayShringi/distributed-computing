# Changelog: Server-Side RAG with pgvector and MCP Knowledge Tools

**Date:** 2026-01-27
**Type:** Feature
**Timestamp:** 1769536737

## Summary

Implemented server-side Retrieval-Augmented Generation (RAG) using PostgreSQL pgvector for vector storage and OpenAI embeddings. Added an MCP server exposing knowledge tools for programmatic access to the knowledge base.

## Features

### 1. Knowledge Base Storage
- PostgreSQL pgvector extension for vector similarity search
- `knowledge_documents` table for document metadata
- `knowledge_chunks` table with 1536-dimension embeddings (text-embedding-3-small)
- HNSW index for fast approximate nearest neighbor search

### 2. Document Management API
- `POST /v1/knowledge/upload` - Upload documents (PDF, MD, TXT)
- `POST /v1/knowledge/ingest/{id}` - Process and index documents
- `GET /v1/knowledge/status/{id}` - Check document status
- `POST /v1/knowledge/search` - Semantic search
- `GET /v1/knowledge/documents` - List all documents

### 3. RAG Integration
- Automatic context injection in chat responses
- Source attribution with relevance scores
- Configurable top-K retrieval

### 4. MCP Server
- Standalone binary: `mcp-knowledge`
- Tools: `knowledge_search`, `knowledge_upload`, `knowledge_ingest`, `knowledge_status`, `knowledge_list`
- JSON-RPC 2.0 over stdio protocol
- JWT authentication

### 5. Seed Runner
- CLI tool for bulk document ingestion
- Supports MD and TXT files
- SHA256-based deduplication

## Files Changed

| File | Type | Description |
|------|------|-------------|
| `server/migrations/002_knowledge_pgvector.sql` | New | pgvector extension and knowledge tables |
| `server/internal/db/knowledge.go` | New | CRUD operations and vector search |
| `server/internal/knowledge/embeddings.go` | New | OpenAI embeddings client |
| `server/internal/knowledge/chunker.go` | New | Deterministic text chunking |
| `server/internal/knowledge/chunker_test.go` | New | Chunking unit tests |
| `server/internal/knowledge/retriever.go` | New | Vector similarity search |
| `server/internal/knowledge/ingest.go` | New | Document ingestion pipeline |
| `server/cmd/server/main.go` | Updated | Added knowledge HTTP endpoints |
| `server/internal/openai/planner.go` | Updated | RAG context injection in Chat() |
| `server/internal/config/config.go` | Updated | Knowledge configuration env vars |
| `server/internal/storage/s3.go` | Updated | Added DownloadFile function |
| `server/cmd/mcp-knowledge/main.go` | New | MCP server implementation |
| `server/cmd/seed-knowledge/main.go` | New | Seed runner CLI |
| `server/database/seed/omniforge-faq.md` | New | Sample FAQ document |
| `server/database/seed/docker-troubleshooting.txt` | New | Sample troubleshooting guide |
| `server/Makefile` | Updated | Added build targets |

## Environment Variables

### New Variables

```bash
# Embeddings configuration
OPENAI_EMBEDDINGS_MODEL=text-embedding-3-small  # default
OPENAI_EMBEDDINGS_DIMENSIONS=1536               # default, optional

# Knowledge configuration
KNOWLEDGE_ADMIN_USERS=admin,devops              # comma-separated usernames
KNOWLEDGE_CHUNK_SIZE=2000                       # characters, default
KNOWLEDGE_CHUNK_OVERLAP=200                     # characters, default
KNOWLEDGE_TOP_K=5                               # retrieval count, default
KNOWLEDGE_S3_BUCKET=omniforge-knowledge         # S3 bucket for documents

# MCP server (for mcp-knowledge binary)
OMNIFORGE_API_URL=http://localhost:3000
OMNIFORGE_JWT_TOKEN=eyJ...
```

## Dependencies Added

```go
github.com/ledongthuc/pdf    // PDF text extraction
github.com/pgvector/pgvector-go  // pgvector Go types
```

## Migration Instructions

### 1. Run Migration

```bash
# Ensure pgvector extension is available on your PostgreSQL instance
# For RDS: pgvector is available on PostgreSQL 15.2+ and Aurora 15.3+

# Run migration
make migrate-knowledge DATABASE_URL="postgresql://user:pass@host:5432/dbname?sslmode=require"
```

### 2. Configure Environment

```bash
# Add to your environment or .env file
export KNOWLEDGE_ADMIN_USERS=admin
export KNOWLEDGE_S3_BUCKET=your-knowledge-bucket
```

### 3. Seed Initial Documents

```bash
# Build seed runner
make seed-knowledge

# Get a JWT token (login first)
TOKEN=$(curl -s -X POST http://localhost:3000/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"yourpass"}' | jq -r '.access_token')

# Run seeder
./build/seed-knowledge -dir=database/seed -api=http://localhost:3000 -token=$TOKEN
```

## How to Test

### 1. Upload a Document

```bash
curl -X POST http://localhost:3000/v1/knowledge/upload \
  -H "Authorization: Bearer $TOKEN" \
  -F "file=@mydocument.pdf" \
  -F "title=My Document"
```

Response:
```json
{"id": "uuid", "title": "My Document", "status": "uploaded"}
```

### 2. Ingest the Document

```bash
curl -X POST http://localhost:3000/v1/knowledge/ingest/DOCUMENT_ID \
  -H "Authorization: Bearer $TOKEN"
```

Response:
```json
{"document_id": "uuid", "chunk_count": 15, "status": "indexed"}
```

### 3. Search the Knowledge Base

```bash
curl -X POST http://localhost:3000/v1/knowledge/search \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"query": "How do I restart Docker?", "k": 5}'
```

Response:
```json
{
  "chunks": [
    {"content": "...", "score": 0.87, "document_title": "Docker Troubleshooting"}
  ],
  "sources": [
    {"document_id": "uuid", "document_title": "Docker Troubleshooting", "score": 0.87}
  ]
}
```

### 4. Chat with RAG Context

```bash
curl -X POST http://localhost:3000/v1/chat \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [{"role": "user", "content": "How do I fix Docker permission issues?"}],
    "system_info": "Ubuntu 22.04"
  }'
```

The chat response will include relevant knowledge base context and source references.

### 5. MCP Server Example

```bash
# Build MCP server
make mcp-knowledge

# Run MCP server (receives JSON-RPC on stdin, outputs on stdout)
export OMNIFORGE_API_URL=http://localhost:3000
export OMNIFORGE_JWT_TOKEN=$TOKEN
./build/mcp-knowledge

# Send initialize request
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}' | ./build/mcp-knowledge

# List tools
echo '{"jsonrpc":"2.0","id":2,"method":"tools/list"}' | ./build/mcp-knowledge

# Search knowledge
echo '{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"knowledge_search","arguments":{"query":"docker restart","k":3}}}' | ./build/mcp-knowledge
```

## Security Notes

- Upload and ingest endpoints require admin access (configured via `KNOWLEDGE_ADMIN_USERS`)
- All endpoints require JWT authentication
- Documents are stored in S3 with user-scoped paths
- SHA256 deduplication prevents duplicate uploads
- MCP server requires JWT token via environment variable

## Known Limitations

- PDF extraction uses basic text extraction (no OCR)
- Maximum file size: 10MB
- Embedding model fixed to OpenAI (no local model support yet)
- MCP server does not support file upload directly (use HTTP endpoint)
