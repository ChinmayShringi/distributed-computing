#!/usr/bin/env python3
"""
qaihub_download_models.py - Download/export models from qai_hub_models

This script exports pre-built AI models from the qai_hub_models package
for deployment on Qualcomm Snapdragon devices.

Usage:
    python scripts/qaihub_download_models.py
    python scripts/qaihub_download_models.py --list-available
    python scripts/qaihub_download_models.py --models stable_diffusion_v2_1,llama_v3_2_3b_instruct
    python scripts/qaihub_download_models.py --dry-run
"""

import argparse
import os
import pkgutil
import re
import shutil
import subprocess
import sys
import tempfile
import time
from dataclasses import dataclass
from pathlib import Path
from typing import List, Optional, Tuple

# --- ANSI Colors ---
RED = '\033[0;31m'
GREEN = '\033[0;32m'
YELLOW = '\033[1;33m'
CYAN = '\033[0;36m'
BOLD = '\033[1m'
NC = '\033[0m'  # No Color

# Disable colors if not a TTY
if not sys.stdout.isatty():
    RED = GREEN = YELLOW = CYAN = BOLD = NC = ''

# --- Default Models ---
DEFAULT_MODELS = [
    "stable_diffusion_v2_1",
    "llama_v3_2_3b_instruct",
]

# --- Model Name Mapping ---
# Maps user-friendly names to qai_hub_models module names
MODEL_ALIASES = {
    # Stable Diffusion variants
    "stable-diffusion-v2.1": "stable_diffusion_v2_1",
    "stable_diffusion_v2.1": "stable_diffusion_v2_1",
    "sd-v2.1": "stable_diffusion_v2_1",
    "stable-diffusion-v1.5": "stable_diffusion_v1_5",
    "stable_diffusion_v1.5": "stable_diffusion_v1_5",
    "sd-v1.5": "stable_diffusion_v1_5",
    # Llama variants
    "llama-v3.2-3b-instruct": "llama_v3_2_3b_instruct",
    "llama-3.2-3b": "llama_v3_2_3b_instruct",
    "llama3.2-3b": "llama_v3_2_3b_instruct",
    # ControlNet
    "controlnet-canny": "controlnet_canny",
    "controlnet": "controlnet_canny",
}


@dataclass
class ExportResult:
    """Result of a model export attempt."""
    success: bool
    requested_name: str
    resolved_name: str
    device: str
    output_path: str
    duration_seconds: float
    error_msg: str = ""


def _step(num: int, total: int, msg: str):
    """Print step header."""
    print(f"\n{YELLOW}[{num}/{total}] {msg}...{NC}")


def _log(msg: str):
    """Print info message."""
    print(f"  {msg}")


def _success(msg: str):
    """Print success message."""
    print(f"{GREEN}  {msg}{NC}")


def _error(msg: str):
    """Print error message to stderr."""
    print(f"{RED}ERROR: {msg}{NC}", file=sys.stderr)


def _warn(msg: str):
    """Print warning message."""
    print(f"{YELLOW}WARN: {msg}{NC}")


def validate_qaihub_configured() -> bool:
    """
    Validate that qai-hub CLI is installed and configured.
    Returns True if valid, False otherwise.
    """
    try:
        result = subprocess.run(
            ["qai-hub", "list-devices"],
            capture_output=True,
            text=True,
            timeout=30
        )
        if result.returncode != 0:
            _error("qai-hub list-devices failed")
            _log(f"stderr: {result.stderr.strip()}")
            return False
        if not result.stdout.strip():
            _error("qai-hub list-devices returned empty output")
            return False
        _success("qai-hub is configured")
        return True
    except FileNotFoundError:
        _error("qai-hub not found. Install with: pip install qai-hub")
        return False
    except subprocess.TimeoutExpired:
        _error("qai-hub list-devices timed out")
        return False
    except Exception as e:
        _error(f"Failed to run qai-hub: {e}")
        return False


