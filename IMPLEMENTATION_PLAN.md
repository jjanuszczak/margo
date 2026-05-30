# Margo v1 Implementation Plan

## 1. Purpose

This document turns the v1 PRD into an execution plan.

It is optimized for:
- proving the core product quickly
- sequencing work so the CLI is usable early
- controlling scope around theming, authoring, and export
- preserving room for later PDF hardening and eventual PPTX work

The implementation plan assumes `Go` as the primary implementation language, while keeping product requirements anchored in [PRD.md](./PRD.md).

## 2. Delivery Strategy

### Core strategy
Build `margo` as an HTML-first deck compiler with a local dev server, a strict content model, and a minimal but formal theme contract.

Do not start with:
- PDF perfection
- generalized plugin architecture
- complex incremental invalidation
- theme inheritance
- cross-platform polish

### Thin-slice objective
The first meaningful milestone is:

`margo new` -> `margo serve` -> edit a slide bundle -> see a styled deck in browser -> `margo build` -> get `dist/html/index.html`

If that loop is not solid, the rest is premature.

## 3. Proposed System Architecture

### Major subsystems
1. CLI layer
2. Project loader and configuration parser
3. Content discovery and deck graph builder
4. Markdown and shortcode rendering pipeline
5. Theme resolution and template renderer
6. HTML presentation runtime generator
7. Dev server and file watcher
8. Build/export pipeline
9. Diagnostics and validation layer
10. Scaffolding system for deck, slide, and theme creation

### High-level data flow
1. Load root config and resolve project paths.
2. Discover theme, archetypes, slides, deck assets, and optional manifest.
3. Parse slide bundles and front matter.
4. Build an in-memory deck model.
5. Resolve ordering, visibility, drafts, sections, and inherited metadata.
6. Render Markdown plus shortcodes into structured slide content.
7. Apply theme layouts, partials, and deck-level theme options.
8. Emit HTML artifact and asset tree.
9. Optionally invoke PDF generation from HTML output.

## 4. Repository And Package Shape

Exact package names can change, but the codebase should likely separate into these areas:

- `cmd/margo`
- `internal/cli`
- `internal/config`
- `internal/project`
- `internal/content`
- `internal/deck`
- `internal/frontmatter`
- `internal/notes`
- `internal/theme`
- `internal/archetype`
- `internal/shortcode`
- `internal/render`
- `internal/output/html`
- `internal/output/pdf`
- `internal/serve`
- `internal/watch`
- `internal/diagnostics`
- `internal/scaffold`
- `internal/fsutil`

This keeps the content model, renderer, and CLI concerns separate enough that the implementation stays testable.

## 5. Core Domain Model

The first serious engineering task is to define stable internal models before writing too much rendering code.

### Key structs or equivalent internal models

#### ProjectConfig
Owns:
- deck metadata
- theme selection
- output settings
- language metadata
- snippet injection
- theme option overrides

#### ThemeMetadata
Owns:
- theme name
- version
- exposed config options
- required template entrypoints
- bundled shortcode/archetype discovery metadata as needed

#### ArchetypeMetadata
Owns:
- archetype name
- description
- default layout/type
- suggested content skeleton

#### SlideFrontMatter
Owns standardized fields such as:
- title
- order
- section
- layout/type
- notes references or inline notes
- draft
- visibility
- footer overrides
- logo suppression
- background settings

#### SlideBundle
Owns:
- bundle path
- source `index.md`
- local assets
- parsed front matter
- body Markdown
- extracted notes

#### SectionModel
Owns:
- section identity
- title
- metadata defaults
- section divider intent

#### DeckModel
Owns:
- deck metadata
- resolved slides
- section structure
- theme reference
- output config
- diagnostics

#### RenderedSlide
Owns:
- resolved layout
- rendered HTML fragment
- resolved asset references
- runtime flags such as draft marker visibility

### Important design constraint
Preserve semantic structure in the deck model rather than flatten everything directly into HTML too early. This is necessary for future PDF hardening and especially later PPTX export.

## 6. File And Content Resolution Rules

These rules should be made explicit in code early, because many later features depend on them.

### Resolution order
1. Root config
2. Theme metadata and required theme files
3. Optional manifest
4. Slide bundles under `slides/`
5. Shared deck assets
6. Archetype definitions

### Ordering logic
- If manifest exists, it controls final slide order.
- Otherwise use slide front matter order.
- Hidden/skipped slides are excluded from outputs by default.
- Drafts are excluded from `build` by default and included in `serve`.

### Section logic
- Section metadata can flow into member slides.
- Section inheritance stays limited to metadata/presentation hints.
- Theme/layout selection remains slide-owned or theme-defaulted.

## 7. Markdown And Rendering Stack

### Recommendation
Use a Go Markdown engine that supports:
- front matter separation
- fenced code blocks
- syntax highlighting integration
- raw HTML passthrough
- extension hooks for shortcodes or pre/post-processing

