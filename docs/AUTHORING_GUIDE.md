# Margo Authoring Guide

This guide explains how to use the current `margo` prototype to:
- create a new deck
- create new slides
- choose a theme
- pick layouts and archetypes
- embed images, shared assets, and video
- build and preview the deck

This reflects the repo's current implementation, not the long-term product vision.

## 1. Build the CLI

From the repo root:

```bash
go build -o ./bin/margo ./cmd/margo
```

You can also run commands with `go run ./cmd/margo ...`, but using `./bin/margo` is easier once you are working inside deck folders.

## 2. Create a New Deck

Create a new deck in a new directory:

```bash
./bin/margo new my-deck
cd my-deck
```

Initialize a deck in the current directory instead:

```bash
mkdir my-deck
cd my-deck
../bin/margo init
```

The generated deck looks like this:

```text
my-deck/
  margo.yaml
  slides/
    01-title/index.md
    02-why/index.md
  themes/
    default/
  archetypes/
    default/
    title/
    section/
    agenda/
    image/
    two-column/
    media-left/
    media-right/
    quote/
    metric/
    closing/
  assets/
  shortcodes/
```

## 3. Understand `margo.yaml`

The root config file controls deck metadata, theme choice, and outputs.

Example:

```yaml
version: 1

deck:
  title: My Deck
  description: Internal review
  language: en
  logo: MARGO
  footer: Internal Strategy Review

theme:
  name: default
  color_mode: light
  typography: editorial
  accent_color: "#8f6f33"

outputs:
  html: true
  pdf: true
```

### Current default-theme options

The built-in `default` theme currently supports:
- `color_mode`: `light` or `dark`
- `typography`: theme preset, currently `editorial` or `executive`
- `accent_color`: any CSS color string

If you provide an unsupported value such as `color_mode: sepia`, `margo build` now fails with a theme option validation error that lists the allowed values.

To switch the deck to dark mode:

```yaml
theme:
  name: default
  color_mode: dark
  typography: executive
  accent_color: "#4db6ac"
```

### Logo support

In the current default theme, `deck.logo` supports either:
- plain text
- a shared deck asset path such as `assets/company-logo.svg`

Text logo example:

```yaml
deck:
  logo: MARGO
```

Image logo example:

```yaml
deck:
  logo: assets/company-logo.svg
```

The default theme will render text directly, or render an `<img>` when the value resolves to a shared image asset.

### Deck-level snippets

The current config also supports approved deck-level snippet slots:

```yaml
snippets:
  head: |
    <meta name="analytics-env" content="staging">
  body_end: |
    <script>window.__deckMode = "preview";</script>
```

Current supported locations:
- `snippets.head`
- `snippets.body_end`

These are injected into both `serve` and `build` output in the default theme.

## 4. Preview and Build

Build the configured outputs:

```bash
../bin/margo build
```

Preview locally:

```bash
../bin/margo serve
```

Useful variants:

```bash
../bin/margo build --include-drafts
../bin/margo serve --no-open
../bin/margo clean
```

Current output locations:
- `dist/html/index.html`
- `dist/pdf/deck.pdf` when PDF is enabled and local Chrome works

## 5. Create New Slides

From inside a deck:

```bash
../bin/margo new slide roadmap
```

If you do not pass an archetype, `margo` will prompt you to choose one interactively.

Create a slide with a specific archetype:

```bash
../bin/margo new slide roadmap --archetype agenda
../bin/margo new slide hero --archetype image
../bin/margo new slide split --archetype two-column
../bin/margo new slide customer --archetype media-right
../bin/margo new slide architecture --archetype media-left
../bin/margo new slide north-star --archetype metric
../bin/margo new slide quote --archetype quote
../bin/margo new slide close --archetype closing
```

Each slide is a bundle:

```text
slides/<slide-id>/index.md
```

You can place slide-local assets beside `index.md`.

## 6. Slide Front Matter

A typical slide starts like this:

