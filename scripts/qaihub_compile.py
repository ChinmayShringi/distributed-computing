#!/usr/bin/env python3
"""Submit a compile job to QAI Hub and optionally wait for completion.

This script uses the qai_hub Python SDK (not the CLI) for richer integration.
It's called by the Go backend (internal/qaihub) for compile operations.

Usage:
    # Submit and get job ID immediately
    python scripts/qaihub_compile.py --model model.onnx --device "Samsung Galaxy S24 (Family)"

    # Submit, wait for completion, and download artifacts
    python scripts/qaihub_compile.py --model model.onnx --device "Samsung Galaxy S24 (Family)" --wait --out ./artifacts/

    # Check status of an existing job
    python scripts/qaihub_compile.py --status --job jXXXXXXX

    # List all recent jobs
    python scripts/qaihub_compile.py --list-jobs --limit 10

Requires: pip install qai-hub (in .venv-qaihub)
"""
import argparse
import json
import os
import sys
import time


def main():
    parser = argparse.ArgumentParser(description="QAI Hub compile helper for EdgeMesh")
    sub = parser.add_subparsers(dest="command")

    # --- compile ---
    compile_p = sub.add_parser("compile", help="Submit a compile job")
    compile_p.add_argument("--model", required=True, help="ONNX path or QAI Hub model ID")
    compile_p.add_argument("--device", default="Samsung Galaxy S24 (Family)", help="Target device name")
    compile_p.add_argument("--options", default="", help="Extra compile options")
    compile_p.add_argument("--wait", action="store_true", help="Wait for job to complete")
    compile_p.add_argument("--out", default="", help="Output dir for artifacts (used with --wait)")

    # --- status ---
    status_p = sub.add_parser("status", help="Check job status")
    status_p.add_argument("--job", required=True, help="Job ID")

    # --- download ---
    download_p = sub.add_parser("download", help="Download job artifacts")
    download_p.add_argument("--job", required=True, help="Job ID")
    download_p.add_argument("--out", required=True, help="Output directory")

    # --- list-jobs ---
    list_p = sub.add_parser("list-jobs", help="List recent jobs")
    list_p.add_argument("--limit", type=int, default=10, help="Max jobs to list")

    # --- list-devices ---
    devices_p = sub.add_parser("list-devices", help="List target devices")
    devices_p.add_argument("--name", default="", help="Filter by name")

    args = parser.parse_args()

    try:
        import qai_hub as hub
    except ImportError:
        _error("qai_hub not installed. Run: pip install qai-hub")
        sys.exit(1)

    if args.command == "compile":
        do_compile(hub, args)
    elif args.command == "status":
        do_status(hub, args)
    elif args.command == "download":
        do_download(hub, args)
    elif args.command == "list-jobs":
        do_list_jobs(hub, args)
    elif args.command == "list-devices":
        do_list_devices(hub, args)
    else:
        parser.print_help()
        sys.exit(1)


def do_compile(hub, args):
    """Submit a compile job."""
    # Resolve device
    devices = hub.get_devices(name=args.device)
    if not devices:
        _output({"ok": False, "error": f"No device found: {args.device}"})
        return
    device = devices[0]

    # Resolve model
    model = args.model
    if not (model.startswith("m") and len(model) > 3):
        # Treat as file path
        if not os.path.exists(model):
            _output({"ok": False, "error": f"Model file not found: {model}"})
            return

    # Submit
    try:
        job = hub.submit_compile_job(
            model=model,
            device=device,
            name=f"edgemesh-{int(time.time())}",
            options=args.options or "",
        )
        job_id = job.job_id if hasattr(job, "job_id") else str(job)
    except Exception as e:
        _output({"ok": False, "error": str(e)})
        return

    result = {
        "ok": True,
        "job_id": job_id,
        "job_url": f"https://aihub.qualcomm.com/jobs/{job_id}",
        "status": "submitted",
        "device": args.device,
    }

    if args.wait:
        # Poll until done
        _log(f"Waiting for job {job_id}...")
        while True:
            st = job.get_status()
            if st.success:
                result["status"] = "success"
                break
            if not st.running:
                result["status"] = f"failed: {st.message}"
                result["ok"] = False
                break
            time.sleep(5)

        if result["ok"] and args.out:
            os.makedirs(args.out, exist_ok=True)
            try:
                target = job.get_target_model()
                if target:
                    out_file = os.path.join(args.out, "model.bin")
                    target.download(out_file)
                    result["artifact_path"] = out_file
            except Exception as e:
                result["download_error"] = str(e)

    _output(result)


def do_status(hub, args):
    """Check job status."""
    try:
        job = hub.get_job(args.job)
        st = job.get_status()
        _output({
            "job_id": args.job,
            "status": st.message,
            "success": st.success,
            "running": st.running if hasattr(st, "running") else False,
        })
    except Exception as e:
        _output({"job_id": args.job, "status": "error", "success": False, "error": str(e)})


def do_download(hub, args):
    """Download artifacts from a completed job."""
    try:
        job = hub.get_job(args.job)
        st = job.get_status()
        if not st.success:
            _output({"ok": False, "error": f"Job not done: {st.message}"})
            return

        os.makedirs(args.out, exist_ok=True)
        target = job.get_target_model()
        if target is None:
            _output({"ok": False, "error": "No target model available"})
            return

        out_file = os.path.join(args.out, "model.bin")
        target.download(out_file)
        _output({"ok": True, "path": out_file, "job_id": args.job})
    except Exception as e:
        _output({"ok": False, "error": str(e)})


def do_list_jobs(hub, args):
    """List recent jobs."""
    try:
        summaries = hub.get_job_summaries(limit=args.limit, offset=0)
        jobs = []
        for s in summaries:
            job_id = s.job_id if hasattr(s, "job_id") else str(s)
            status = str(s.status) if hasattr(s, "status") else "unknown"
            name = s.name if hasattr(s, "name") else ""
            jobs.append({"job_id": job_id, "status": status, "name": name})
        _output({"count": len(jobs), "jobs": jobs})
    except Exception as e:
        _output({"count": 0, "jobs": [], "error": str(e)})


def do_list_devices(hub, args):
    """List target devices."""
    try:
        devices = hub.get_devices(name=args.name)
        result = []
        for d in devices:
            result.append({"name": d.name, "os": d.os})
        _output({"count": len(result), "devices": result})
    except Exception as e:
        _output({"count": 0, "devices": [], "error": str(e)})


def _output(data):
    """Print JSON to stdout."""
    print(json.dumps(data))


def _log(msg):
    """Print to stderr so it doesn't interfere with JSON stdout."""
    print(msg, file=sys.stderr)


def _error(msg):
    """Print error JSON."""
    _output({"ok": False, "error": msg})


if __name__ == "__main__":
    main()
