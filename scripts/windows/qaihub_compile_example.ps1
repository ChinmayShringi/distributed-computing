# qaihub_compile_example.ps1 — Compile a model via Qualcomm AI Hub
# Usage: powershell -ExecutionPolicy Bypass -File scripts/windows/qaihub_compile_example.ps1 [-Device "Samsung Galaxy S24 (Family)"] [-TargetRuntime "precompiled_qnn_onnx"]

param(
    [string]$Device = "Samsung Galaxy S24 (Family)",
    [string]$TargetRuntime = "precompiled_qnn_onnx"
)

$ErrorActionPreference = "Stop"

$VenvDir = ".venv-qaihub"
$ModelDir = "models"
$ModelFile = "$ModelDir\mobilenet_v2.onnx"
$ModelUrl = "https://github.com/onnx/models/raw/main/validated/vision/classification/mobilenet/model/mobilenetv2-12.onnx"
$ArtifactsDir = "artifacts"

Write-Host ""
Write-Host "=== QAI Hub Compile Example ===" -ForegroundColor Cyan
Write-Host "  Device:         $Device"
Write-Host "  Target Runtime: $TargetRuntime"
Write-Host ""

# --- Step 1: Check API token ---
Write-Host "[1/7] Checking API token..." -ForegroundColor Yellow
if (-not $env:QAI_HUB_API_TOKEN) {
    Write-Host "ERROR: QAI_HUB_API_TOKEN environment variable is not set." -ForegroundColor Red
    Write-Host ""
    Write-Host "Set it before running this script:" -ForegroundColor Yellow
    Write-Host '  $env:QAI_HUB_API_TOKEN = "your_token_here"' -ForegroundColor White
    Write-Host "  Or in CMD: set QAI_HUB_API_TOKEN=your_token_here" -ForegroundColor White
    Write-Host ""
    Write-Host "Get your token at https://aihub.qualcomm.com/" -ForegroundColor Yellow
    exit 1
}
Write-Host "  Token is set."

# --- Step 2: Ensure venv ---
Write-Host "[2/7] Ensuring venv..." -ForegroundColor Yellow
if (-not (Test-Path "$VenvDir\Scripts\Activate.ps1")) {
    Write-Host "  Venv not found. Running setup script..."
    & "$PSScriptRoot\setup_qaihub.ps1"
}
& "$VenvDir\Scripts\Activate.ps1"
Write-Host "  Venv activated."

# --- Step 3: Download model ---
Write-Host "[3/7] Checking model..." -ForegroundColor Yellow
if (-not (Test-Path $ModelDir)) {
    New-Item -ItemType Directory -Path $ModelDir | Out-Null
}
if (-not (Test-Path $ModelFile)) {
    Write-Host "  Downloading mobilenet_v2.onnx..."
    Invoke-WebRequest -Uri $ModelUrl -OutFile $ModelFile -UseBasicParsing
    Write-Host "  Downloaded to $ModelFile"
} else {
    Write-Host "  Model already exists at $ModelFile"
}

# --- Step 4: Submit compile job ---
Write-Host "[4/7] Submitting compile job (this may take several minutes)..." -ForegroundColor Yellow
# Force UTF-8 output to avoid Unicode errors with emoji in qai-hub progress display
$env:PYTHONIOENCODING = "utf-8"
$prevPref = $ErrorActionPreference
$ErrorActionPreference = "Continue"
$output = qai-hub submit-compile-job `
    --model $ModelFile `
    --device $Device `
    --input_specs "dict(input=(1,3,224,224))" `
    --compile_options "--target_runtime $TargetRuntime" `
    --wait 2>&1 | Out-String
$compileExit = $LASTEXITCODE
$ErrorActionPreference = $prevPref

Write-Host $output

if ($compileExit -ne 0) {
    Write-Host "ERROR: Compile job failed (exit code $compileExit)." -ForegroundColor Red
    exit 1
}

# --- Step 5: Parse job ID ---
Write-Host "[5/7] Parsing job ID..." -ForegroundColor Yellow
# Job IDs look like j5mzw1w7p (j + alphanumeric, 6+ chars) — avoid matching plain "job"
$jobMatch = [regex]::Match($output, '\b(j[a-z0-9]{5,})\b')
if (-not $jobMatch.Success) {
    Write-Host "ERROR: Could not parse job ID from output." -ForegroundColor Red
    Write-Host "Output was:" -ForegroundColor Red
    Write-Host $output
    exit 1
}
$JobId = $jobMatch.Groups[1].Value
Write-Host "  Job ID: $JobId"

# Save job ID
if (-not (Test-Path $ArtifactsDir)) {
    New-Item -ItemType Directory -Path $ArtifactsDir | Out-Null
}
$JobId | Out-File -FilePath "$ArtifactsDir\last_qaihub_job.txt" -Encoding utf8 -NoNewline
Write-Host "  Saved to $ArtifactsDir\last_qaihub_job.txt"

# --- Step 6: Download artifacts ---
Write-Host "[6/7] Downloading artifacts..." -ForegroundColor Yellow
$OutDir = "$ArtifactsDir\qaihub\$JobId"
python scripts/qaihub_download_job.py --job $JobId --out $OutDir

# --- Step 7: Done ---
Write-Host ""
Write-Host "[7/7] Done!" -ForegroundColor Green
Write-Host "  Artifacts saved to: $OutDir" -ForegroundColor Green
Write-Host ""
