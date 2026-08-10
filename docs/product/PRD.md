# Margo v1 Product Requirements Document

## 1. Overview

### Product name
Margo

### Working meaning
`margo` stands for "markdown go".

### Product summary
Margo is a local CLI tool for building beautiful slide decks from Markdown using a Hugo-inspired workflow. It is optimized for authors who are comfortable with Git, Markdown, configuration files, and code-driven theming, and who want polished branded decks without depending on browser-based editors or design tools.

Margo v1 is a slide-deck system, not a general publishing system. It uses a strict project structure, slide bundles, themes, archetypes, layouts, shortcodes, and deck-wide configuration to generate presentation artifacts. The canonical render target is HTML. PDF export is included as a practical derivative of the HTML rendering path. PPTX is a later output target and is not part of the initial release.

## 2. Goals

### Primary goal
Enable a small number of users to create polished, real-world, branded presentation decks end-to-end from Markdown with a zero-config happy path after project scaffolding.

### Secondary goals
- Deliver a Hugo-like mental model adapted for slides rather than websites.
- Make branded deck authoring simple for deck writers, even when themes are moderately sophisticated.
- Provide one strong default theme with clear customization pathways.
- Support a fast local authoring workflow with `margo serve`.
- Preserve enough semantic structure to support future editable PPTX export.

### v1 success criteria
- A user can run `margo new <deck-name>` or `margo init`, then `margo serve`, and immediately get a polished working deck without modifying theme internals.
- A branded company presentation theme can be created and used to generate polished real decks.
- A deck of roughly 20 slides with typical text and images rebuilds fast enough during `serve` to feel comfortably responsive.
- HTML output works offline when opened or served after the build is produced.
- PDF output is usable for sharing and printing, even if it is not yet a print-perfect independent rendering system.

## 3. Non-Goals For v1

These are future considerations, not permanent exclusions.

- Full presenter mode
- Speaker notes export
- Remote presentation controls
- Slide URL deep-linking
- PowerPoint import
- PPTX export
- Multi-deck repositories as a first-class workflow
- Collaborative editing
- Hosting or deployment workflows
- Browser-based authoring
- Plugin system beyond themes, shortcodes, partials, and includes
- Alternate publishing modes such as article view or web narrative mode
- Handout layouts or multi-slide-per-page printing
- Accessibility as a formal release gate
- Reproducible builds as a formal release gate

## 4. Target User

### Primary user
The initial user is technically comfortable and can work in a Hugo-like environment.

### Broader target audience
Margo v1 is primarily for:
- Engineers making code-native decks in Git
- Startup and product teams making polished internal presentations
- Teams or theme authors building branded reusable presentation systems

When there is tension in v1 design, authoring simplicity for deck writers wins over maximum theming power.

## 5. Product Principles

- Markdown-first authoring
- Hugo-inspired structure and concepts, without requiring Hugo compatibility
- Strict conventions over flexibility in v1
- Themes own visual behavior
- Decks can apply constrained theme-approved configuration
- Escape hatches exist, but the happy path should not depend on them
- Local-first, offline-capable workflow after installation
- Product-first PRD, implementation details deferred unless they affect product scope

## 6. Core User Experience

### Expected v1 flow
1. A user creates a deck with `margo new <deck-name>` or `margo init`.
2. Margo scaffolds a working deck, default theme wiring, and example archetypes/content.
3. The user authors one slide per bundle in `slides/`.
4. The user previews the deck with `margo serve`, which watches content, config, and theme files.
5. The user runs `margo build` to generate all configured outputs.
6. The user can run `margo clean` to remove generated artifacts and tool-managed build state.

### CLI shape in v1
- `margo new <deck-name>`
- `margo init`
- `margo build`
- `margo serve`
- `margo clean`
- `margo new slide`
- `margo new theme`

### CLI behavior requirements
- Interactive prompts are allowed generally in v1.
- Every interactive prompt must also have a non-interactive equivalent through flags or configuration.
- `margo serve` opens the deck preview in a browser by default.
- Browser auto-open must be configurable so it can be disabled.
- `margo serve` uses a fixed default port.
- If the default port is unavailable, `margo serve` should ask the user how to proceed rather than silently switching.
- `margo build` builds all configured outputs in one run.
- Validation is integrated into `build` and `serve`; no separate validation command is required in v1.