```yaml
---
title: Why Margo
order: 2
layout: content
section: Strategy
footer_text: Product Strategy
background:
  color: "#fbf6ec"
  image: backdrop.svg
  overlay: "linear-gradient(180deg, rgba(255,255,255,0.24), rgba(255,255,255,0))"
  opacity: 1
image_hints:
  fit: contain
  position: center
  caption: Authoring and output model overview
notes:
  - Mention Hugo mental model
  - Emphasize HTML-first output
---
```

### Current standardized slide fields

- `title`
- `order`
- `section`
- `layout`
- `type`
- `draft`
- `visibility`
- `hide_logo`
- `hide_footer`
- `footer_text`
- `background`
- `image_hints`
- `notes`

### Drafts and visibility

```yaml
draft: true
visibility: hidden
```

Current behavior:
- `serve` includes drafts and marks them visibly
- `build` excludes drafts unless `--include-drafts` is used
- `visibility: hidden` slides are excluded from normal output

### Notes

Notes can live in front matter:

```yaml
notes:
  - Open with the customer story
  - Do not over-explain PDF export yet
```

Or in a body section:

```md
## Notes

Mention the Hugo mental model.
```

Notes are preserved in the model but excluded from normal slide HTML.

## 7. Choose a Layout or Archetype

In practice, archetypes and layouts are closely related in the current prototype.

### Built-in archetypes in the scaffold

- `default`
- `title`
- `section`
- `agenda`
- `image`
- `two-column`
- `media-left`
- `media-right`
- `quote`
- `metric`
- `closing`

### Current layout names

These are the layout names used by the default theme:

- `content`
- `title`
- `section`
- `agenda`
- `image`
- `two-column`
- `media-left`
- `media-right`
- `quote`
- `metric`
- `closing`

If you want to switch a slide manually:

```yaml
layout: media-right
```

### Layout guidance

- `content`: general-purpose text/content slide
- `title`: opening slide
- `section`: divider slide
- `agenda`: ordered list of topics
- `image`: visual-first slide
- `two-column`: split text or mixed content
- `media-left` / `media-right`: first image becomes the media pane, remaining content becomes the text pane
- `quote`: pull quote with attribution
- `metric`: single KPI slide
- `closing`: thank-you or wrap-up slide

## 8. Embed Images

### Slide-local images

Put the image in the same slide bundle:

```text
slides/02-why/
  index.md
  diagram.svg
```

Reference it in Markdown:

```md
![Margo flow](diagram.svg)
```

Reference it in background front matter:

```yaml
background:
  image: diagram.svg
```

### Shared deck assets

Put reusable assets under the deck-level `assets/` directory:

```text
assets/
  shared-grid.svg
  video-poster.svg
```

Reference them from slides:

```md
![Shared brand texture](assets/shared-grid.svg)
```

Or from front matter:

```yaml
background:
  image: assets/shared-grid.svg
```

Current behavior:
- slide-local assets are staged into `dist/html/slides/<slide-id>/...`
- shared assets are staged into `dist/html/assets/...`

## 9. Control Backgrounds and Image Presentation

### Background treatment

```yaml
background:
  color: "#101820"
  image: backdrop.svg
  overlay: "linear-gradient(180deg, rgba(0,0,0,0.22), rgba(0,0,0,0.55))"
  opacity: 1
```

### Image hints

```yaml
image_hints:
  fit: contain
  position: center
  caption: Architecture overview
```

Current supported image hints in the default theme:
- `fit`
- `position`
- `caption`

## 10. Use Shortcodes

The current default theme ships with theme-provided shortcodes.

### Callout

```md
{{< callout tone="info" >}}
Theme-provided shortcodes make expressive slides possible without abandoning Markdown.
{{< /callout >}}
```

### Columns

```md
{{< columns >}}
{{< column >}}
### Authoring

- Markdown
- Front matter
- Archetypes
{{< /column >}}
{{< column >}}
### Output

- HTML first
- PDF next
- PPTX later
{{< /column >}}
{{< /columns >}}
```

### Stat

```md
{{< stat value="20" label="Slides" detail="A reasonably sized deck should still feel fast." />}}
```

### Video

