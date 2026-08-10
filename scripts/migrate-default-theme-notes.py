#!/usr/bin/env python3
"""Upgrade an unmodified pre-notes default theme in an existing Margo deck."""

from __future__ import annotations

import argparse
import hashlib
import re
import shutil
import sys
from datetime import datetime, timezone
from pathlib import Path


LEGACY_DECK_SHA256 = "ffd3b484c9c9d6a958fa996c8aa4622a5df4a45e2648b4d801320fc83a4cc049"
LEGACY_CSS_SHA256 = "e49691fe045885333623bb911a7c297a87487dc14be8147bf052f7ab7f9b2214"


def digest(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def configured_theme(config: str) -> str | None:
    match = re.search(r"(?ms)^theme:\s*\n(?:^[ \t]+.*\n)*?^[ \t]+name:\s*([^#\s]+)", config)
    return match.group(1).strip('"\'') if match else None


def enable_notes(config: str) -> str:
    if re.search(r"(?m)^presentation:\s*$", config):
        raise ValueError("margo.yaml already has presentation settings; add presentation.navigation.notes: true manually")
    return config.rstrip() + "\n\npresentation:\n  navigation:\n    notes: true\n"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("deck_root", type=Path, help="Path to the existing deck project")
    args = parser.parse_args()

    deck_root = args.deck_root.resolve()
    config_path = deck_root / "margo.yaml"
    if not config_path.is_file():
        parser.error(f"{config_path} does not exist")
    config = config_path.read_text()
    theme_name = configured_theme(config)
    if theme_name != "default":
        parser.error("only an unmodified older default theme is supported; migrate custom themes manually")

    layout_path = deck_root / "themes" / theme_name / "layouts" / "deck.html"
    css_path = deck_root / "themes" / theme_name / "assets" / "theme.css"
    if not layout_path.is_file() or not css_path.is_file():
        parser.error("default theme deck.html and theme.css are required")
    if "data-notes-toggle" in layout_path.read_text() and re.search(r"(?ms)^presentation:.*?notes:\s*true", config):
        print("notes migration already applied; no changes made")
        return 0
    if digest(layout_path) != LEGACY_DECK_SHA256 or digest(css_path) != LEGACY_CSS_SHA256:
        parser.error("theme differs from the supported older default scaffold; no files changed. Merge the current default theme controls manually.")

    source_root = Path(__file__).resolve().parents[1]
    source_layout = source_root / "examples/reference-deck/themes/default/layouts/deck.html"
    source_css = source_root / "examples/reference-deck/themes/default/assets/theme.css"
    if not source_layout.is_file() or not source_css.is_file():
        parser.error("current default theme templates are unavailable beside this script")

    backup_dir = deck_root / ".margo-backups" / ("notes-" + datetime.now(timezone.utc).strftime("%Y%m%dT%H%M%SZ"))
    backup_dir.mkdir(parents=True)
    for path in (config_path, layout_path, css_path):
        target = backup_dir / path.relative_to(deck_root)
        target.parent.mkdir(parents=True, exist_ok=True)
        shutil.copy2(path, target)

    config_path.write_text(enable_notes(config))
    shutil.copy2(source_layout, layout_path)
    shutil.copy2(source_css, css_path)
    print(f"updated {deck_root}")
    print(f"backup: {backup_dir}")
    return 0


if __name__ == "__main__":
    try:
        raise SystemExit(main())
    except ValueError as error:
        print(f"error: {error}", file=sys.stderr)
        raise SystemExit(2)