## 7. Canonical Project Structure

Margo v1 uses a strict, opinionated project layout.

Expected top-level structure:

```text
<deck>/
  margo.yaml
  slides/
    <slide-bundle>/
      index.md
  assets/
  themes/
  archetypes/
  layouts/
```

### Structure notes
- A repository represents a single deck in v1.
- `slides/` is required and contains one subdirectory per slide bundle.
- Each slide bundle contains `index.md` as the unit of authorship.
- Shared deck assets live outside slide bundles in a deck-level assets directory.
- `layouts/` may exist as a conventional location, but deck-local layout override is not a primary v1 customization path; layout customization should normally happen by creating or modifying a theme.
- The exact internal layout of themes, archetypes, and layouts may be refined in implementation, but the top-level conventions are explicit in v1.

## 8. Content Model

### Slide bundle model
- Each slide is a bundle.
- Each slide bundle has one `index.md`.
- Slide-local assets may live alongside `index.md`.
- One `index.md` defines exactly one slide.

### Slide ordering
- Default sequencing is controlled through slide front matter order metadata.
- A deck manifest is optional in v1.
- When present, the manifest overrides slide front matter ordering.
- The v1 manifest is narrow in scope and lists slide order only.

### Sections
- Sections are first-class in v1.
- Section divider slides may be explicit author-created slides.
- Auto-generated section divider slides are supported as a convenience.
- Explicit section slides are the primary authoring model.
- Slides may inherit section metadata and presentation hints.
- Section inheritance is limited to metadata and presentation hints, not theme/layout switching.

### Visibility and draft state
- Slide visibility is controlled in front matter.
- Slides may be marked hidden or skipped.
- Hidden or skipped slides are excluded from generated presentation outputs by default.
- Draft slides are supported at slide level only.
- `margo build` excludes drafts by default and provides an option to include them.
- When drafts are included in build outputs, they should remain visibly marked as draft in the render pipeline where practical.
- `margo serve` shows draft slides by default and visibly marks them.

### Notes
- Speaker notes are part of the core content model.
- Notes may be authored as named Markdown files under a slide bundle's `notes/` directory. Legacy front matter and body note sections remain supported as the default `Notes` bucket.
- Interactive HTML can reveal a selected note beneath the active slide when `presentation.navigation.notes` is enabled.
- Notes must survive parsing and build pipelines for future presenter/export usage.
- Notes are excluded from print HTML and PDF output by default.

### Reuse
- Author-side shared Markdown reuse is supported through explicit include-style insertion only.
- Includes are project-local only in v1.
- Themes provide reuse through layouts, partials, and shortcodes.
- There is no separate first-class shared-slide-content feature in v1.

## 9. Authoring Model

### Markdown
- Markdown is the primary content format.
- Raw HTML inside Markdown is allowed as an escape hatch.

### Shortcodes
- Shortcodes are supported in v1.
- Both theme-provided and deck-defined shortcodes are allowed.
- Deck-defined shortcodes may contain arbitrary template logic.
- The initial release assumes a trusted local-author environment rather than a hardened sandbox for shortcode execution.

### Archetypes
- Archetypes are a first-class authoring primitive in v1.
- Archetypes can prefill front matter, starter content structure, default layout selection, and content stubs.
- `margo new slide` should let the user choose an archetype when none is explicitly passed.
- A standard default archetype must exist.
- `margo new slide` places new slides at the end by default.
- `margo new slide` creates the slide bundle with `index.md`.
- Archetypes require a small machine-readable metadata file for discovery and CLI presentation.

## 10. Theme Model

### Theme philosophy
- Themes are the owner of visual behavior in v1.
- Margo v1 ships with one strong default theme.
- The default theme should support both light and dark modes, selected at deck level.
- The default theme should provide a small set of typography presets.
- The default theme should expose business-ready branding options such as logo and footer configuration.
- The default theme should include a curated vocabulary of theme-provided slide types.
- The default theme should include a small practical shortcode set, with strongest emphasis on layout/composition helpers.

### Theme customization model
- Deck-level visual customization is constrained to theme-approved options.
- Decks cannot arbitrarily override theme templates or CSS outside the theme system in v1.
- Layout/template customization is done by creating or modifying a theme rather than layering deck-local template overrides.
- A deck references exactly one active theme in v1.
- Theme composition or inheritance is out of scope for v1.
- Theme selection is explicit in root config.

