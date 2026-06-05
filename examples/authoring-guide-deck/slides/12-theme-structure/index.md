---
title: Theme Structure And Overrides
order: 13
layout: two-column
type: two-column
section: Theming
footer_text: Theme Anatomy
---

### A theme can provide

- `layouts/deck.html`
- `layouts/print-deck.html`
- `layouts/slide-*.html`
- `partials/*.html`
- `assets/theme.css`
- `shortcodes/*.html`

<!-- column-break -->

### A deck can override locally

- `partials/*.html`
- `shortcodes/*.html`
- `theme` options in `margo.yaml`

### Default slide regions

- `chrome`: logo, slide number, footer text
- `header`: section label plus title
- `context`: optional supporting setup
- `body`: the main composition
- `annotations`: caption or source
- `footer`: persistent deck metadata

This keeps project-specific variation explicit rather than magical.
