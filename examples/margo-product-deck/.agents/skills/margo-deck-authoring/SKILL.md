---
name: margo-deck-authoring
description: Create, edit, review, build, upgrade, or package a Margo deck. Use for slide content, front matter, assets, notes, archetypes, deck outputs, safe scaffold upgrades, and portable deck archives. Do not use for theme implementation work or GitHub Pages setup.
---

1. Read AGENTS.md and references/conventions.md before changing the deck.
2. Read references/commands.md before running a Margo command.
3. Keep source changes in deck-owned files. Do not edit dist/ output.
4. Use the smallest relevant build or test to verify the change. Build the deck when author-facing output changes.
5. Before upgrading an existing project, run margo upgrade --plan. Apply only with margo upgrade --apply after reviewing additions, updates, and preserved custom files.