def auto_select_device() -> Tuple[str, str]:
    """
    Auto-select the best target device for export.
    Returns (device_name, selection_reason).

    Priority:
    1. Snapdragon X Elite + Windows
    2. Any Snapdragon X Elite
    3. Any Windows device with Snapdragon
    4. First device in list
    """
    try:
        result = subprocess.run(
            ["qai-hub", "list-devices"],
            capture_output=True,
            text=True,
            timeout=30
        )
        if result.returncode != 0:
            return ("", "qai-hub list-devices failed")

        lines = result.stdout.strip().split('\n')
        devices = []

        # Parse device table - skip header lines
        for line in lines:
            line = line.strip()
            if not line or line.startswith('-') or line.startswith('Device'):
                continue
            # Device names are typically the first column
            parts = line.split()
            if parts:
                device_name = parts[0]
                # Reconstruct full device name if it has spaces (check for common patterns)
                if "Snapdragon" in line or "Galaxy" in line or "Pixel" in line:
                    # Try to extract device name more carefully
                    # Look for pattern like "Snapdragon X Elite CRD"
                    match = re.search(r'(Snapdragon[^\|]+|Samsung[^\|]+|Google[^\|]+)', line)
                    if match:
                        device_name = match.group(1).strip()
                devices.append((device_name, line))

        if not devices:
            return ("", "No devices found")

        # Priority 1: Snapdragon X Elite + Windows
        for device_name, line in devices:
            if "Snapdragon X Elite" in line and "Windows" in line.lower():
                return (device_name, "Snapdragon X Elite + Windows (preferred)")

        # Priority 2: Any Snapdragon X Elite
        for device_name, line in devices:
            if "Snapdragon X Elite" in line:
                return (device_name, "Snapdragon X Elite")

        # Priority 3: Any Windows + Snapdragon
        for device_name, line in devices:
            if "Snapdragon" in line and "Windows" in line.lower():
                return (device_name, "Windows Snapdragon device")

        # Priority 4: First device
        device_name, line = devices[0]
        return (device_name, f"First available device (no Snapdragon X Elite found)")

    except Exception as e:
        return ("", f"Error selecting device: {e}")


def discover_available_models() -> List[str]:
    """
    Discover all available model names in qai_hub_models.models.
    Returns list of model module names.
    """
    models = []
    try:
        import qai_hub_models.models as models_pkg
        for importer, modname, ispkg in pkgutil.iter_modules(models_pkg.__path__):
            if ispkg:
                # Check if this module has an export submodule
                try:
                    model_path = os.path.join(models_pkg.__path__[0], modname)
                    if os.path.exists(os.path.join(model_path, "export.py")):
                        models.append(modname)
                    elif os.path.exists(os.path.join(model_path, "__init__.py")):
                        # Some models might have export in __init__
                        models.append(modname)
                except Exception:
                    pass
        return sorted(models)
    except ImportError:
        _error("qai_hub_models not installed. Install with: pip install qai-hub-models")
        return []
    except Exception as e:
        _error(f"Error discovering models: {e}")
        return []


def resolve_model_name(requested: str, available: List[str]) -> Optional[str]:
    """
    Resolve a user-friendly model name to an available qai_hub_models name.
    Returns the resolved name or None if not found.
    """
    # Normalize: lowercase, replace hyphens/dots with underscores
    normalized = requested.lower().replace("-", "_").replace(".", "_")

    # Check direct match
    if normalized in available:
        return normalized

    # Check alias mapping
    alias_key = requested.lower().replace("_", "-")
    if alias_key in MODEL_ALIASES:
        resolved = MODEL_ALIASES[alias_key]
        if resolved in available:
            return resolved

    # Check if normalized form exists in aliases
    if normalized in MODEL_ALIASES:
        resolved = MODEL_ALIASES[normalized]
        if resolved in available:
            return resolved

    # Fuzzy match: find closest match
    for model in available:
        if normalized in model or model in normalized:
            return model

    return None


def detect_output_arg_style(model_name: str) -> str:
    """
    Detect whether the model's export command uses --output-dir or --output_dir.
    Returns the correct argument name.
    """
    try:
        result = subprocess.run(
            ["python", "-m", f"qai_hub_models.models.{model_name}.export", "--help"],
            capture_output=True,
            text=True,
            timeout=30
        )
        help_text = result.stdout + result.stderr

        if "--output-dir" in help_text:
            return "--output-dir"
        elif "--output_dir" in help_text:
            return "--output_dir"
        else:
            # Default to --output-dir
            return "--output-dir"
    except Exception:
        return "--output-dir"


