#!/usr/bin/env python3

from __future__ import annotations

import argparse
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from pathlib import Path
from typing import Iterable


SEMVER_RE = re.compile(
    r"^(?P<core>0|[1-9]\d*)\.(0|[1-9]\d*)\.(0|[1-9]\d*)"
    r"(?P<prerelease>-[0-9A-Za-z.-]+)?"
    r"(?P<build>\+[0-9A-Za-z.-]+)?$"
)
DEFAULT_TARGETS = ["darwin/arm64", "darwin/amd64", "linux/amd64"]


def fail(message: str) -> "NoReturn":
    print(message, file=sys.stderr)
    sys.exit(1)


def run(
    command: list[str],
    *,
    cwd: Path,
    env: dict[str, str] | None = None,
    capture_output: bool = False,
) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        command,
        cwd=str(cwd),
        env=env,
        check=True,
        text=True,
        capture_output=capture_output,
    )


def normalize_version(raw: str) -> tuple[str, str]:
    version = raw[1:] if raw.startswith("v") else raw
    if not SEMVER_RE.fullmatch(version):
        fail(
            f"invalid version {raw!r}; expected semver like 0.1.1 or v0.1.1"
        )
    return version, f"v{version}"


def git_status(repo_root: Path) -> tuple[bool, list[str]]:
    result = run(
        ["git", "status", "--short"],
        cwd=repo_root,
        capture_output=True,
    )
    lines = [line for line in result.stdout.splitlines() if line.strip()]
    return len(lines) == 0, lines


def ensure_command(name: str) -> None:
    if shutil.which(name) is None:
        fail(f"required command not found on PATH: {name}")


def archive_name(target: str) -> str:
    goos, goarch = target.split("/", 1)
    extension = "zip" if goos == "windows" else "tar.gz"
    return f"margo-{goos}-{goarch}.{extension}"


def archive_path(dist_root: Path, target: str) -> Path:
    goos, goarch = target.split("/", 1)
    archive_base = f"margo-{goos}-{goarch}"
    return dist_root / archive_base / archive_name(target)


def host_target(repo_root: Path) -> str:
    result = run(["go", "env", "GOOS", "GOARCH"], cwd=repo_root, capture_output=True)
    parts = [line.strip() for line in result.stdout.splitlines() if line.strip()]
    if len(parts) != 2:
        fail(f"unexpected go env output while determining host target: {result.stdout!r}")
    return f"{parts[0]}/{parts[1]}"


def extract_tar(archive: Path, destination: Path) -> None:
    run(["tar", "-xzf", str(archive), "-C", str(destination)], cwd=destination)


def check_archives(dist_root: Path, version: str, targets: Iterable[str]) -> list[dict[str, str]]:
    verified: list[dict[str, str]] = []
    for target in targets:
        path = archive_path(dist_root, target)
        if not path.exists():
            fail(f"expected release archive not found: {path}")
        verified.append(
            {
                "target": target,
                "path": str(path),
            }
        )
    return verified


def smoke_test_archive(repo_root: Path, archive: Path, version: str, target: str) -> dict[str, str]:
    goos, _goarch = target.split("/", 1)
    expected_binary = "margo.exe" if goos == "windows" else "margo"
    package_dir = f"margo-{version}"

    with tempfile.TemporaryDirectory(prefix="margo-release-smoke-") as tmp:
        tmpdir = Path(tmp)
        if archive.suffix == ".zip":
            shutil.unpack_archive(str(archive), str(tmpdir))
        else:
            extract_tar(archive, tmpdir)

        binary = tmpdir / package_dir / expected_binary
        if not binary.exists():
            fail(f"expected binary missing from archive {archive}: {binary}")

        if target == host_target(repo_root):
            binary.chmod(binary.stat().st_mode | 0o111)
            result = run([str(binary), "version"], cwd=repo_root, capture_output=True)
            output = result.stdout.strip()
            if version not in output:
                fail(
                    f"archive smoke test failed for {archive}: expected version {version} in output {output!r}"
                )
            return {"archive": str(archive), "version_output": output}

        file_result = run(["file", str(binary)], cwd=repo_root, capture_output=True)
        return {
            "archive": str(archive),
            "version_output": "skipped execution for non-host target",
            "binary_info": file_result.stdout.strip(),
        }


