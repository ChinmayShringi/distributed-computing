# qaihub_download_models.ps1 - Download/export models from qai_hub_models
# Usage: powershell -ExecutionPolicy Bypass -File scripts/windows/qaihub_download_models.ps1
#        powershell -ExecutionPolicy Bypass -File scripts/windows/qaihub_download_models.ps1 -Models "stable_diffusion_v2_1,llama_v3_2_3b_instruct"
#
# This script:
#   1. Creates/activates .venv-qaihub virtual environment
#   2. Installs qai-hub and qai-hub-models packages
#   3. Runs the Python download script
#   4. Artifacts are saved to artifacts/qaihub/models/

param(
    [string]$Models = "stable_diffusion_v2_1,llama_v3_2_3b_instruct",
    [string]$Device = "",
    [string]$TargetRuntime = "precompiled_qnn_onnx",
    [switch]$ListAvailable,
    [switch]$DryRun,
    [int]$Timeout = 1800
)

$ErrorActionPreference = "Stop"

$VenvDir = ".venv-qaihub"
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Split-Path -Parent (Split-Path -Parent $ScriptDir)
$PythonScript = Join-Path $RepoRoot "scripts/qaihub_download_models.py"

Write-Host ""
Write-Host "=== QAI Hub Models Download ===" -ForegroundColor Cyan
Write-Host "  Models:         $Models"
Write-Host "  Target Runtime: $TargetRuntime"
Write-Host "  Timeout:        $Timeout seconds per model"
Write-Host ""

# --- Step 1: Check Python ---
Write-Host "[1/5] Checking Python..." -ForegroundColor Yellow
try {
    $pythonVersion = python --version 2>&1
    Write-Host "  $pythonVersion"
} catch {
    Write-Host "ERROR: Python is not installed or not on PATH." -ForegroundColor Red
    Write-Host "Install Python 3.8+ from https://www.python.org/downloads/" -ForegroundColor Red
    exit 1
}

# --- Step 2: Create or reuse venv ---
Write-Host "[2/5] Setting up virtual environment ($VenvDir)..." -ForegroundColor Yellow
Push-Location $RepoRoot
try {
    if (Test-Path "$VenvDir\Scripts\Activate.ps1") {
        Write-Host "  Venv already exists, reusing."
    } else {
        Write-Host "  Creating new venv..."
        python -m venv $VenvDir
        if ($LASTEXITCODE -ne 0) {
            Write-Host "ERROR: Failed to create venv." -ForegroundColor Red
            exit 1
        }
    }

    # Activate venv
    & "$VenvDir\Scripts\Activate.ps1"
    Write-Host "  Venv activated."
} finally {
    Pop-Location
}

# --- Step 3: Install dependencies ---
Write-Host "[3/5] Installing dependencies..." -ForegroundColor Yellow
Push-Location $RepoRoot

# Upgrade pip first
Write-Host "  Upgrading pip..."
python -m pip install --upgrade pip --quiet 2>&1 | Out-Null

# Install qai-hub and qai-hub-models
Write-Host "  Installing qai-hub..."
pip install qai-hub --quiet 2>&1 | Out-Null

Write-Host "  Installing qai-hub-models (this may take a while)..."
pip install qai-hub-models --quiet 2>&1 | Out-Null

# Verify installation
$qaihubVersion = qai-hub --version 2>&1
Write-Host "  Installed: qai-hub $qaihubVersion"

Pop-Location

# --- Step 4: Run Python script ---
Write-Host "[4/5] Running model download script..." -ForegroundColor Yellow

# Set UTF-8 encoding for proper output
$env:PYTHONIOENCODING = "utf-8"

# Build arguments
$pythonArgs = @($PythonScript)

if ($ListAvailable) {
    $pythonArgs += "--list-available"
} else {
    $pythonArgs += @("--models", $Models)
    $pythonArgs += @("--target-runtime", $TargetRuntime)
    $pythonArgs += @("--timeout", $Timeout)

    if ($Device) {
        $pythonArgs += @("--device", $Device)
    }

    if ($DryRun) {
        $pythonArgs += "--dry-run"
    }
}

# Run Python script
Push-Location $RepoRoot
$prevPref = $ErrorActionPreference
$ErrorActionPreference = "Continue"

python @pythonArgs
$exitCode = $LASTEXITCODE

$ErrorActionPreference = $prevPref
Pop-Location

# --- Step 5: Done ---
Write-Host ""
if ($exitCode -eq 0) {
    Write-Host "[5/5] Download complete!" -ForegroundColor Green
    $artifactsPath = Join-Path $RepoRoot "artifacts/qaihub/models"
    Write-Host "  Artifacts saved to: $artifactsPath" -ForegroundColor Green
} else {
    Write-Host "[5/5] Download finished with errors (exit code $exitCode)" -ForegroundColor Yellow
}
Write-Host ""

exit $exitCode
