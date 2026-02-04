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

## Convenience Command

The CLI client provides an optional wrapper:

```bash
go run ./cmd/client qaihub-list-devices
```

This shells out to `qai-hub list-devices` (tries the `.venv-qaihub` venv first, then falls back to PATH). No gRPC server required.

## What This Is For

- **Compiling** models for Qualcomm chipsets (Snapdragon, etc.)
- **Profiling** inference performance on real cloud-hosted devices
- **Deploying** optimized models targeting runtimes like LiteRT/TFLite and ONNX Runtime

This does **not** provide local NPU acceleration on AMD or non-Qualcomm hardware. It is a cloud API client.
