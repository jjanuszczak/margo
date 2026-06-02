---
title: Mermaid QA
order: 7
layout: content
footer_text: Diagram Regression Check
---

## Mermaid diagrams should upgrade cleanly

{{< mermaid caption="Reference-deck Mermaid smoke test" align="center" >}}
flowchart TD
  A["Author writes Markdown"] --> B["Shortcode expands in HTML"]
  B --> C["Browser loads Mermaid"]
  C --> D["Diagram renders as SVG"]
{{< /mermaid >}}

- Keep the source readable if Mermaid fails to load
- Render the upgraded SVG in served and built HTML
- Preserve the same slide content in print HTML
