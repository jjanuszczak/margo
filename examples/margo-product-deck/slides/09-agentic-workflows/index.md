---
title: Ready for agentic workflows
order: 9
layout: content
section: Workflow
---

## Give an agent a real project, not a browser tab full of hidden state.

{{< columns >}}
{{< column >}}
### Agent-friendly source

- Markdown slide bundles and YAML front matter
- Shared and slide-local assets with explicit paths
- Named notes that retain the detailed context
- Deck-local `AGENTS.md` guidance and authoring skills
{{< /column >}}
{{< column >}}
### Scriptable verification

- A deterministic local CLI for create, build, serve, pack, and unpack
- Explicit theme and output configuration in `margo.yaml`
- Reviewable diffs instead of opaque editor state
- A static HTML artifact ready for automated publishing
{{< /column >}}
{{< /columns >}}

{{< callout tone="info" >}}
An agent can inspect the project, update the source, run a build, and hand back a clear diff without reconstructing the deck in a browser.
{{< /callout >}}
