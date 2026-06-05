---
title: Render Pipeline
order: 14
layout: content
section: Theming
footer_text: Rendering Model
---

## The pieces operate in sequence

1. archetypes create the initial slide files
2. slide Markdown and front matter are loaded
3. shortcodes expand inside the body
4. Markdown is converted to HTML
5. generic helpers are available to templates
6. slide layouts and partials wrap the content
7. deck and print layouts wrap the slide collection

{{< callout tone="info" >}}
The Go code should stay generic: parsing, validation, asset resolution, and model shaping. Layout-specific composition belongs in templates and partials.
{{< /callout >}}
