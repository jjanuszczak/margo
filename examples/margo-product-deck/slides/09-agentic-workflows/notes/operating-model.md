## Agent operating model

Margo projects are deliberately legible to coding agents: the configuration, source slides, assets, theme, archetypes, and agent guidance live in the deck directory. Generated `dist/` output is not the source of truth.

An agentic workflow should read `AGENTS.md`, select the deck- or theme-authoring skill, make narrow source changes, run `margo build`, and report the result. Theme templates and shortcodes are trusted local project code, so an agent should inspect imported theme content before running it.

## Release review

Keep this slide aligned with the actual CLI and scaffolding behavior. If portable agent guidance, deck archives, or scriptable build behavior changes, update this source and its notes before the release tag.
