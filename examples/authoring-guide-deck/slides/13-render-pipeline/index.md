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
5. slide layouts wrap the content
6. deck and print layouts wrap the slide collection
7. partials provide reusable layout fragments

{{< callout tone="info" >}}
The Go code should stay generic: parsing, validation, asset resolution, and model shaping. Markup shape belongs in templates.
{{< /callout >}}
