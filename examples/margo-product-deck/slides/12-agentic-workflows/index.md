---
title: Ready for agentic workflows
order: 12
layout: content
section: Workflow
---

## Give an agent a real project, not a browser tab full of hidden state.

Agents work from the project source, build it, and return a clear diff, not a reconstructed browser deck. Evidence-first triage identifies whether the deck, theme, Margo, or local environment owns a failure.

{{< columns >}}
{{< column >}}
### Agent-friendly source

- Markdown slide bundles and YAML front matter
- Shared and slide-local assets with explicit paths
- Named notes that retain the detailed context
- Deck-local guidance for authoring, themes, Pages, and error triage
{{< /column >}}
{{< column >}}
### Scriptable verification

- A deterministic local CLI for create, build, serve, pack, and unpack
- Explicit theme and output configuration in `margo.yaml`
- Reviewable diffs instead of opaque editor state
- Safe upgrades preserve authored work and customized guidance

{{< /column >}}
{{< /columns >}}
