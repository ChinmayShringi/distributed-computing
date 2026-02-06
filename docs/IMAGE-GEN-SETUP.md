# Image Generation Setup — Full Stack

**Goal**: Implement distributed image generation across 3 device types with different model sizes.

---

## Architecture

```
User: "generate an image of a sunset"
        │
        ▼
┌─────────────────────────────────────────┐
│  ORCHESTRATOR (Smart Planner)           │
│  Detects "image" → IMAGE_GENERATE task  │
│  Routes to device with GPU/NPU          │
└────────────┬────────────────────────────┘
             │
    ┌────────┴────────┬─────────────────┐
    ▼                 ▼                 ▼
┌─────────┐    ┌──────────┐    ┌───────────┐
│  Mac    │    │ Windows  │    │  Arduino  │
│         │    │ Snapdragon│    │  ESP32    │
│ SD 1.5  │    │ SD XL    │    │  TinyML   │
│ 4GB RAM │    │ 8GB RAM  │    │  256KB    │
│ 30s gen │    │ 15s gen  │    │  5s gen   │
│         │    │ (NPU)    │    │ (low res) │
└─────────┘    └──────────┘    └───────────┘
```

---

## Device Setup

### 1. Mac (Stable Diffusion 1.5)

**Install Automatic1111 WebUI**:
```bash
cd ~/
git clone https://github.com/AUTOMATIC1111/stable-diffusion-webui.git
cd stable-diffusion-webui
./webui.sh
# Opens on http://localhost:7860
```

**Start EdgeMesh server with image API**:
```bash
IMAGE_API_ENDPOINT=http://localhost:7860 GRPC_ADDR=0.0.0.0:50051 go run ./cmd/server
```

**Test**:
```bash
curl -X POST http://localhost:7860/sdapi/v1/txt2img \
  -H "Content-Type: application/json" \
  -d '{"prompt":"a sunset over mountains","steps":20,"width":512,"height":512}'
```

### 2. Windows Snapdragon (Stable Diffusion XL)

**Install Stable Diffusion WebUI** (ARM64 compatible):
```powershell
# Option A: Automatic1111 (if ARM64 build available)
git clone https://github.com/AUTOMATIC1111/stable-diffusion-webui.git
cd stable-diffusion-webui
.\webui-user.bat

# Option B: ComfyUI (lighter, better for ARM64)
git clone https://github.com/comfyanonymous/ComfyUI.git
cd ComfyUI
python -m venv venv
.\venv\Scripts\activate
pip install torch torchvision torchaudio --index-url https://download.pytorch.org/whl/cpu
pip install -r requirements.txt
python main.py
```

**Start EdgeMesh server**:
```powershell
set IMAGE_API_ENDPOINT=http://localhost:7860
set GRPC_ADDR=0.0.0.0:50051
C:\Users\chinmay\server-windows-arm64.exe
```

### 3. Arduino (TinyML — Simplified)

**Boards we support**:
- ESP32-S3 (WiFi + 8MB PSRAM)
- Arduino Portenta H7 (WiFi)
- Arduino Nano RP2040 Connect (WiFi)

**Two approaches**:

#### Option A: Arduino as Worker Node (Full gRPC)
- Cross-compile Go server for Arduino's architecture
- Extremely difficult — Go on Arduino is experimental
- **Not recommended for hackathon**

#### Option B: Arduino as HTTP Endpoint (Lightweight)
- Arduino runs a simple HTTP server: `POST /generate → returns base64 image`
- Mac/Windows coordinator calls Arduino's HTTP endpoint for TinyML inference
- Much simpler, reliable

**Recommended: Option B**

**Arduino sketch** (pseudo-code):
```cpp
#include <WiFi.h>
#include <WebServer.h>
#include "tinyml_model.h" // Your TinyML image gen model

WebServer server(8080);

void handleGenerate() {
  String prompt = server.arg("prompt");
  uint8_t* image = generateTinyImage(prompt); // TinyML inference
  String base64 = base64Encode(image);
  server.send(200, "application/json", "{\"image\":\"" + base64 + "\"}");
}

void setup() {
  WiFi.begin("SSID", "PASSWORD");
  server.on("/generate", HTTP_POST, handleGenerate);
  server.begin();
}
```

