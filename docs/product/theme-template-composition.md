# Theme Template Composition Recommendation

## Summary

Margo should borrow a narrow, high-value part of Hugo's theme model:
- theme partials
- lightweight named-template composition

It should **not** try to import Hugo's full template lookup and inheritance complexity.

The current Margo model already has the right major building blocks:
- deck shell templates
- slide layout templates
- shortcodes

The problem is that, without partials, too much shell and brand-chrome responsibility accumulates inside `deck.html` and `print-deck.html`.

## Current State

Today Margo themes support:
- `layouts/deck.html`
- optional `layouts/print-deck.html`
- `layouts/slide-*.html`
- `shortcodes/*.html`

Templates are effectively parsed one file at a time for each render path:
- interactive deck render
- print deck render
- per-slide layout render

There is no theme-level `partials/` concept.

As a result:
- `deck.html` grows large and hard to maintain
- repeated brand and frame markup is duplicated
- print and interactive shells are harder to keep aligned
- related slide layouts cannot share small reusable template fragments cleanly

## Recommendation

Add **theme partials**, **deck-level partials**, and **light template composition**, but keep the scope disciplined.

### Add partials

Support:

```text
partials/*.html
themes/<name>/partials/*.html
```

These should be parsed alongside deck, print, and slide templates so templates can use standard Go template calls such as:

```gotemplate
{{ template "brand-lockup" . }}
```

Recommended precedence:
1. deck-level partial
2. theme-level partial

This mirrors the existing deck-local versus theme-level shortcode model and keeps project-specific overrides explicit.

### Add named-template composition

Allow theme files to use Go template `define`/`template` composition across parsed theme files.

This does not require a Hugo-style inheritance model. It only requires Margo to parse a small template set together rather than as isolated single files.

## What Partials Should Be Used For

Good candidates:
- brand lockup / logo rendering
- slide frame header
- slide frame footer
- deck controls
- shared `<head>` fragments
- print frame chrome
- repeated metadata blocks
- repeated wrappers shared by multiple slide layouts

Deck-level partials would also be useful for:
- deck-specific disclaimer blocks
- project-specific footer or metadata fragments
- reusable structural fragments that are too layout-oriented for a shortcode

This would immediately reduce pressure on large theme files such as:
- `deck.html`
- `print-deck.html`

## What Not To Borrow From Hugo

Margo should **not** adopt:
- Hugo's broader publishing model
- content-type and list/single lookup complexity
- deep theme inheritance chains
- large amounts of implicit template resolution
- generalized site/data/template abstractions not needed for decks

Margo is a deck compiler, not a website generator.

## Why This Is The Right Scope

This change would:
- improve theme maintainability
- reduce duplication
- make print and interactive shells easier to keep aligned
- preserve the current Margo mental model

It would do that without:
- widening product scope dramatically
- introducing heavy template magic
- making theme behavior much harder to reason about

## Recommended Implementation Shape

1. Add support for `partials/*.html` and `themes/<name>/partials/*.html`.
2. Parse deck, print, and slide templates together with available partials.
3. Resolve partial names with deck-level override precedence over theme-level partials.
4. Permit named templates via Go template `define`/`template`.
5. Keep shortcode behavior separate from partial behavior.
6. Do not add multi-level theme inheritance in the first pass.

## Product Position

This should be treated as a **theme authoring and maintainability enhancement**, not a broad architecture rewrite.

The intended outcome is:
- thinner `deck.html`
- thinner `print-deck.html`
- better reuse across slide and shell templates
- explicit deck-local overrides where a project needs structural customization
- cleaner long-term evolution of Margo themes
