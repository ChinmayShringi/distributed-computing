# setup_qaihub.ps1 — Idempotent setup for Qualcomm AI Hub CLI
# Usage: powershell -ExecutionPolicy Bypass -File scripts/windows/setup_qaihub.ps1
# Requires: Python 3 on PATH, QAI_HUB_API_TOKEN environment variable

$ErrorActionPreference = "Stop"

$VenvDir = ".venv-qaihub"

Write-Host ""
Write-Host "=== Qualcomm AI Hub CLI Setup ===" -ForegroundColor Cyan
Write-Host ""

# --- Step 1: Check Python ---
Write-Host "[1/6] Checking Python..." -ForegroundColor Yellow
try {
    python --version
} catch {
    Write-Host "ERROR: Python is not installed or not on PATH." -ForegroundColor Red
    Write-Host "Install Python 3.8+ from https://www.python.org/downloads/" -ForegroundColor Red
    exit 1
}

# --- Step 2: Create or reuse venv ---
Write-Host "[2/6] Setting up virtual environment ($VenvDir)..." -ForegroundColor Yellow
if (Test-Path "$VenvDir\Scripts\Activate.ps1") {
    Write-Host "  Venv already exists, reusing."
} else {
    Write-Host "  Creating new venv..."
    python -m venv $VenvDir
}

# Activate venv
& "$VenvDir\Scripts\Activate.ps1"

# --- Step 3: Install/upgrade pip and qai-hub ---
Write-Host "[3/6] Installing qai-hub..." -ForegroundColor Yellow
python -m pip install --upgrade pip --quiet
pip install qai-hub --quiet
Write-Host "  qai-hub installed."

# --- Step 4: Verify qai-hub is available ---
Write-Host "[4/6] Verifying qai-hub..." -ForegroundColor Yellow
qai-hub --help | Select-Object -First 3

# --- Step 5: Configure API token ---
Write-Host "[5/6] Configuring API token..." -ForegroundColor Yellow
$DefaultToken = "b3yhelucambdm13uz9usknrhu98l6pln1dzboooy"
$Token = if ($env:QAI_HUB_API_TOKEN) { $env:QAI_HUB_API_TOKEN } else { $DefaultToken }

qai-hub configure --api_token $Token
Write-Host "  API token configured."

# --- Step 6: List devices ---
Write-Host "[6/6] Running qai-hub list-devices..." -ForegroundColor Yellow
qai-hub list-devices

Write-Host ""
Write-Host "=== Setup complete ===" -ForegroundColor Green
Write-Host ""