**Register Arduino in EdgeMesh**:
```bash
# Manually register since Arduino won't run Go server
go run ./cmd/client --addr localhost:50051 register \
  --id "arduino-esp32" \
  --name "Arduino-ESP32-TinyML" \
  --self-addr "10.206.x.x:8080" \
  --platform "arduino" --arch "arm"
```

**Modify coordinator** to call Arduino's HTTP endpoint for IMAGE_GENERATE tasks targeted at Arduino.

---

## Task Flow

### User Request → Plan
```bash
curl -X POST http://localhost:8080/api/plan \
  -d '{"text":"generate an image of a cat","max_workers":0}'
```

**Plan output**:
```json
{
  "groups": [{
    "index": 0,
    "tasks": [{
      "task_id": "img-1",
      "kind": "IMAGE_GENERATE",
      "input": "generate an image of a cat",
      "target_device_id": "windows-snapdragon-qcw31"
    }]
  }]
}
```

**Rationale**: Routes to Windows Snapdragon (has NPU + GPU)

### Cost Estimation
```bash
curl -X POST http://localhost:8080/api/plan-cost -d '{...plan...}'
```

**Output**:
```json
{
  "total_predicted_ms": 15000,
  "device_costs": [
    {"device_name": "Mac", "total_ms": 30000, "notes": "CPU-only, slower"},
    {"device_name": "Windows-Snapdragon", "total_ms": 15000, "notes": "GPU/NPU accelerated"},
    {"device_name": "Arduino-TinyML", "total_ms": 5000, "notes": "TinyML (lower quality)"}
  ],
  "recommended_device_name": "Windows-Snapdragon"
}
```

### Execution
```bash
curl -X POST http://localhost:8080/api/submit-job \
  -d '{"text":"generate a sunset image","max_workers":0}'
```

**Device executes** → calls Stable Diffusion API → saves image → returns path

---

## Quick Setup (Next 3 Hours)

| Step | Task | Time | Priority |
|---|---|---|---|
| 1 | Install SD WebUI on Mac | 30 min | HIGH |
| 2 | Install SD on Windows (ComfyUI) | 45 min | HIGH |
| 3 | Test IMAGE_GENERATE on Mac | 15 min | HIGH |
| 4 | Test IMAGE_GENERATE on Windows | 15 min | HIGH |
| 5 | Test distributed image gen job | 10 min | HIGH |
| 6 | Arduino HTTP endpoint (if ESP32) | 1 hour | MEDIUM |
| 7 | Update LLM system prompt for image tasks | 10 min | LOW |

---

## Environment Variables

| Variable | Default | Purpose |
|---|---|---|
| `IMAGE_API_ENDPOINT` | `http://127.0.0.1:7860` | Stable Diffusion WebUI address |
| `LLM_ENDPOINT` | `http://127.0.0.1:11434` | Ollama for text tasks |
| `LLM_MODEL` | `qwen3:8b` | Text generation model |

**On each device, set the appropriate endpoint before starting the server.**

---

## Testing

### Test 1: Plan Generation
```bash
curl -X POST http://localhost:8080/api/plan \
  -d '{"text":"generate an image of a cat"}'
# Should return: kind="IMAGE_GENERATE", target=<GPU device>
```

### Test 2: Cost Estimation
```bash
# Create plan with IMAGE_GENERATE task
curl -X POST http://localhost:8080/api/plan-cost -d '{...}'
# Should return: predicted_ms=15000 for GPU, 30000 for CPU
```

### Test 3: Execute
```bash
curl -X POST http://localhost:8080/api/submit-job \
  -d '{"text":"generate a sunset image"}'
# Check result:
curl "http://localhost:8080/api/job?id=<job_id>"
# Output should be: "image saved to: /path/to/shared/image_xxx.png"
```

---

## What You Need

**Arduino info needed**:
1. Board model (ESP32-S3, Portenta, etc)
2. Does it have WiFi built-in?
3. What TinyML model/library are you using?

**Software to install**:
1. Stable Diffusion on Mac (Automatic1111 or ComfyUI)
2. Stable Diffusion on Windows (ComfyUI recommended for ARM64)
3. Arduino IDE + TinyML library (if going the Arduino route)

Once you provide Arduino details, I'll finish the implementation.
