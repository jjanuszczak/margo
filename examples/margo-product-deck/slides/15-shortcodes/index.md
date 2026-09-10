---
title: Shortcodes for content that needs specialized rendering
order: 15
layout: content
section: Themes
---

## Write the source. Let the theme render the visual.

### Math equations

{{< math >}}
i\hbar\frac{\partial}{\partial t}\Psi(\mathbf{r},t) = \hat{H}\Psi(\mathbf{r},t)
{{< /math >}}

### Flowcharts

{{< mermaid align="left" >}}
flowchart LR
  A[Author writes Markdown] --> B{Needs a specialized visual?}
  B -->|Yes| C[Use a shortcode]
  B -->|No| D[Use Markdown]
  C --> E[Build the deck]
  D --> E
{{< /mermaid >}}
