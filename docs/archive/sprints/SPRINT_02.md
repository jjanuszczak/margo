# Sprint 02

## Goal

Make `margo` genuinely usable for building real HTML-first decks without touching `margo` source code.

Sprint 01 established the foundation. The repo has since moved well beyond that baseline:
- CLI scaffold/build/serve loop exists
- deck and theme scaffolding exist
- HTML rendering exists
- sections, drafts, visibility, notes, assets, shortcodes, and layouts exist
- benchmark and fixture coverage exist
- default theme is credible enough to use as a product surface

Sprint 02 should focus on authoring completeness and closing the biggest remaining PRD gaps for HTML-first deck creation.

## Sprint Outcome

At the end of this sprint, a deck author should be able to:
- create a deck
- create slides with built-in archetypes
- define deck-local shortcodes
- reuse project-local Markdown content with includes
- inject approved snippets into the deck shell
- configure the active theme with clearer validation and guidance
- build and serve a polished deck without editing `margo` internals

## Constraints

- Keep the sprint focused on authoring completeness, not platform expansion.
- Do not widen the output surface beyond the current HTML/PDF state.
- Do not start PPTX work.
- Do not start presenter mode.
- Do not introduce plugin systems or remote theme packaging.
- Prefer improving the existing model over inventing a second authoring language.

## Backlog

### P0: Deck-defined shortcodes

- Make deck-local shortcodes a first-class rendering source alongside theme shortcodes.
- Define resolution order clearly when a shortcode name exists in both deck and theme scope.
- Ensure deck-local shortcodes work in scaffolded decks without source modifications.
- Add tests that prove deck-local shortcodes render in a real deck flow.

Acceptance criteria:
- A deck author can create `shortcodes/<name>.html` in a deck and use it in slide Markdown.
- Theme-provided shortcodes still work.
- Resolution behavior is deterministic and documented.

### P0: Project-local Markdown includes

- Implement explicit include-style Markdown reuse for project-local files only.
- Support include use inside slide Markdown bodies.
- Reject or warn on invalid/missing includes cleanly.
- Prevent recursive include loops.

Acceptance criteria:
- A deck author can include shared Markdown fragments from local project files.
- Missing include targets produce clear diagnostics.
- Include cycles are detected and reported.
- Included content participates in the normal Markdown and shortcode pipeline.

### P0: Deck-level snippet injection

- Implement root-config snippet injection for approved insertion points only.
- Support at least:
  - document `head`
  - end of `body`
- Ensure snippet behavior is consistent in `build` and `serve`.
- Keep snippet configuration deck-level only.

Acceptance criteria:
- A deck author can add inline snippets in `margo.yaml`.
- Snippets appear in both served and built HTML.
- Unsupported insertion points fail clearly.

### P0: Theme option ergonomics

- Strengthen validation for theme-owned deck options.
- Improve diagnostics for:
  - unknown options
  - wrong types
  - invalid values when the theme declares an allowed surface
- Make theme metadata more useful as a user contract.
- Update docs to show the intended config model clearly.

Acceptance criteria:
- Invalid theme option input fails with actionable diagnostics.
- Default theme options are easy to discover and use.
- The authoring guide reflects the actual supported options.

### P1: Theme contract validation

- Improve failure handling for malformed or incomplete themes.
- Validate expected theme entrypoints and layout references more explicitly.
- Surface clearer errors for missing layouts and malformed metadata.
- Add regression tests around invalid theme shapes.

Acceptance criteria:
- Broken themes fail fast with path-aware diagnostics.
- The default theme and scaffolded themes still pass end-to-end.

### P1: Starter-deck polish

- Improve the scaffolded starter deck so it demonstrates the best current authoring patterns.
- Ensure it showcases:
  - logo support
  - sections
  - shortcodes
  - media-oriented layouts
  - shared assets
- Keep the starter deck simple enough to understand quickly.

Acceptance criteria:
- A fresh `margo new` deck is a good teaching asset, not just a smoke test.
- Starter-deck HTML output reads as polished and representative.

### P1: Real dogfood deck

- Create a real presentation in-repo using only deck files, not code comments or implementation notes.
- Use it as a manual acceptance artifact for the sprint.
- Ensure it exercises the sprint features in a realistic narrative flow.

Acceptance criteria:
- There is one committed dogfood deck beyond the benchmark/reference fixtures.
- The deck can be built and served end to end.

## Out Of Scope For Sprint 02

- deeper PDF fidelity work
- PPTX export
- presenter mode
- multi-deck repository support
- plugin architecture
- theme packaging or remote installation
- collaboration/browser-based authoring

## Exit Checklist

- [ ] deck-local shortcodes are implemented and tested
- [ ] project-local Markdown includes are implemented and tested
- [ ] snippet injection is implemented and tested
- [ ] theme option validation is improved and documented
- [ ] theme contract validation is stronger
- [ ] starter deck better demonstrates the product
- [ ] one real dogfood deck exists and builds

## Suggested Test Coverage

- unit tests for include resolution and cycle detection
- unit tests for shortcode resolution precedence
- unit tests for snippet config parsing and placement
- fixture-flow tests for deck-local shortcode rendering
- fixture-flow tests for includes and snippet injection
- CLI integration test for a fresh deck using the new features

## Exit Criteria

Sprint 02 is complete when:
- a new deck can be created and customized using deck files alone
- authors can extend content with local shortcodes and includes
- deck-level snippet injection works in the approved locations
- theme options and theme failures produce strong diagnostics
- the starter deck and dogfood deck both work as believable examples

## Next Sprint Preview

If Sprint 02 lands cleanly, the next sprint should likely choose one of two directions:

1. Export quality
- deeper PDF validation and quality work
- possibly early PPTX planning, but not implementation unless HTML semantics are stable

2. Product finish
- more default-theme refinement
- broader layout vocabulary
- stronger documentation and release-style packaging