def export_model(
    model_name: str,
    device: str,
    output_base: str,
    target_runtime: str,
    timeout_seconds: int
) -> ExportResult:
    """
    Export a single model using qai_hub_models.
    Returns ExportResult with status and details.
    """
    start_time = time.time()
    output_dir = os.path.join(output_base, model_name)

    try:
        # Create output directory
        os.makedirs(output_dir, exist_ok=True)

        # Detect output arg style
        output_arg = detect_output_arg_style(model_name)

        # Build export command
        cmd = [
            "python", "-m", f"qai_hub_models.models.{model_name}.export",
            "--device", device,
            "--target-runtime", target_runtime,
        ]

        # Try with output-dir first
        cmd_with_output = cmd + [output_arg, output_dir]

        _log(f"Running: {' '.join(cmd_with_output[:6])}...")

        result = subprocess.run(
            cmd_with_output,
            capture_output=True,
            text=True,
            timeout=timeout_seconds,
            cwd=os.getcwd()
        )

        duration = time.time() - start_time

        if result.returncode == 0:
            # Check if output directory has files
            if os.path.exists(output_dir) and os.listdir(output_dir):
                return ExportResult(
                    success=True,
                    requested_name=model_name,
                    resolved_name=model_name,
                    device=device,
                    output_path=output_dir,
                    duration_seconds=duration
                )
            else:
                # Export succeeded but no files - try to find where it put them
                # Look for common output patterns
                possible_dirs = [
                    os.path.join(os.getcwd(), "build"),
                    os.path.join(os.getcwd(), model_name),
                    os.path.join(os.getcwd(), f"{model_name}_export"),
                ]
                for pdir in possible_dirs:
                    if os.path.exists(pdir) and os.listdir(pdir):
                        # Copy to our output dir
                        shutil.copytree(pdir, output_dir, dirs_exist_ok=True)
                        return ExportResult(
                            success=True,
                            requested_name=model_name,
                            resolved_name=model_name,
                            device=device,
                            output_path=output_dir,
                            duration_seconds=duration
                        )

                return ExportResult(
                    success=True,
                    requested_name=model_name,
                    resolved_name=model_name,
                    device=device,
                    output_path=output_dir,
                    duration_seconds=duration,
                    error_msg="Export succeeded but output location unclear"
                )
        else:
            error_msg = result.stderr.strip() or result.stdout.strip()
            # Truncate long errors
            if len(error_msg) > 500:
                error_msg = error_msg[:500] + "..."
            return ExportResult(
                success=False,
                requested_name=model_name,
                resolved_name=model_name,
                device=device,
                output_path="",
                duration_seconds=duration,
                error_msg=error_msg
            )

    except subprocess.TimeoutExpired:
        duration = time.time() - start_time
        return ExportResult(
            success=False,
            requested_name=model_name,
            resolved_name=model_name,
            device=device,
            output_path="",
            duration_seconds=duration,
            error_msg=f"Timeout after {timeout_seconds}s"
        )
    except Exception as e:
        duration = time.time() - start_time
        return ExportResult(
            success=False,
            requested_name=model_name,
            resolved_name=model_name,
            device=device,
            output_path="",
            duration_seconds=duration,
            error_msg=str(e)
        )


