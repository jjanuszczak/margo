#!/usr/bin/env python3

from __future__ import annotations

import json
import subprocess
import sys
from datetime import datetime, timezone
from pathlib import Path


def run_check(name: str, command: list[str], cwd: Path) -> dict[str, object]:
    result = subprocess.run(
        command,
        cwd=str(cwd),
        text=True,
        capture_output=True,
    )
    return {
        "name": name,
        "status": "PASS" if result.returncode == 0 else "FAIL",
        "command": command,
        "stdout": result.stdout,
        "stderr": result.stderr,
        "returncode": result.returncode,
    }


def main() -> int:
    repo_root = Path(__file__).resolve().parents[3]
    skill_root = Path(__file__).resolve().parents[1]
    version = "0.0.0-eval"

    checks = [
        run_check(
            "prepare_release",
            [
                sys.executable,
                str(skill_root / "scripts" / "prepare_release.py"),
                version,
                "--allow-dirty",
            ],
            repo_root,
        ),
        run_check(
            "verify_release",
            [
                sys.executable,
                str(skill_root / "scripts" / "verify_release.py"),
                version,
                "--skip-gh-release-check",
                "--skip-brew",
            ],
            repo_root,
        ),
    ]

    overall_status = "PASS" if all(check["status"] == "PASS" for check in checks) else "FAIL"
    payload = {
        "timestamp": datetime.now(timezone.utc).isoformat(),
        "overall_status": overall_status,
        "checks": checks,
    }

    report_dir = skill_root / "evals" / "reports"
    report_dir.mkdir(parents=True, exist_ok=True)
    report_path = report_dir / "latest_results.json"
    report_path.write_text(json.dumps(payload, indent=2) + "\n", encoding="utf-8")

    print(f"Evaluation complete. Status: {overall_status}")
    print(f"Report saved to: {report_path}")
    return 0 if overall_status == "PASS" else 1


if __name__ == "__main__":
    raise SystemExit(main())
