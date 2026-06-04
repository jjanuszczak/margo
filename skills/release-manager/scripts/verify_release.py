#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path


SEMVER_RE = re.compile(
    r"^(0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)"
    r"(-[0-9A-Za-z.-]+)?"
    r"(\+[0-9A-Za-z.-]+)?$"
)
DEFAULT_TARGETS = ["darwin/arm64", "darwin/amd64", "linux/amd64"]


def fail(message: str) -> "NoReturn":
    print(message, file=sys.stderr)
    sys.exit(1)


def run(command: list[str], *, cwd: Path, capture_output: bool = False) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        command,
        cwd=str(cwd),
        check=True,
        text=True,
        capture_output=capture_output,
    )


def normalize_version(raw: str) -> tuple[str, str]:
    version = raw[1:] if raw.startswith("v") else raw
    if not SEMVER_RE.fullmatch(version):
        fail(f"invalid version {raw!r}")
    return version, f"v{version}"


def archive_path(repo_root: Path, target: str) -> Path:
    goos, goarch = target.split("/", 1)
    archive_dir = repo_root / "dist" / "release" / f"margo-{goos}-{goarch}"
    extension = "zip" if goos == "windows" else "tar.gz"
    return archive_dir / f"margo-{goos}-{goarch}.{extension}"


def host_target(repo_root: Path) -> str:
    result = run(["go", "env", "GOOS", "GOARCH"], cwd=repo_root, capture_output=True)
    parts = [line.strip() for line in result.stdout.splitlines() if line.strip()]
    if len(parts) != 2:
        fail(f"unexpected go env output while determining host target: {result.stdout!r}")
    return f"{parts[0]}/{parts[1]}"


def extract_tar(archive: Path, destination: Path) -> None:
    run(["tar", "-xzf", str(archive), "-C", str(destination)], cwd=destination)


def smoke_test_archive(repo_root: Path, archive: Path, version: str, target: str) -> str:
    goos, _goarch = target.split("/", 1)
    package_dir = f"margo-{version}"
    binary_name = "margo.exe" if goos == "windows" else "margo"

    with tempfile.TemporaryDirectory(prefix="margo-release-verify-") as tmp:
        tmpdir = Path(tmp)
        extract_tar(archive, tmpdir)
        binary = tmpdir / package_dir / binary_name
        if not binary.exists():
            fail(f"expected binary missing from archive {archive}")
        if target != host_target(repo_root):
            file_result = run(["file", str(binary)], cwd=repo_root, capture_output=True)
            return f"skipped execution for non-host target: {file_result.stdout.strip()}"
        binary.chmod(binary.stat().st_mode | 0o111)
        result = run([str(binary), "version"], cwd=repo_root, capture_output=True)
        output = result.stdout.strip()
        if version not in output:
            fail(f"unexpected version output from {archive}: {output!r}")
        return output


def verify_github_release(repo_root: Path, repo: str, tag: str) -> dict[str, str]:
    if shutil.which("gh") is None:
        fail("gh is required for GitHub release verification")
    result = run(
        ["gh", "release", "view", tag, "--repo", repo, "--json", "tagName,name,url,isDraft,isPrerelease"],
        cwd=repo_root,
        capture_output=True,
    )
    payload = json.loads(result.stdout)
    if payload["tagName"] != tag:
        fail(f"GitHub release tag mismatch: expected {tag}, got {payload['tagName']}")
    if payload["isDraft"]:
        fail(f"GitHub release {tag} is still a draft")
    return {
        "tag": payload["tagName"],
        "name": payload["name"],
        "url": payload["url"],
    }


def verify_tap_formula(tap_path: Path, tag: str) -> Path:
    formula = tap_path / "Formula" / "margo.rb"
    if not formula.exists():
        fail(f"tap formula not found: {formula}")
    content = formula.read_text(encoding="utf-8")
    expected_url = f"https://github.com/jjanuszczak/margo/archive/refs/tags/{tag}.tar.gz"
    if expected_url not in content:
        fail(f"tap formula does not reference {expected_url}")
    if "REPLACE_WITH_SOURCE_TARBALL_SHA256" in content:
        fail("tap formula still contains the placeholder sha256")
    sha_match = re.search(r'sha256\s+"([0-9a-f]{64})"', content)
    if not sha_match:
        fail("tap formula is missing a concrete sha256 value")
    return formula


def run_brew_checks(repo_root: Path, formula: Path) -> list[str]:
    return run_brew_checks_with_formula_name(repo_root, "jjanuszczak/margo/margo")


def run_brew_checks_with_formula_name(repo_root: Path, formula_name: str) -> list[str]:
    if shutil.which("brew") is None:
        fail("brew is required for Homebrew verification")
    info = run(
        ["brew", "info", "--json=v2", formula_name],
        cwd=repo_root,
        capture_output=True,
    )
    payload = json.loads(info.stdout)
    installed = payload.get("formulae", [{}])[0].get("installed", [])
    install_or_upgrade = (
        ["brew", "upgrade", "--build-from-source", formula_name]
        if installed
        else ["brew", "install", "--build-from-source", formula_name]
    )
    commands = [
        install_or_upgrade,
        ["brew", "test", formula_name],
        ["brew", "audit", "--strict", "--formula", formula_name],
    ]
    executed: list[str] = []
    for command in commands:
        run(command, cwd=repo_root)
        executed.append(" ".join(command))
    return executed


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Verify a finished margo release and optional Homebrew tap update."
    )
    parser.add_argument("version", help="Semver release version, with or without a leading v")
    parser.add_argument(
        "--repo",
        default="jjanuszczak/margo",
        help="GitHub repository to query with gh",
    )
    parser.add_argument(
        "--tap-path",
        help="Optional checkout path for jjanuszczak/homebrew-margo",
    )
    parser.add_argument(
        "--skip-gh-release-check",
        action="store_true",
        help="Skip gh release verification",
    )
    parser.add_argument(
        "--skip-brew",
        action="store_true",
        help="Skip brew install/test/audit even when a tap path is provided",
    )
    parser.add_argument(
        "--brew-formula",
        default="jjanuszczak/margo/margo",
        help="Tap-qualified Homebrew formula name to verify",
    )
    args = parser.parse_args()

    repo_root = Path(__file__).resolve().parents[3]
    version, tag = normalize_version(args.version)
    results: dict[str, object] = {"version": version, "tag": tag, "archives": []}

    for target in DEFAULT_TARGETS:
        archive = archive_path(repo_root, target)
        if not archive.exists():
            fail(f"release archive missing: {archive}")
        output = smoke_test_archive(repo_root, archive, version, target)
        results["archives"].append({"target": target, "archive": str(archive), "version_output": output})

    if not args.skip_gh_release_check:
        results["github_release"] = verify_github_release(repo_root, args.repo, tag)

    if args.tap_path:
        tap_path = Path(args.tap_path).resolve()
        formula = verify_tap_formula(tap_path, tag)
        results["tap_formula"] = str(formula)
        if not args.skip_brew:
            results["brew_checks"] = run_brew_checks_with_formula_name(
                repo_root,
                args.brew_formula,
            )

    print(json.dumps(results, indent=2))
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
