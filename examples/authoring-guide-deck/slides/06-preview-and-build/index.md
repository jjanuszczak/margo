---
title: Preview, Build, And Clean
order: 6
layout: content
section: Workflow
footer_text: Dev Loop
---

## The loop is intentionally short

- `margo serve` previews at `127.0.0.1:1313`
- `margo build` writes the configured outputs
- `--port` handles conflicts cleanly
- `--include-drafts` is useful during editing
- `margo clean` removes generated output

{{< stat value="3" label="Core Commands" detail="Serve, build, and clean cover most day-to-day authoring." />}}
