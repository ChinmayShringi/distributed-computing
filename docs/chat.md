# Local Chat Runtime

The Web UI includes a chat interface powered by a local LLM runtime (Ollama or LM Studio). This provides actual chat responses running entirely on your Mac.

## Overview

The chat runtime is **completely separate** from the Qualcomm Model Pipeline:
- **Qualcomm Pipeline**: Model compilation verification (cloud-based, cannot run locally)
- **Chat Runtime**: Local LLM inference for actual chat responses

This separation ensures you have a working chatbot on Mac even though Qualcomm-compiled models cannot run on non-Snapdragon hardware.

## Supported Providers

### Echo (Mock/Testing)

For testing when no LLM runtime is available, use the echo provider:

```bash
export CHAT_PROVIDER=echo
```

This echoes back user messages without requiring any external LLM. Useful for:
- Testing the chat integration
- Development when Ollama/LM Studio isn't working
- CI/CD environments

### Ollama (Default)

[Ollama](https://ollama.ai/) is a lightweight LLM runtime for Mac/Linux/Windows.

**Installation:**

```bash
# Mac
brew install ollama

# Windows
winget install Ollama.Ollama

# Or download from https://ollama.ai/download
```

**Start Ollama and pull a model:**

```bash
# Start Ollama service
ollama serve

# Pull a model (see recommendations below)
ollama pull llama3.2:3b
```

**Recommended Models:**

| Model | Size | Chat | Tool Calling | Notes |
|-------|------|------|--------------|-------|
| `llama3.2:3b` | 2.0GB | Good | Yes | Best for agent use |
| `phi3:mini` | 2.2GB | Good | No | Fast, but no tool support |
| `mistral:7b` | 4.1GB | Excellent | Yes | Higher quality |
| `qwen2.5:7b` | 4.7GB | Excellent | Yes | Strong reasoning |

**Important:** For agent functionality (tool calling), use `llama3.2:3b`, `mistral:7b`, or `qwen2.5:7b`. Models like `phi3:mini` do not support tool calling.

**Environment variables:**

```bash
export CHAT_PROVIDER=ollama
export CHAT_BASE_URL=http://localhost:11434
export CHAT_MODEL=llama3.2:3b  # Use a model with tool support for agent
```

### OpenAI-Compatible (LM Studio)

[LM Studio](https://lmstudio.ai/) provides an OpenAI-compatible API for local models.

**Setup:**

1. Download LM Studio from https://lmstudio.ai/
2. Download a model (e.g., Llama, Mistral)
3. Start the local server (defaults to port 1234)

**Environment variables:**

```bash
export CHAT_PROVIDER=openai
export CHAT_BASE_URL=http://localhost:1234
export CHAT_MODEL=local-model
export CHAT_API_KEY=optional-if-required
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `CHAT_PROVIDER` | `ollama` | Provider type: `ollama` or `openai` |
| `CHAT_BASE_URL` | `http://localhost:11434` (Ollama) | Base URL for the LLM API |
| `CHAT_MODEL` | `llama2` (Ollama) | Model name to use |
| `CHAT_API_KEY` | (none) | API key (optional, for OpenAI-compatible) |
| `CHAT_TIMEOUT_SECONDS` | `60` | Request timeout |

## REST API

### Health Check

```bash
curl http://localhost:8080/api/chat/health
```

Response:
```json
{
  "ok": true,
  "provider": "ollama",
  "base_url": "http://localhost:11434",
  "model": "llama2"
}
```

### Send Chat Message

```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {"role": "user", "content": "Hello!"}
    ]
  }'
```

Response:
```json
{
  "reply": "Hello! How can I help you today?"
}
```

The API supports multi-turn conversations by including the full message history:

```bash
curl -X POST http://localhost:8080/api/chat \
  -H "Content-Type: application/json" \
  -d '{
    "messages": [
      {"role": "user", "content": "My name is Alice"},
      {"role": "assistant", "content": "Nice to meet you, Alice!"},
      {"role": "user", "content": "What is my name?"}
    ]
  }'
```

## Web UI

The Web UI includes a "Local Chat Runtime" card with:

1. **Status indicator**: Shows whether the chat runtime is available
2. **Refresh Status**: Re-check the runtime health
3. **Chat interface**: Send messages and see responses

The chat maintains conversation history within the browser session.

## Troubleshooting

### "Runtime unavailable"

1. Ensure Ollama is running: `ollama serve`
2. Verify the model is downloaded: `ollama list`
3. Check the base URL is correct
4. Try: `curl http://localhost:11434` (should return "Ollama is running")

### "Chat failed: connection refused"

The LLM server is not running or is on a different port. Start Ollama or LM Studio and verify the port.

### Slow responses

Local LLM inference depends on your hardware. Smaller models (e.g., 7B parameters) are faster than larger ones. Consider using quantized models for better performance.

### Ollama Metal errors on Mac

If you see errors like `MTLLibraryErrorDomain` or `failed to initialize Metal backend`:

1. **Use echo provider for testing**: `CHAT_PROVIDER=echo make web`
2. **Try an older Ollama version**: `brew install ollama@0.1.x`
3. **Use LM Studio instead**: `CHAT_PROVIDER=openai CHAT_BASE_URL=http://localhost:1234 make web`
4. **Try CPU-only mode**: `OLLAMA_NO_METAL=1 ollama serve` (may not work on all versions)

This is a known issue with certain Ollama versions and macOS GPU drivers.

## Integration with Qualcomm Pipeline

The chat runtime and Qualcomm pipeline are intentionally separate:

1. **Qualcomm Pipeline**: Use to verify model compilation workflow
2. **Chat Runtime**: Use for actual chat functionality

This design allows you to:
- Verify your Qualcomm AI Hub setup works (compilation jobs submit successfully)
- Have a working chatbot for actual use (via local Ollama/LM Studio)
- See both statuses clearly in the UI

In production on Qualcomm devices, you would replace the local runtime with QNN-compiled models. On Mac (development), the local runtime provides equivalent functionality.

## LLM Agent with Tool Calling

The agent endpoint (`/api/agent`) provides an LLM that can call tools to interact with the device mesh.

### Available Tools

| Tool | Description |
|------|-------------|
| `get_capabilities` | List all registered devices with hardware info and benchmarks |
| `execute_shell_cmd` | Execute shell commands on devices (dangerous commands blocked) |
| `get_file` | Read files from devices (full, head, tail, or range modes) |

### Agent Configuration

The agent requires a model that supports tool/function calling.

**Option 1: Ollama (Recommended)**

```bash
export CHAT_PROVIDER=ollama
export CHAT_BASE_URL=http://localhost:11434
export CHAT_MODEL=llama3.2:3b  # or mistral:7b, qwen2.5:7b

# Optional
export AGENT_MAX_ITERATIONS=8  # Max tool calling iterations
```

**Option 2: LM Studio (OpenAI-compatible)**

```bash
export CHAT_PROVIDER=openai
export CHAT_BASE_URL=http://localhost:1234
export CHAT_MODEL=qwen3-vl-8b

# Optional
export AGENT_MAX_ITERATIONS=8  # Max tool calling iterations
```

**Note:** Not all models support tool calling. See the model table above for compatible models.

### REST API

#### Agent Health Check

```bash
curl http://localhost:8080/api/agent/health
```

#### Send Agent Request

```bash
curl -X POST http://localhost:8080/api/agent \
  -H "Content-Type: application/json" \
  -d '{"message": "show me disk usage on any device"}'
```

Response:
```json
{
  "reply": "Here is the disk usage on device abc123...",
  "iterations": 2,
  "tool_calls": [
    {"iteration": 1, "tool_name": "get_capabilities", "result_len": 512},
    {"iteration": 1, "tool_name": "execute_shell_cmd", "result_len": 1024}
  ]
}
```

### CLI Usage

```bash
# Interactive mode
edgecli chat

# Single message mode
edgecli chat "list all devices"

# Custom server address
edgecli chat --web-addr localhost:8080 "run df -h on any device"
```

### Example Tool-Calling Transcript

```
User: "show me disk usage on the best laptop"

1. Agent calls: get_capabilities({include_benchmarks: true})
   -> Returns: [{device_id: "abc123", platform: "macos", ...}]

2. Agent calls: execute_shell_cmd({device_id: "abc123", command: "df -h"})
   -> Returns: {exit_code: 0, stdout: "Filesystem  Size  Used  Avail..."}

3. Agent responds:
   "Here's the disk usage on device abc123 (macOS):
    - Root filesystem: 500GB total, 200GB used (40%)
    - Data volume: 1TB total, 600GB used (60%)"
```

### Safety Controls

The `execute_shell_cmd` tool blocks dangerous commands:
- `rm -rf /`, `rm -rf ~`, `rm -rf .`
- `dd if=`, `mkfs`, `format`
- `shutdown`, `reboot`, `poweroff`
- `curl | sh`, `wget | bash`
- Fork bombs and other destructive patterns

Most read-only commands are allowed (ls, cat, df, ps, grep, etc.).

### Smoke Test

```bash
# Start servers
make server  # In terminal 1
CHAT_PROVIDER=openai CHAT_BASE_URL=http://localhost:1234 make web  # In terminal 2

# Run smoke test
./scripts/smoke/agent_smoke.sh
```
