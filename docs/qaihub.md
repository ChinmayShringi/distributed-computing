# Qualcomm AI Hub CLI Setup

Qualcomm AI Hub (`qai-hub`) is a CLI client for the [Qualcomm AI Hub](https://aihub.qualcomm.com/) cloud service. It compiles, profiles, and deploys AI models targeting Qualcomm devices and runtimes (LiteRT/TFLite, ONNX Runtime, etc.).

**This runs from any Windows x86 host** — no local Qualcomm NPU or special hardware is required. The actual compilation and profiling happen in the cloud.

## Prerequisites

- Python 3.8+ installed and on PATH
- A Qualcomm AI Hub account and API token (get one at https://aihub.qualcomm.com/). A default token is hardcoded in the setup script.

## Quick Setup (Windows)

Run the setup script (a default API token is built in):

```powershell
powershell -ExecutionPolicy Bypass -File scripts/windows/setup_qaihub.ps1
```

To override the token, set `QAI_HUB_API_TOKEN` before running:

```cmd
set QAI_HUB_API_TOKEN=your_token_here
powershell -ExecutionPolicy Bypass -File scripts/windows/setup_qaihub.ps1
```

The script creates a Python venv at `.venv-qaihub/`, installs `qai-hub`, configures the API token, and runs `qai-hub list-devices` to verify.

## Manual Setup

```bash
# 1. Create and activate a virtual environment
python -m venv .venv-qaihub
.venv-qaihub\Scripts\Activate.ps1   # PowerShell
# or: .venv-qaihub\Scripts\activate.bat   # CMD

# 2. Install qai-hub
python -m pip install --upgrade pip
pip install qai-hub

# 3. Configure with your API token
qai-hub configure --api_token YOUR_TOKEN

# 4. Verify
qai-hub list-devices
```

## Smoke Test

Run the smoke test to confirm everything works:

```powershell
powershell -ExecutionPolicy Bypass -File scripts/windows/qaihub_smoketest.ps1
```

This activates the venv and runs `qai-hub list-devices`, failing if the command errors or returns empty output.

## CLI Commands

### Doctor

Check qai-hub installation and configuration:

```bash
go run ./cmd/edgecli qaihub doctor
```

Output includes:
- qai-hub binary availability and version
- `QAI_HUB_API_TOKEN` environment variable status
- CLI functionality check

Use `--json` for machine-readable output:

```bash
go run ./cmd/edgecli qaihub doctor --json
```

### Compile

Compile an ONNX model for Qualcomm devices:

```bash
go run ./cmd/edgecli qaihub compile \
  --onnx /path/to/model.onnx \
  --target "Samsung Galaxy S24" \
  --runtime precompiled_qnn_onnx \
  --out ./artifacts/qaihub
```

Flags:
- `--onnx` (required): Path to ONNX model file
- `--target` (required): Target device (e.g., "Samsung Galaxy S24")
- `--runtime`: Target runtime (default: "precompiled_qnn_onnx")
- `--out`: Output directory (default: `./artifacts/qaihub/<timestamp>`)
- `--json`: Output as JSON

### List Devices

The CLI client also provides a simple wrapper:

```bash
go run ./cmd/client qaihub-list-devices
```

This shells out to `qai-hub list-devices` (tries the `.venv-qaihub` venv first, then falls back to PATH). No gRPC server required.

## Web UI Integration

When the web server is running, you can also use the Web UI to:

1. **Run Doctor**: Click "Run Doctor" in the Qualcomm Model Pipeline card to check qai-hub status
2. **Compile Model**: Enter ONNX path and target device, then click "Compile Model"

REST endpoints:
- `GET /api/qaihub/doctor` - Check installation status
- `POST /api/qaihub/compile` - Compile a model

## Environment Variables

| Variable | Description |
|----------|-------------|
| `QAI_HUB_API_TOKEN` | API token for Qualcomm AI Hub authentication |

## Limitations

**Important**: Compiled models target Qualcomm Snapdragon devices and **cannot run locally on Mac or non-Qualcomm hardware**. The qai-hub CLI is a cloud API client that submits jobs to Qualcomm's cloud infrastructure.

For local chat/inference on Mac, use Ollama or LM Studio instead (see [chat.md](chat.md)).

## What This Is For

- **Compiling** models for Qualcomm chipsets (Snapdragon, etc.)
- **Profiling** inference performance on real cloud-hosted devices
- **Deploying** optimized models targeting runtimes like LiteRT/TFLite and ONNX Runtime

This does **not** provide local NPU acceleration on AMD or non-Qualcomm hardware. It is a cloud API client.
