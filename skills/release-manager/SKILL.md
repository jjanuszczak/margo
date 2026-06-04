---
name: release-manager
description: Prepare and package a new margo release, generate any required follow-on tasks for tagging, GitHub releases, and Homebrew tap updates, and automate verification of the finished release. Use when shipping a new version or validating that a published release is installable.
---

# Release Manager

This skill automates the current `margo` release flow around:

- `scripts/release.sh`
- `scripts/homebrew-source-sha.sh`
- the personal tap model documented in `docs/HOMEBREW.md`

## Workflow

### 1. Prepare A Release

Run the preparation script from the repo root:

```bash
python3 skills/release-manager/scripts/prepare_release.py 0.1.1
```

This script:

- validates the requested version
- checks the local git state
- runs `go test ./...` with a repo-local `GOCACHE`
- builds release archives with `scripts/release.sh`
- smoke-tests each built archive by extracting it and running `margo version`
- writes a release manifest and a follow-up checklist under `dist/release/release-<version>/`

Optional flags:

- `--allow-dirty`: continue even if the worktree is dirty
- `--skip-go-test`: skip the full Go test suite
- `--tap-path /path/to/homebrew-margo`: include tap-specific follow-up instructions

### 2. Verify The Finished Release

After the Git tag, GitHub release, and optional tap update exist, run:

```bash
python3 skills/release-manager/scripts/verify_release.py 0.1.1 --tap-path /path/to/homebrew-margo
```

This script:

- re-validates the built archives
- verifies the GitHub release via `gh` when available
- checks that the tap formula points at the expected tag and has a non-placeholder checksum
- runs `brew install` or `brew upgrade --build-from-source`, then `brew test`, and `brew audit --strict` using the tap-qualified formula name when a tap checkout is available

Useful flags:

- `--skip-gh-release-check`
- `--skip-brew`
- `--brew-formula jjanuszczak/margo/margo`

## Instructions

- Use this skill for the current repo’s manual release model only. Do not introduce bottles, GoReleaser, or `homebrew-core` steps unless the repo docs change.
- Prefer running `prepare_release.py` first and reading its generated checklist before creating tags or GitHub releases.
- If networked `gh` or `brew` steps cannot be completed in the current environment, report the exact generated follow-up tasks to the user instead of paraphrasing them loosely.

## Self-Evaluation

You MUST verify the skill after changing it:

```bash
python3 skills/release-manager/evals/runner.py
```

Read:

- `skills/release-manager/evals/reports/latest_results.json`

If the evaluation fails, fix the scripts or instructions and re-run the evaluation before handing the work back.
