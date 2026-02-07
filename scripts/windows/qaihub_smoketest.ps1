# qaihub_smoketest.ps1 — Verify qai-hub CLI is working
# Usage: powershell -ExecutionPolicy Bypass -File scripts/windows/qaihub_smoketest.ps1

$ErrorActionPreference = "Stop"

$VenvDir = ".venv-qaihub"

Write-Host ""
Write-Host "=== Qualcomm AI Hub Smoke Test ===" -ForegroundColor Cyan
Write-Host ""

# Activate venv
if (-not (Test-Path "$VenvDir\Scripts\Activate.ps1")) {
    Write-Host "ERROR: Venv not found at $VenvDir." -ForegroundColor Red
    Write-Host "Run setup first: powershell -ExecutionPolicy Bypass -File scripts/windows/setup_qaihub.ps1" -ForegroundColor Yellow
    exit 1
}

& "$VenvDir\Scripts\Activate.ps1"

# Run list-devices and capture output
Write-Host "Running qai-hub list-devices..." -ForegroundColor Yellow
$output = qai-hub list-devices 2>&1 | Out-String

if ($LASTEXITCODE -ne 0) {
    Write-Host "FAIL: qai-hub list-devices exited with code $LASTEXITCODE" -ForegroundColor Red
    Write-Host $output
    exit 1
}

if ([string]::IsNullOrWhiteSpace($output)) {
    Write-Host "FAIL: qai-hub list-devices returned empty output." -ForegroundColor Red
    exit 1
}

Write-Host $output
Write-Host "PASS: qai-hub list-devices succeeded." -ForegroundColor Green
Write-Host ""