```md
{{< video src="https://example.com/demo.mp4" poster="assets/video-poster.svg" caption="Shortcodes can carry shared deck assets cleanly." />}}
```

Current `video` behavior:
- `src` may be an external URL or local asset path
- `poster` may be a slide-local or deck-level asset path

### Deck-local shortcodes

The scaffold also creates a deck-local shortcode at:

```text
shortcodes/eyebrow.html
```

Example use:

```md
{{< eyebrow label="Why this exists" />}}
```

## 11. Reuse Project-Local Markdown with Includes

`margo` supports explicit project-local Markdown includes.

Supported syntax:

```md
{{< include "shared/summary.md" >}}
```

Current rules:
- include paths are project-local
- include paths may not escape the project root
- includes are expanded before normal Markdown rendering
- nested includes work
- include cycles are rejected with a clear error

Example project structure:

```text
my-deck/
  shared/
    authoring.md
    output.md
  slides/
    02-why/index.md
```

Example slide usage:

```md
{{< columns >}}
{{< column >}}
{{< include "shared/authoring.md" >}}
{{< /column >}}
{{< column >}}
{{< include "shared/output.md" >}}
{{< /column >}}
{{< /columns >}}
```

This keeps repeated Markdown content in one place without introducing a parameterized content system.

## 12. Create or Choose a Theme

### Use the default theme

Every scaffolded deck starts with:

```yaml
theme:
  name: default
```

### Create a new theme

From inside a deck:

```bash
../bin/margo new theme custom
```

That creates a default-inspired theme scaffold under:

```text
themes/custom/
```

### Create a blank theme

```bash
../bin/margo new theme minimalist --blank
```

### Switch themes

Edit `margo.yaml`:

```yaml
theme:
  name: custom
```

Current rule: one active theme per deck.

## 13. Edit Theme Layouts

The default theme uses:

```text
themes/default/theme.yaml
themes/default/layouts/deck.html
themes/default/layouts/slide-default.html
themes/default/layouts/slide-title.html
themes/default/layouts/slide-section.html
themes/default/layouts/slide-agenda.html
themes/default/layouts/slide-image.html
themes/default/layouts/slide-two-column.html
themes/default/layouts/slide-media-left.html
themes/default/layouts/slide-media-right.html
themes/default/layouts/slide-quote.html
themes/default/layouts/slide-metric.html
themes/default/layouts/slide-closing.html
themes/default/shortcodes/*.html
```

The easiest way to customize appearance today is:
1. create a new theme scaffold
2. point `margo.yaml` at it
3. edit the theme's `layouts/` and `shortcodes/`

## 14. Common Authoring Patterns

### Standard content slide

```yaml
---
title: Product Strategy
order: 4
layout: content
section: Strategy
---
```

### Explicit section divider slide

```bash
../bin/margo new slide strategy --archetype section
```

### Two-column slide

```md
## What changed

- Faster builds
- Better layouts

<!-- column-break -->

## What is next

- Theme options
- PDF refinement
```

### Media split slide

For `media-left` or `media-right`, place the first image in the body before the remaining content:

```md
![Customer spotlight](spotlight.svg)

## Customer Story

- Clearer authoring flow
- Better visual system
- Shared brand assets
```

## 15. Known Current Limitations

This guide reflects the current prototype. Important limitations:

- PDF depends on a local Chrome/Chromium-compatible browser and may fail in restricted remote environments
- PPTX export is not implemented
- presenter mode is not implemented
- theme APIs and project conventions may still evolve
- browser auto-refresh is currently wired through the scaffolded default theme

## 16. Good Files to Study

If you want real working examples, start with:

- [examples/reference-deck/margo.yaml](../examples/reference-deck/margo.yaml)
- [examples/reference-deck/slides/02-why/index.md](../examples/reference-deck/slides/02-why/index.md)
- [examples/reference-deck/themes/default/theme.yaml](../examples/reference-deck/themes/default/theme.yaml)
- [examples/reference-deck/themes/default/layouts/deck.html](../examples/reference-deck/themes/default/layouts/deck.html)
- [examples/benchmark-deck](../examples/benchmark-deck)
