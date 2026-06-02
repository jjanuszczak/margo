---
title: Configure margo.yaml
order: 5
layout: content
section: Workflow
footer_text: Root Config
---

## One file sets the deck contract

- deck metadata defines title, logo, language, and footer
- theme selection activates the visual system
- outputs control which artifacts are built
- snippets inject approved shell-level HTML when needed

{{< callout tone="info" >}}
Treat `margo.yaml` as the deck's operating contract. It is the first file to read when you are orienting yourself in a new project.
{{< /callout >}}

```yaml
theme:
  name: default
  color_mode: dark
  typography: executive
  accent_color: "#4db6ac"
```