The exact library can be chosen during implementation, but the pipeline should not tightly couple shortcode resolution to template rendering internals.

### Proposed rendering stages
1. Read `index.md`
2. Split front matter and body
3. Extract notes section if present
4. Resolve includes
5. Expand shortcodes
6. Convert Markdown to HTML
7. Post-process images, code metadata, and asset paths
8. Pass structured content into theme layout templates

### Includes
- Implement project-local include resolution before or during Markdown preprocessing.
- Keep includes simple and explicit in v1.
- Reject cyclical includes with a clear error.

### Shortcodes
- Support both theme-defined and deck-defined shortcodes.
- Resolve deck-defined shortcodes from project scope and theme-defined ones from theme scope.
- Keep the execution model explicit and deterministic.

## 8. Theme System Plan

### Minimal v1 contract
The initial required theme contract should likely include:
- theme metadata file
- base deck template
- default slide layout
- theme CSS entrypoint
- optional theme JS entrypoint

Everything else should remain optional and convention-based.

### Theme loading
Implement theme loading as a strict validation step:
- verify metadata file exists
- verify required entrypoints exist
- load theme-defined config schema/options
- discover theme partials, layouts, and shortcodes

### Default theme strategy
Treat the default theme as both:
- a production feature
- an implementation test fixture

The default theme should be built in parallel with the renderer, not after it. It will force the contract to become real.

## 9. HTML Runtime Plan

### Scope
The HTML runtime should stay small.

Responsibilities:
- slide-by-slide navigation
- keyboard navigation
- basic browser presentation behavior
- draft markers in `serve`

Avoid in v1:
- presenter mode
- fragment/reveal logic
- route-per-slide complexity
- heavy JS-driven layout behavior

### Recommended model
Generate one HTML document containing discrete slide containers plus a very small runtime script for navigation. This matches the PRD while keeping export and offline behavior simpler.

## 10. PDF Plan

### Scope
PDF is a derivative export, not a second renderer.

### Recommended implementation path
1. Produce stable HTML output first.
2. Add a PDF exporter that drives a local Chrome/Chromium-class browser in print mode.
3. Use a deterministic HTML entrypoint from `dist/html`.

### Engineering stance
Do not implement PDF until the HTML output and theme layout contracts are stable enough to avoid churn.

### Development workflow
- `serve` should expose a way to request current PDF export on demand.
- PDF generation should not run automatically on every change.

## 11. CLI Plan

### First commands to implement
1. `margo build`
2. `margo new`
3. `margo init`
4. `margo serve`

These establish the core product loop.

### Next commands
5. `margo new slide`
6. `margo new theme`
7. `margo clean`

### CLI behavior notes
- Interactive prompts should be layered on top of non-interactive command primitives.
- The CLI layer should not own business logic; it should call internal services with fully resolved options.

## 12. Scaffolding Plan

Scaffolding is a product feature, not a convenience script. It should be built early.

### `margo new` / `margo init`
Must generate:
- root config
- slide directory with example bundles
- default theme wiring
- archetypes
- starter assets as needed

### `margo new slide`
Must:
- discover available archetypes
- prompt when archetype is omitted
- create a slide bundle with `index.md`
- place the slide at the end by default

### `margo new theme`
Should support:
- blank theme scaffold
- default-theme-inspired scaffold

## 13. Validation And Diagnostics Plan

Diagnostics quality matters early because the tool is convention-heavy.

### Build failure categories
Hard failures:
- invalid front matter
- missing required theme files
- missing layouts
- broken template references
- malformed includes or include cycles

Warnings:
- missing referenced assets
- recoverable notes/body parsing issues where content can still render
- non-fatal theme option mismatches if a safe fallback exists

### Diagnostics requirements
- include path and line number where possible
- distinguish warnings from errors clearly
- use consistent terminal formatting from the start

## 14. Watcher And Dev Server Plan

### Initial strategy
Prefer simple, reliable whole-project watching over premature incremental sophistication.

Watch at least:
- root config
- `slides/`
- `themes/`
- `archetypes/`
- deck-level assets

### Rebuild strategy
- Start with full deck rebuild on change.
- Optimize only if performance on the reference deck is unsatisfactory.

### Browser behavior
- Open the browser automatically by default.
- Support a disable flag.
- Use a fixed default port.
- Prompt if the port is unavailable.

## 15. Output Layout Plan

### Initial filesystem conventions
- `dist/html/index.html`
- `dist/html/...assets`
- `dist/pdf/<fixed-artifact-name>.pdf`

The exact PDF filename can be defined later, but it should be convention-based and not user-configurable in v1.

### Clean behavior
`margo clean` should remove:
- `dist/`
- tool-managed caches or temp artifacts

Cache location can remain internal as long as cleanup is reliable.

## 16. Milestones

## Milestone 0: Foundation

Goal:
Create a compilable CLI skeleton and internal package boundaries.

Deliverables:
- Go module setup
- command entrypoint
- config file loader
- basic diagnostics framework
- fixture directory structure for starter deck and default theme

