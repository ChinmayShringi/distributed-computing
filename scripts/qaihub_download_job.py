#!/usr/bin/env python3
"""Download compiled artifacts from a QAI Hub job.

Usage:
    python scripts/qaihub_download_job.py --job jXXXXXXXX --out ./artifacts/qaihub/jXXXXXXXX/

Requires qai_hub package (pip install qai-hub) and a configured API token.
"""
import argparse
import os
import sys


def main():
    parser = argparse.ArgumentParser(description="Download QAI Hub job artifacts")
    parser.add_argument("--job", required=True, help="Job ID (e.g., jXXXXXXXX)")
    parser.add_argument("--out", required=True, help="Output directory")
    args = parser.parse_args()

    try:
        import qai_hub as hub
    except ImportError:
        print(
            "ERROR: qai_hub not installed. Install with: pip install qai-hub",
            file=sys.stderr,
        )
        sys.exit(1)

    print(f"Fetching job {args.job}...")
    try:
        job = hub.get_job(args.job)
    except Exception as e:
        print(f"ERROR: Failed to fetch job {args.job}: {e}", file=sys.stderr)
        sys.exit(1)

    status = job.get_status()
    print(f"Job status: {status.message}")

    if not status.success:
        print(f"ERROR: Job did not succeed (status: {status.message})", file=sys.stderr)
        sys.exit(1)

    os.makedirs(args.out, exist_ok=True)
    print(f"Downloading artifacts to {args.out}...")

    # For compile jobs, download the compiled target model
    target_model = job.get_target_model()
    if target_model is None:
        print("ERROR: No target model available (job may have failed).", file=sys.stderr)
        sys.exit(1)

    out_file = os.path.join(args.out, "model.bin")
    target_model.download(out_file)
    print(f"Compiled model saved to: {out_file}")


if __name__ == "__main__":
    main()