def format_duration(seconds: float) -> str:
    """Format duration in human-readable format."""
    if seconds < 60:
        return f"{seconds:.1f}s"
    minutes = int(seconds // 60)
    secs = int(seconds % 60)
    return f"{minutes}m {secs}s"


def print_summary_table(results: List[ExportResult]):
    """Print a formatted summary table of results."""
    print()
    print(f"{BOLD}{'Model':<30} {'Status':<10} {'Duration':<12} {'Output Path'}{NC}")
    print("-" * 80)

    for r in results:
        status = f"{GREEN}SUCCESS{NC}" if r.success else f"{RED}FAILED{NC}"
        duration = format_duration(r.duration_seconds)
        output = r.output_path if r.success else (r.error_msg[:40] + "..." if len(r.error_msg) > 40 else r.error_msg)

        # Reset color codes for alignment
        status_plain = "SUCCESS" if r.success else "FAILED"
        print(f"{r.resolved_name:<30} {status:<20} {duration:<12} {output}")

    print("-" * 80)
    successes = sum(1 for r in results if r.success)
    print(f"Total: {successes}/{len(results)} succeeded")


def parse_args() -> argparse.Namespace:
    """Parse command line arguments."""
    parser = argparse.ArgumentParser(
        description="Download/export models from qai_hub_models for Snapdragon deployment",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog="""
Examples:
  python scripts/qaihub_download_models.py
  python scripts/qaihub_download_models.py --list-available
  python scripts/qaihub_download_models.py --models stable_diffusion_v2_1,llama_v3_2_3b_instruct
  python scripts/qaihub_download_models.py --dry-run
"""
    )
    parser.add_argument(
        "--models", "-m",
        type=str,
        default=",".join(DEFAULT_MODELS),
        help=f"Comma-separated list of models to export (default: {','.join(DEFAULT_MODELS)})"
    )
    parser.add_argument(
        "--device", "-d",
        type=str,
        default="",
        help="Target device name (auto-selects if empty)"
    )
    parser.add_argument(
        "--target-runtime", "-r",
        type=str,
        default="precompiled_qnn_onnx",
        help="Target runtime for export (default: precompiled_qnn_onnx)"
    )
    parser.add_argument(
        "--output-base", "-o",
        type=str,
        default="artifacts/qaihub/models",
        help="Base output directory (default: artifacts/qaihub/models)"
    )
    parser.add_argument(
        "--list-available", "-l",
        action="store_true",
        help="List available models and exit"
    )
    parser.add_argument(
        "--dry-run", "-n",
        action="store_true",
        help="Show what would be done without executing"
    )
    parser.add_argument(
        "--timeout", "-t",
        type=int,
        default=1800,
        help="Timeout per model in seconds (default: 1800 = 30 minutes)"
    )
    return parser.parse_args()


def main() -> int:
    """Main entry point."""
    args = parse_args()

    print(f"\n{CYAN}=== QAI Hub Models Download ==={NC}")
    print(f"  Models: {args.models}")
    print(f"  Target Runtime: {args.target_runtime}")

    total_steps = 5

    # --- Step 1: Validate qai-hub ---
    _step(1, total_steps, "Validating qai-hub configuration")
    if not validate_qaihub_configured():
        _error("qai-hub is not configured. Run setup script first:")
        _log("  powershell -ExecutionPolicy Bypass -File scripts/windows/setup_qaihub.ps1")
        return 1

    # --- Step 2: Select device ---
    _step(2, total_steps, "Selecting target device")
    if args.device:
        device = args.device
        _log(f"Using specified device: {device}")
    else:
        device, reason = auto_select_device()
        if not device:
            _error(f"Could not select device: {reason}")
            return 1
        _log(f"Auto-selected: {device}")
        _log(f"Reason: {reason}")

    # --- Step 3: Discover models ---
    _step(3, total_steps, "Discovering available models")
    available = discover_available_models()
    if not available:
        _error("No models found in qai_hub_models")
        _log("Install with: pip install qai-hub-models")
        return 1
    _success(f"Found {len(available)} models in qai_hub_models")

    if args.list_available:
        print(f"\n{BOLD}Available models:{NC}")
        for m in available:
            print(f"  {m}")
        return 0

    # --- Step 4: Export models ---
    _step(4, total_steps, "Exporting models")

    requested_models = [m.strip() for m in args.models.split(",") if m.strip()]
    results: List[ExportResult] = []

    for i, requested in enumerate(requested_models, 1):
        resolved = resolve_model_name(requested, available)

        if not resolved:
            _warn(f"[{i}/{len(requested_models)}] Model not found: {requested}")
            results.append(ExportResult(
                success=False,
                requested_name=requested,
                resolved_name=requested,
                device=device,
                output_path="",
                duration_seconds=0,
                error_msg="Model not found in qai_hub_models"
            ))
            continue

        if args.dry_run:
            _log(f"[{i}/{len(requested_models)}] Would export: {resolved}")
            results.append(ExportResult(
                success=True,
                requested_name=requested,
                resolved_name=resolved,
                device=device,
                output_path=f"{args.output_base}/{resolved}",
                duration_seconds=0,
                error_msg="(dry-run)"
            ))
            continue

        _log(f"[{i}/{len(requested_models)}] Exporting: {resolved}")
        result = export_model(
            model_name=resolved,
            device=device,
            output_base=args.output_base,
            target_runtime=args.target_runtime,
            timeout_seconds=args.timeout
        )
        result.requested_name = requested
        results.append(result)

        if result.success:
            _success(f"  Exported to: {result.output_path} ({format_duration(result.duration_seconds)})")
        else:
            _error(f"  Failed: {result.error_msg}")

    # --- Step 5: Summary ---
    _step(5, total_steps, "Summary")
    print_summary_table(results)

    # Exit code: 0 if at least one success
    successes = sum(1 for r in results if r.success)
    if successes == 0:
        _error("All exports failed")
        return 1

    _success(f"\nArtifacts saved to: {args.output_base}/")
    return 0


if __name__ == "__main__":
    sys.exit(main())
