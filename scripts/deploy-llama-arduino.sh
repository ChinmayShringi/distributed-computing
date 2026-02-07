#!/bin/bash
# Deploy llama.cpp and Qwen-0.5B to Arduino UNO Q (Dragonwing)
# Prerequisites: Arduino on network, SSH access as arduino@<IP>

set -e
ARDUINO_IP="${ARDUINO_IP:-10.206.56.57}"
ARDUINO_USER="${ARDUINO_USER:-arduino}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Download URLs
LLAMA_URL="https://github.com/ggerganov/llama.cpp/releases/download/b7957/llama-b7957-bin-310p-openEuler-aarch64.tar.gz"
MODEL_URL="https://huggingface.co/Qwen/Qwen2.5-0.5B-Instruct-GGUF/resolve/main/qwen2.5-0.5b-instruct-q4_k_m.gguf"

# Local temp dir
mkdir -p dist/arduino-deploy
cd dist/arduino-deploy

echo "=== Preparing Deployment for Dragonwing ($ARDUINO_IP) ==="

# 1. Download llama.cpp binary (if not exists)
if [ ! -f "llama-bin.tar.gz" ]; then
    echo "Downloading llama.cpp binary..."
    curl -L -o llama-bin.tar.gz "$LLAMA_URL"
fi

# 2. Extract and prepare binary
if [ ! -f "llama-server" ]; then
    echo "Extracting llama-server..."
    tar -xzf llama-bin.tar.gz
    # The tar might contain a folder or direct binaries. We need 'llama-server'.
    # Adjust based on tarball structure. Assuming standard structure:
    if [ -f "bin/llama-server" ]; then
        cp bin/llama-server .
    elif [ -f "llama-server" ]; then
        : # It's here
    else
        # Try finding it
        find . -name "llama-server" -exec cp {} . \;
    fi
fi

# 3. Download Model (if not exists)
if [ ! -f "qwen2.5-0.5b-instruct-q4_k_m.gguf" ]; then
    echo "Downloading Qwen-0.5B model..."
    curl -L -o qwen2.5-0.5b-instruct-q4_k_m.gguf "$MODEL_URL"
fi

# 4. Build EdgeMesh server for Linux ARM64
echo "Building EdgeMesh Server..."
cd "$PROJECT_ROOT"
GOOS=linux GOARCH=arm64 go build -o dist/arduino-deploy/edgemesh-server-linux-arm64 ./cmd/server
cd dist/arduino-deploy

# 5. Push to Device
echo "Copying files to device (this may take a minute)..."
# Create remote directory
ssh "$ARDUINO_USER@$ARDUINO_IP" "mkdir -p /tmp/edgemesh"

# SCP files
scp llama-server \
    qwen2.5-0.5b-instruct-q4_k_m.gguf \
    edgemesh-server-linux-arm64 \
    "$ARDUINO_USER@$ARDUINO_IP:/tmp/edgemesh/"

# 6. Start Services
echo "Starting services on device..."

# Generate startup script on device
ssh "$ARDUINO_USER@$ARDUINO_IP" "cat > /tmp/edgemesh/start.sh << 'EOF'
#!/bin/bash
cd /tmp/edgemesh
chmod +x llama-server edgemesh-server-linux-arm64

# Start llama-server
echo 'Starting llama-server...'
# Port 8080, NGL 99 (use GPU/NPU if driver allows, else CPU fallback), 4 threads
nohup ./llama-server -m qwen2.5-0.5b-instruct-q4_k_m.gguf --port 8080 --host 127.0.0.1 -c 2048 > llama.log 2>&1 &
LLAMA_PID=\$!

# Wait for llama-server to be ready
echo 'Waiting for llama-server...'
for i in {1..30}; do
    if curl -s http://127.0.0.1:8080/health > /dev/null; then
        echo 'llama-server is up!'
        break
    fi
    sleep 1
done

# Start EdgeMesh Server
echo 'Starting EdgeMesh Server...'
export GRPC_ADDR=0.0.0.0:50051
export LLM_ENDPOINT=http://127.0.0.1:8080
export LLM_MODEL=qwen2.5-0.5b-instruct-q4_k_m.gguf
export DEVICE_ID=dragonwing-arduino
export SHARED_DIR=/tmp/edgemesh/shared
mkdir -p \$SHARED_DIR

nohup ./edgemesh-server-linux-arm64 > edgemesh.log 2>&1 &
MESH_PID=\$!

echo \"Started. PIDs: llama=\$LLAMA_PID, mesh=\$MESH_PID\"
EOF
"

# Execute startup script
ssh "$ARDUINO_USER@$ARDUINO_IP" "bash /tmp/edgemesh/start.sh"

echo "=== Deployment Complete ==="
echo "Register the device from your Mac:"
echo "go run ./cmd/client register --id dragonwing-arduino --name DragonwingAI --self-addr $ARDUINO_IP:50051 --platform linux --arch arm64 --has-npu"