Exit criteria:
- repository builds
- CLI help works
- config file can be located and parsed

## Milestone 1: Thin HTML Slice

Goal:
Prove the end-to-end HTML deck pipeline with a minimal theme.

Deliverables:
- slide discovery from `slides/`
- front matter parsing
- one-slide-per-bundle model
- default ordering by front matter
- simple theme contract enforcement
- default slide layout rendering
- output to `dist/html/index.html`

Exit criteria:
- one sample deck renders as navigable HTML
- theme selection from config works
- missing layout/theme failures are surfaced cleanly

## Milestone 2: Scaffolding And Archetypes

Goal:
Make first-run and day-to-day authoring real.

Deliverables:
- `margo new`
- `margo init`
- `margo new slide`
- archetype metadata support
- default starter deck
- example/reference deck alignment

Exit criteria:
- zero-config success after scaffold
- new slide generation works with explicit or prompted archetype choice

## Milestone 3: Serve Loop

Goal:
Make authoring viable.

Deliverables:
- local server
- browser auto-open
- file watching for slides, config, theme files, assets
- rebuild and refresh loop
- terminal diagnostics during `serve`

Exit criteria:
- edits to a sample slide appear quickly in browser
- draft slides appear in serve mode and are visibly marked

## Milestone 4: Theme And Content Power

Goal:
Reach the authoring surface promised by the PRD.

Deliverables:
- theme partials
- theme shortcodes
- deck-defined shortcodes
- includes
- sections
- section metadata inheritance
- visibility handling
- notes parsing and preservation
- background and image hint handling
- footer/logo overrides

Exit criteria:
- reference deck exercises all major v1 content model features
- default theme can render the branded-business-style examples expected in the PRD

## Milestone 5: PDF Export

Goal:
Add usable PDF without destabilizing HTML.

Deliverables:
- PDF export command path integrated into `build`
- serve-time on-demand PDF generation
- stable print styles and page sizing assumptions

Exit criteria:
- sample reference deck exports to usable PDF
- PDF path works offline after install assuming local browser dependency is present

## Milestone 6: Hardening

Goal:
Raise confidence for v1 release.

Deliverables:
- `margo clean`
- ignore mechanism
- improved diagnostics
- default theme polish
- reference deck coverage
- performance checks on 20-slide benchmark deck

Exit criteria:
- reference deck and starter deck both work end-to-end
- serve performance is subjectively fast on the target deck size
- primary commands behave consistently in interactive and non-interactive modes

## 17. Testing Strategy

### Test layers
1. Unit tests for parsing, config loading, ordering, and front matter normalization
2. Integration tests for project loading and rendering pipeline
3. Golden-output tests for rendered HTML fragments and full deck output
4. CLI smoke tests for scaffold/build flows
5. Manual visual QA for default theme and PDF output

### What should be tested early
- slide ordering logic
- manifest precedence
- section inheritance rules
- draft and visibility filtering
- include resolution and cycle detection
- shortcode resolution order
- theme metadata validation

### Reference deck as test asset
Use the reference deck as:
- a regression fixture
- a visual acceptance asset
- a sample project for manual review

## 18. Main Risks And Mitigations

### Risk 1: Theme contract grows too vague
Mitigation:
- keep the required contract small
- build the default theme against it immediately
- fail fast on missing theme entrypoints

### Risk 2: HTML and PDF diverge badly
Mitigation:
- keep HTML runtime simple
- avoid JS-heavy layout behavior
- add PDF only after HTML structure stabilizes

### Risk 3: Shortcodes and includes become a second programming language
Mitigation:
- keep includes simple
- keep shortcode resolution explicit
- avoid introducing parameterized content reuse in v1

### Risk 4: Scope collapses under theme ambition
Mitigation:
- one active theme only
- no theme inheritance
- no deck-local template override path as a first-class feature

### Risk 5: Serve performance disappoints
Mitigation:
- start with full rebuilds
- measure on the 20-slide reference deck
- optimize the slowest pipeline stages only after observing real bottlenecks

## 19. Implementation Priorities

If time is constrained, prioritize in this order:

1. Core deck model
2. HTML rendering pipeline
3. `serve` loop
4. Scaffolding
5. Theme contract hardening
6. Content model completeness
7. PDF export
8. Cleanup and polish

This order keeps the team building product value rather than infrastructure theater.

## 20. Recommended Immediate Next Tasks

The first implementation sprint should likely do the following:

1. Initialize the Go module and CLI entrypoint.
2. Define the internal config, theme metadata, slide bundle, and deck model structs.
3. Choose the YAML, Markdown, templating, and file-watching libraries.
4. Implement root config loading and strict project discovery.
5. Create a minimal default theme fixture with one default slide layout.
6. Implement slide bundle discovery and front matter parsing.
7. Render a minimal multi-slide HTML deck to `dist/html/index.html`.

Once that works, add `serve` before broadening the feature surface.
