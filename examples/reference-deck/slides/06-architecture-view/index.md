---
title: Architecture View
order: 6
layout: media-left
type: media-left
---

## Shared assets should work across slides

{{< github-repo repo="jjanuszczak/margo" caption="Static repo card shortcode for product and engineering decks." />}}

![Shared brand texture](assets/shared-grid.svg)

- Deck-wide brand assets belong in `assets/`
- Slide bundles still own local imagery
- Both should render and stage predictably