### Theme implementation model
- Themes define layouts using HTML templates plus CSS, JS, reusable partials, and shortcodes.
- Theme JavaScript should remain minimal in v1 and mainly support navigation and small presentation behaviors.
- Built-in layout types such as agenda or metric remain theme conventions, not core Margo concepts.

### Theme contract
- A minimal formal theme contract is required in v1.
- The contract should require only a small set of entrypoints sufficient for scaffolding, validation, and predictable rendering.
- Themes require a small machine-readable metadata file with minimal schema such as name, version, and exposed config options.
- The theme API and conventions are allowed to evolve after v1 as the product proves itself.

### Theme packaging
- Local in-repo theme development is the primary v1 workflow.
- The design should leave room for later packaged/distributed themes.

## 11. Configuration Model

### Root deck config
- Margo v1 uses a single root deck config file.
- Root config format is YAML only.
- A deck-level schema or version field is allowed and recommended.

### Root config responsibilities
The root config controls:
- Deck metadata
- Theme selection
- Output settings
- Navigation/section settings where applicable
- Theme token or option overrides exposed by the chosen theme
- Language metadata
- Approved snippet injection

### Standardized deck metadata core
The deck metadata model should include a small standardized set of fields, including:
- title
- subtitle
- author
- date
- description
- language
- organization
- copyright

Additional theme-specific metadata may exist where needed.

### Snippet injection
- Deck-level snippet injection is supported in v1.
- Snippet injection is limited to approved document locations such as `head` and end-of-`body`.
- Snippets are authored inline in root config.
- Snippets apply in both `serve` and `build` so preview matches output.
- Snippet injection is generic and may be used for cases such as Google Analytics tracking.

## 12. Front Matter Model

### Standardized slide front matter core
Margo v1 should standardize a small common front matter core covering structural and common presentation concerns.

This core should include fields for:
- title
- order
- section
- layout or type
- notes
- draft
- visibility
- footer control
- logo suppression
- background configuration

### Front matter boundaries
- Slide front matter controls metadata, layout/type selection, notes, and a small set of presentation options.
- Slide front matter is not a general-purpose styling surface.
- Generic CSS class hooks are out of scope for v1.

### Footer overrides
- Deck-level footer configuration is supported by the default theme.
- Per-slide overrides are limited to hide footer or alternate short footer text.

### Logo overrides
- A deck-wide logo can be configured by the theme.
- A slide can suppress the logo through front matter.

## 13. Backgrounds And Images

### Backgrounds
Margo v1 standardizes a small set of background-related slide fields, including:
- background image
- background color
- overlay/tint behavior
- opacity-related controls

Themes may choose how fully specific layouts honor those settings.

### Images
- Margo should support both standardized image-capable authoring patterns and custom handling through shortcodes or raw HTML.
- A small core set of image presentation hints should be standardized.
- Those hints should cover common needs such as fit, position, and caption.
- Image control is split by scope:
  - Front matter for slide-wide imagery and treatment
  - Inline Markdown or shortcode syntax for content images within the slide body

## 14. Outputs

### Canonical output model
- HTML is the canonical rendering model in v1.
- PDF is produced from the HTML rendering path.
- PPTX is out of scope for v1, but the content model should preserve semantic structure that makes later editable export feasible.

### Output directories
Output paths are fixed by convention in v1:
- `dist/html`
- `dist/pdf`
- `dist/pptx` reserved for later

### Output artifacts
- HTML entrypoint is `dist/html/index.html`.
- Artifact naming is fixed by convention in v1.
- The build produces portable build artifacts, not hosting/deployment workflows.

### HTML requirements
- HTML output is slide-by-slide with browser navigation semantics.
- The HTML deck is the deck artifact itself, not a wrapper landing page.
- Each slide does not need its own addressable URL or route in v1.
- Output may use normal asset folders rather than a single self-contained file.
- Built output should work offline when opened or served after generation.

### PDF requirements
- PDF export is included in v1.
- PDF export should be usable for sharing and printing.
- PDF export may depend on a local Chrome/Chromium-class browser if necessary.
- PDF export should be available during `serve` on demand rather than rebuilt automatically on every change.

## 15. Development Workflow