def build_followup_markdown(
    *,
    tag: str,
    version: str,
    repo_root: Path,
    asset_paths: list[Path],
    tap_path: Path | None,
) -> str:
    assets_block = "\n".join(f"  {path}" for path in asset_paths)
    lines = [
        f"# Release Follow-Up For {tag}",
        "",
        "The local preparation steps completed successfully.",
        "",
        "## Required follow-on tasks",
        "",
        "1. Review the current branch and release notes content.",
        "2. Create and push the tag:",
        "   ```bash",
        f"   git tag {tag}",
        f"   git push origin {tag}",
        "   ```",
        "3. Create the GitHub release:",
        "   ```bash",
        "   gh release create "
        f"{tag} --repo jjanuszczak/margo --title {tag} --generate-notes \\",
        assets_block,
        "   ```",
        "4. Compute the Homebrew source tarball checksum:",
        "   ```bash",
        f"   ./scripts/homebrew-source-sha.sh {tag}",
        "   ```",
    ]

    if tap_path is not None:
        lines.extend(
            [
                "5. Update the tap formula:",
                "   Edit `Formula/margo.rb` in the tap checkout so that:",
                f"   - `url` points to `https://github.com/jjanuszczak/margo/archive/refs/tags/{tag}.tar.gz`",
                "   - `sha256` matches the checksum from the previous step",
                "6. Verify the tap formula:",
                "   ```bash",
                f"   brew install --build-from-source {tap_path / 'Formula' / 'margo.rb'}",
                "   brew test margo",
                f"   brew audit --strict --formula {tap_path / 'Formula' / 'margo.rb'}",
                "   ```",
                "7. Commit and push the tap update.",
            ]
        )
    else:
        lines.extend(
            [
                "5. Update `Formula/margo.rb` in `jjanuszczak/homebrew-margo` with the new tag URL and checksum.",
                "6. In the tap checkout, run `brew install --build-from-source`, `brew test margo`, and `brew audit --strict --formula ./Formula/margo.rb`.",
                "7. Commit and push the tap update.",
            ]
        )

    lines.extend(
        [
            "",
            "## Notes",
            "",
            f"- Repo root: `{repo_root}`",
            f"- Built version: `{version}`",
        ]
    )
    return "\n".join(lines) + "\n"


def main() -> int:
    parser = argparse.ArgumentParser(
        description="Prepare and package a margo release."
    )
    parser.add_argument("version", help="Semver release version, with or without a leading v")
    parser.add_argument(
        "--tap-path",
        help="Optional checkout path for jjanuszczak/homebrew-margo",
    )
    parser.add_argument(
        "--allow-dirty",
        action="store_true",
        help="Continue even if the git worktree is dirty",
    )
    parser.add_argument(
        "--skip-go-test",
        action="store_true",
        help="Skip go test ./...",
    )
    args = parser.parse_args()

    repo_root = Path(__file__).resolve().parents[3]
    version, tag = normalize_version(args.version)
    dist_root = repo_root / "dist" / "release"
    report_root = dist_root / f"release-{version}"
    manifest_path = report_root / "manifest.json"
    followup_path = report_root / "release-followup.md"
    tap_path = Path(args.tap_path).resolve() if args.tap_path else None

    ensure_command("git")
    ensure_command("go")

    clean, status_lines = git_status(repo_root)
    if not clean and not args.allow_dirty:
        fail(
            "git worktree is dirty; rerun with --allow-dirty if that is intentional.\n"
            + "\n".join(status_lines)
        )

    env = os.environ.copy()
    env["GOCACHE"] = str(repo_root / ".gocache")
    env["GOMODCACHE"] = str(repo_root / ".gomodcache")

    if not args.skip_go_test:
        run(["go", "test", "./..."], cwd=repo_root, env=env)

    release_command = [
        str(repo_root / "scripts" / "release.sh"),
        *DEFAULT_TARGETS,
    ]
    env["VERSION"] = version
    run(release_command, cwd=repo_root, env=env)

    archive_records = check_archives(dist_root, version, DEFAULT_TARGETS)
    smoke_results = [
        smoke_test_archive(repo_root, Path(record["path"]), version, record["target"])
        for record in archive_records
    ]

    report_root.mkdir(parents=True, exist_ok=True)
    asset_paths = [Path(record["path"]) for record in archive_records]
    followup_path.write_text(
        build_followup_markdown(
            tag=tag,
            version=version,
            repo_root=repo_root,
            asset_paths=asset_paths,
            tap_path=tap_path,
        ),
        encoding="utf-8",
    )

    manifest = {
        "version": version,
        "tag": tag,
        "repo_root": str(repo_root),
        "dirty_worktree": not clean,
        "git_status": status_lines,
        "archives": archive_records,
        "smoke_tests": smoke_results,
        "followup_path": str(followup_path),
    }
    manifest_path.write_text(json.dumps(manifest, indent=2) + "\n", encoding="utf-8")

    print(f"Prepared release {tag}")
    print(f"Manifest: {manifest_path}")
    print(f"Follow-up: {followup_path}")
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
