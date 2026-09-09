# Deck conventions

## Project shape

- margo.yaml: deck metadata, active theme, and output configuration.
- slides/<slide-id>/index.md: one Markdown slide bundle. Slide-local assets and named notes belong in the same bundle.
- assets/: shared deck assets.
- archetypes/: authoring-time templates used by margo new slide.
- shortcodes/: deck-local content components. A deck-local shortcode overrides a theme shortcode with the same name.
- themes/<theme-name>/: deck-local themes. One theme is active at a time through margo.yaml.

## Slides

Use YAML front matter for title, order, layout, section, draft, visibility, background, image_hints, and notes when needed. Use Markdown for the slide body. Draft slides appear in serve but normal builds omit them; visibility: hidden excludes slides from normal output.

Choose an existing layout and archetype first. The default scaffold supports content, title, section, agenda, image, two-column, media-left, media-right, quote, metric, and closing layouts.

## Assets and notes

Reference a slide-local image by filename. Reference a shared asset with an assets/ path. Put named notes under slides/<slide-id>/notes/; notes stay out of print HTML and PDF output.

## Outputs

Margo builds source files into dist/. Treat dist/ as generated output. For output defects, compare the interactive HTML, print HTML, and PDF rather than changing source blindly.