### Serve mode
- `margo serve` is required in v1.
- It watches slide content, root config, and theme files.
- It should prioritize perceived speed over fine-grained invalidation sophistication.
- Fast whole-deck rebuilds are acceptable if they feel quick.
- Browser refresh behavior is enough; no in-browser dev overlay is required.
- Warnings stay in the terminal, not the browser preview.

### Performance expectation
- On a roughly 20-slide deck with typical text and images, `serve` rebuilds should feel reasonably fast in normal local authoring use.

## 16. Validation And Error Handling

### Strictness model
- `margo build` should fail on hard correctness issues.
- `margo build` should warn on recoverable problems.

### Hard failures in v1
- Missing layouts
- Invalid front matter
- Broken template references

### Warnings in v1
- Missing referenced assets should be warnings with degraded output rather than hard failures.

### Diagnostics
- Errors and warnings should include source file and line number where possible.
- Source reporting during parse/build is sufficient for v1; generated-output source maps are out of scope.

## 17. Assets And Dependencies

### Assets
- Bundle-local assets are first-class.
- Shared deck-level assets are supported.
- The initial release does not need theme assets to be a major author-facing concept, though themes may use them internally.

### Fonts
- Custom fonts are supported in v1.
- Font registration is controlled through themes rather than direct root-config registration.

### Offline operation
- After installation, normal authoring and output generation must work without internet access.
- The default theme and generated starter deck must avoid external network dependencies such as CDN assets or hosted fonts.

## 18. Ignore And Clean Behavior

### Ignore mechanism
- Margo v1 includes a root ignore mechanism for build/watch processing.
- The exact filename is left to implementation.
- The ignore mechanism affects build/watch behavior only.

### Clean command
- `margo clean` is included in v1.
- It removes all generated build artifacts and tool-managed cache/temp files.
- Cache location may remain an implementation detail.

## 19. Starter Experience

### `margo new` and `margo init`
- Both commands are supported in v1.
- `margo new <deck-name>` creates a new directory.
- `margo init` scaffolds into the current directory.
- `margo init` should allow initialization into an existing directory when there is no file conflict, and fail clearly on collisions.

### Scaffold contents
The scaffold should include:
- A working starter deck
- The default theme wired in
- Representative slide archetypes and examples

The scaffold should teach the intended authoring model rather than merely create empty directories.

### Starter shape
- There is one primary starter deck for v1.
- Variations should be demonstrated through archetypes and examples rather than multiple starter types.

## 20. Reference Deck

- The initial release should include a canonical reference deck.
- The reference deck serves both as an internal acceptance fixture and a user-facing example.
- It should exercise layouts, sections, code blocks, notes handling, drafts, backgrounds, images, and export-relevant behavior.

## 21. Accessibility, Security, And Platform Boundaries

### Accessibility
- Accessibility is not a formal release gate for v1.
- The default theme should still avoid obviously poor defaults where practical.

### Security model
- V1 assumes a trusted local-author environment.
- Themes, templates, shortcodes, raw HTML, and snippets are not sandboxed as a security boundary.

### Platform scope
- The full workflow may be macOS-first in v1.
- Cross-platform parity is not required as a release commitment in the initial PRD.

## 22. Future-Oriented Design Constraints

Even when not implemented in v1, the design should avoid blocking:
- Localization and multi-language support
- Presenter view and notes-aware runtime features
- PPTX export with editable structure
- Theme packaging/versioning
- External theme workspaces
- Stronger theme API stability later
- Reproducibility improvements later

## 23. Acceptance Scenario For v1

The anchor acceptance scenario for Margo v1 is:

A user creates a branded company presentation deck using the default scaffold and a theme-driven workflow, writes slides as Markdown bundles, configures a business-ready visual identity through deck-level theme options, previews the result locally with fast rebuilds, and produces a polished HTML deck plus a usable PDF without touching browser-based design tools or manually editing low-level theme internals.

## 24. Open Items To Resolve In Implementation Planning

These are intentionally deferred to implementation planning rather than blocking the PRD:
- Exact root config filename and field names
- Exact manifest filename and schema
- Exact default serve port
- Exact theme metadata schema
- Exact archetype metadata schema
- Exact default theme required files
- Exact built-in default-theme shortcode inventory
- Exact PDF generation mechanism
- Exact warning text conventions and terminal formatting
- Exact draft and visibility render markers
