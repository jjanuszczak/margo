# AGENTS.md

This file is for a new developer or agent picking up `margo`.

It is intentionally practical. Read this first so you understand:
- what the repo is
- what works today
- where the important code lives
- how to verify changes safely
- which areas are still in flux

## What This Repo Is

`margo` is a Go presentation engine and CLI with an optional Rust + Tauri 2
desktop app for authoring Markdown slide decks using a Hugo-like project model.

Current shape:
- local CLI for automation, scaffolding, builds, packaging, and publishing
- desktop companion for visual review and source editing, targeting macOS and Omarchy/Arch Linux
- HTML-first output
- scaffold/build/serve workflow
- default theme plus deck-local custom themes
- HTML, PDF, and editable PPTX output with format-specific limits
- Git-based vendored theme install workflow

This is not a finished product. It is beyond prototype scaffolding, but still actively evolving in product scope, theme contract, and export quality.

## Read These First

Start with:
- [README.md](./README.md)
- [docs/product/PRD.md](./docs/product/PRD.md)
- [docs/product/implementation-plan.md](./docs/product/implementation-plan.md)
- [docs/AUTHORING_GUIDE.md](./docs/AUTHORING_GUIDE.md)

For desktop work, also read:
- [Desktop development and use](./desktop/README.md)
- [Desktop PRD](./docs/product/desktop-app-prd.md)
- [Desktop implementation plan](./docs/product/desktop-app-implementation-plan.md)
- [Desktop runtime and protocol](./docs/architecture/desktop-runtime.md)

Historical context:
- [docs/archive/sprints](./docs/archive/sprints)

Use the product docs for intent and scope boundaries. Use the authoring guide for what the tool is supposed to feel like for end users.

## Current Product State

### Working CLI surface

- `margo new <deck-name>`
- `margo init`
- `margo build`
- `margo serve`
- `margo clean`
- `margo new slide <name>`
- `margo new note <name> --slide <bundle>`
- `margo slide insert <name> ...`
- `margo slide move <slide> ...`
- `margo slide delete <slide>`
- `margo new theme <name>`
- `margo theme add <repo> [--ref <rev>] [--name <local-name>]`
- `margo theme update <name>`
- `margo theme list`

### Implemented platform pieces

- `margo.yaml` parsing and project discovery
- YAML slide front matter
- Markdown rendering with Goldmark
- deck-local themes under `themes/<name>/`
- per-slide layouts
- slide archetypes
- draft and hidden filtering
- notes preservation without normal slide rendering
- sections and synthetic section divider slides
- deck-local shortcodes
- project-local Markdown includes
- deck-level snippet injection at approved locations
- deck and slide asset staging
- interactive HTML output
- print-oriented HTML output
- PDF generation from print HTML
- vendored theme install/list
- benchmark and regression fixtures
- release archive script

### Desktop app MVP

- Rust + Tauri 2 shell with a bundled Go engine using a private JSON-lines protocol
- project picker and recent projects; no separate installed CLI required
- rendered slide thumbnails, overview, and a preview that fits the pane on resize
- Monaco Markdown editor with auto-save, manual save, per-file undo/redo, and conflict handling
- named bundle note creation/editing and rendered notes preview
- add, duplicate, reorder, and trash slides through shared CLI operations
- local image/video insertion and HTML/PDF/PPTX export entry points
- light, dark, and system app appearance with independent editor themes

Apple Silicon macOS builds and native workflows have been checked locally. Actual
Omarchy/Hyprland runtime acceptance remains pending. CI build jobs are configured
for macOS, Ubuntu/WebKitGTK, and Arch; configuration alone is not passing evidence.

### Implemented but still rough

- PDF fidelity and environment reliability
- theme print overrides beyond the default/productized examples
- theme portability UX beyond the current repo-root install assumption
- manifest-driven sequencing beyond the current loader path

### Not done

- full PPTX shortcode/theme parity
- desktop theme/settings panels, selected-slide presentation start, export cancellation, and zoom presets
- signed/notarized desktop distribution and automatic updates
- presenter mode
- plugin architecture
- multi-deck repo workflow
- arbitrary slide sizes and ratios beyond the current defaults

## Current Priorities

There is no single active sprint doc you should treat as the source of truth.

When choosing work, check:
1. the PRD
2. the implementation plan
3. open GitHub issues
4. whether the change tightens the current model or expands product scope

As of now, the highest-risk product areas are:
- PDF quality and validation
- theme API stability
- authoring polish and CLI ergonomics
- desktop source safety, native preview geometry/focus, and Omarchy acceptance

Recent additions that matter:
- optional Tauri desktop app with Monaco, rendered thumbnails, notes preview, and appearance preferences
- theme-aware print templates
- vendored theme installation via `theme add`
- `serve --port` for non-default preview ports

## Architectural Principles

These are important project-level design choices, not incidental implementation details.

### 1. Keep the Go engine generic

Prefer Go code to provide:
- parsing
- validation
- asset resolution
- staging
- model shaping
- generic template helper primitives

Avoid pushing presentation-specific HTML/CSS shape decisions into Go unless there is a strong cross-cutting reason.

### 2. Keep rendering shape in templates

Shortcodes, layouts, and future partials should own:
- markup structure
- class composition
- inline style composition where needed
- theme-facing presentational decisions

Example:
- generic helpers like required param checks and asset resolution belong in Go
- figure-specific class/style assembly belongs in shortcode templates, not in Go helpers

### 3. Prefer reusable primitives over named feature logic

If a helper can validate or resolve something for many shortcodes or templates, it is a good candidate for Go.

If a helper only exists to assemble one component's final markup shape, it probably belongs in template composition instead.

### 4. Keep deck, theme, and shortcode concerns separate

Margo should preserve a clear boundary:
- engine mechanics in Go
- theme and deck presentation in templates
- author-facing content usage in Markdown/front matter

Do not quietly collapse those layers together.

Treat engine/theme decoupling as a first-class rule, not a stylistic preference.

- Avoid making themes depend on ad hoc engine-only template fields when the same behavior can be derived from rendered markup or stable generic helpers.
- Prefer theme behavior that keys off HTML structure, CSS classes, declared assets, or reusable template primitives rather than feature-specific booleans injected from Go.
- If the engine must expose something to templates, prefer a stable mechanism-level helper over a narrow feature flag that couples one theme behavior to one internal implementation detail.

Example:
- loading a chart runtime because the deck contains `.shortcode-chart` markup is a good theme-side decision
- requiring a theme to read a special field like `.HasCharts` is tighter coupling and should be avoided unless there is a strong cross-cutting reason

### 5. Keep the desktop app on the canonical engine

Go owns project interpretation, file revisions, safe mutations, and rendering.
Rust owns native lifecycle, dialogs, sidecar management, and preview placement.
The JS UI owns buffers and interaction state. Do not duplicate Margo parsing or
rendering in Rust/JS, or replace source files with an editor-owned document model.

The preview runs in a separate unprivileged native webview, with generated files
served from a read-only random-token loopback URL. Never give theme-authored HTML
the editor's native capabilities. Bundle the engine from the same checkout.

The content model stores named-note paths relative to the bundle. Desktop source
reads/saves require project-relative paths. Preserve that conversion in snapshots.
App/editor appearance is independent of the deck's theme. Use the approved
`assets/new-logo-*.png` wordmarks rather than creating replacement branding.

### 6. Favor explicit override models

When deck-level and theme-level customization both exist, prefer explicit local override behavior over magic lookup or deep inheritance.

This already applies to shortcodes and should remain the bias for future features such as partials.

## Repo Layout

Top-level directories:

```text
cmd/
desktop/
docs/
examples/
internal/
scripts/
skills/
```

### Agent Skills

Predefined agent skills are stored in the `skills/` directory. These skills provide specialized workflows and domain expertise for Gemini CLI when working on the `margo` project.

- **[github-issue-manager](./skills/github-issue-manager)**: Manages the creation of GitHub issues and development branches. Use this to start work on a new feature or bug fix.
- **[git-workflow-lifecycle](./skills/git-workflow-lifecycle)**: Automates the end-to-end lifecycle of a feature branch, from submission (commit, push, PR) to cleanup.

To ensure these skills are active in your environment, they should be installed into the workspace scope:
```bash
gemini skills install skills/github-issue-manager.skill --scope workspace
gemini skills install skills/git-workflow-lifecycle.skill --scope workspace
```
After installation, reload the skills in your interactive session using `/skills reload`.

### Important code areas

- [desktop/src](./desktop/src)
  Tauri web UI, Monaco adapter, app appearance, and buffer reconciliation

- [desktop/src-tauri/src/main.rs](./desktop/src-tauri/src/main.rs)
  Native shell, dialogs, bundled engine lifecycle, preview isolation and geometry

- [desktop/scripts](./desktop/scripts)
  Target-specific Go sidecar builds, approved-logo icons, and engine measurements

- [internal/desktop](./internal/desktop)
  Private app protocol, project snapshots, source saves, preview server, notes and thumbnails

- [internal/cli/desktop.go](./internal/cli/desktop.go)
  Private engine entry point and reuse of canonical CLI slide operations

- [cmd/margo/main.go](./cmd/margo/main.go)
  CLI entrypoint

- [internal/cli](./internal/cli)
  Command dispatch and user-facing CLI flows

- [internal/config](./internal/config)
  `margo.yaml` parsing and validation

- [internal/project](./internal/project)
  Deck-root discovery

- [internal/content](./internal/content)
  Slide discovery, front matter parsing, notes extraction, includes

- [internal/deck](./internal/deck)
  In-memory deck model, filtering, section building

- [internal/theme](./internal/theme)
  Theme loading, metadata, validation, install workflow

- [internal/shortcode](./internal/shortcode)
  Shortcode rendering and resolution

- [internal/output/render](./internal/output/render)
  Shared render helpers used by multiple output pipelines

- [internal/output/html](./internal/output/html)
  Interactive HTML rendering pipeline

- [internal/output/printhtml](./internal/output/printhtml)
  Static print-oriented HTML rendering pipeline

- [internal/output/pdf](./internal/output/pdf)
  PDF generation from print HTML

- [internal/scaffold](./internal/scaffold)
  Deck/theme/slide scaffolding and default theme templates

- [internal/serve](./internal/serve)
  Preview server and PDF export endpoint

- [internal/watch](./internal/watch)
  File watching for rebuild flows

- [internal/manifest](./internal/manifest)
  Manifest loading support

- [internal/ignore](./internal/ignore)
  Ignore rules for project walking/build inputs

## Example Decks And Fixtures

These matter a lot:

- [examples/reference-deck](./examples/reference-deck)
  Main regression/acceptance fixture

- [examples/benchmark-deck](./examples/benchmark-deck)
  Committed 20-slide benchmark deck

- [examples/arca-investor-memo](./examples/arca-investor-memo)
  Real dogfood deck and the main theme-aware PDF fidelity check

Use them for:
- manual QA
- regression tests
- validating author-facing behavior
- checking whether theme changes hold up on a real deck

Prefer committed fixtures and temp directories in tests over ad hoc throwaway decks.

## How To Build And Run

Build the CLI:

```bash
go build -o ./bin/margo ./cmd/margo
```

Run directly:

```bash
go run ./cmd/margo help
```

Create and use a test deck:

```bash
./bin/margo new my-deck
cd my-deck
../bin/margo build
../bin/margo serve
```

Use a non-default preview port when needed:

```bash
../bin/margo serve --port 1414
```

Install a vendored theme from Git:

```bash
../bin/margo theme add https://example.com/brand-theme.git --ref v0.1.0
../bin/margo theme list
```

Important limitation:
- `theme add` currently assumes the Git repo root is itself a single Margo theme
- it does not yet install a theme from a subdirectory inside a larger monorepo

### Desktop development

From the repository root, install Go 1.22+, Node.js 22+, stable Rust, and the
[Tauri platform prerequisites](https://v2.tauri.app/start/prerequisites/), then run:

```bash
cd desktop
npm ci
npm run dev
```

Development and packaging build the Go sidecar automatically. For a local macOS
app, run `npm run build -- --debug --bundles app` from `desktop/`; launch
`desktop/src-tauri/target/debug/bundle/macos/Margo.app` from Finder. Quit and reopen
a running instance after rebuilding. Native Arch build and portal instructions
are in the [desktop guide](./desktop/README.md).

## Testing And Verification

### Full suite

```bash
env GOCACHE=/Users/johnjanuszczak/Projects/margo/.gocache go test ./...
```

In restricted environments, set `GOCACHE` explicitly as above.

### Focused test areas

- content parsing: `./internal/content`
- themes/install/options: `./internal/theme`
- interactive rendering: `./internal/output/html`
- print rendering: `./internal/output/printhtml`
- PDF glue: `./internal/output/pdf`
- CLI workflows: `./internal/cli`
- desktop protocol, source safety, notes and previews: `./internal/desktop`

### Desktop checks

```bash
npm test --prefix desktop
npm run web:build --prefix desktop
cargo fmt --manifest-path desktop/src-tauri/Cargo.toml -- --check
```

Use a disposable copy of a committed deck for editing and structural-operation QA.
Verify source save/undo, external edits, note creation/reopening/preview, thumbnails,
app/editor themes, window/divider resizing, and exports. Chrome/Chromium supplies
thumbnail capture and PDF export. The optional real-capture test is:

```bash
MARGO_DESKTOP_THUMBNAIL_TEST=1 go test ./internal/desktop -run Thumbnail
```

Native acceptance must run on the actual OS. Linux compilation does not verify
Omarchy's Wayland geometry, input, GPU rendering, or desktop file-picker portals.

### Benchmark

```bash
go test -run '^$' -bench BenchmarkBenchmarkDeckBuildFlow ./internal/output/html
```

### Manual checks

When a change affects user-facing output, prefer checking:
- `examples/reference-deck`
- `examples/benchmark-deck`
- `examples/arca-investor-memo`
- a fresh scaffolded deck from `margo new`

## Release / Packaging

Release archive script:
- [scripts/release.sh](./scripts/release.sh)

Example:

```bash
VERSION=0.1.0 ./scripts/release.sh
```

This builds versioned archives that bundle:
- the binary
- [README.md](./README.md)
- [docs/AUTHORING_GUIDE.md](./docs/AUTHORING_GUIDE.md)

Version stamping comes from:
- [internal/version/version.go](./internal/version/version.go)

The script packages the CLI only. The Homebrew tap also installs the CLI, not the
app. Desktop builds use Tauri separately; signed/notarized app releases, automatic
updates, and `.pkg` packaging are not configured. See [release procedures](./docs/RELEASING.md).

## Common Sharp Edges

### 1. The repo root is not a deck

`margo` source itself is not a presentation project.

Do not expect `margo build` to work from repo root.

### 2. PDF has two separate concerns

There are two classes of PDF problems:
- print HTML correctness
- browser/headless environment behavior

Do not conflate them.

If a PDF looks wrong, compare:
- interactive HTML
- print HTML
- generated PDF

The print pipeline now uses a dedicated static print artifact rather than rewriting the interactive deck HTML.

### 3. Theme behavior is convention-heavy

The default theme is still the primary reference contract.

Changes to:
- layout names
- theme metadata
- option names
- slide wrapper structure
- print template behavior

will ripple into:
- scaffolding
- fixtures
- docs
- tests
- real deck fidelity

Also be careful about architectural drift:
- do not move shortcode-specific presentational logic into Go unless it is genuinely reusable
- prefer template composition over new feature-specific helper code
- when adding helper functions, ask whether they are mechanism-level or component-specific

### 4. The print and interactive pipelines are related but not identical

`internal/output/html` and `internal/output/printhtml` share render logic, but they are not the same shell.

When changing deck-level wrappers or CSS assumptions, verify both pipelines.

### 5. Keep features disciplined

The product direction avoids turning authoring into a second programming language.

Prefer:
- explicit, constrained features
- deck-local and theme-local customization
- minimal extension points with clear ownership
- generic engine primitives with presentational composition in templates

Avoid:
- broad DSLs
- implicit magic behavior
- arbitrary generalization unless the product needs it
- growing a library of shortcode-specific Go helpers when template logic is sufficient

### 6. Desktop notes and preview boundaries

Named notes live at `slides/<bundle>/notes/<name>.md`. The desktop notes reader
works independently of a theme's Notes controls; published controls still require
`presentation.navigation.notes: true` and theme support. Slide previews fit the
rendered stage at a fixed source viewport. Do not fix resizing by changing theme
breakpoints or enabling page scrollbars that overlap native controls.

## How To Work Safely

When implementing a change:

1. Identify whether it is:
   - product behavior
   - theme behavior
   - scaffold behavior
   - test-only behavior

2. Update all affected layers together:
   - implementation
   - scaffold output if applicable
   - committed fixtures if applicable
   - tests
   - docs for user-facing behavior

3. Verify with the smallest correct test scope first.

4. If output changes, verify with a real deck flow.

5. If the change touches theme rendering, check both:
   - the default/reference deck
   - the ArCa dogfood deck

6. If you add template helpers in Go, explicitly check whether they are:
   - generic mechanism helpers that could serve multiple shortcodes/layouts
   - presentational helpers that should stay in templates instead

## Documentation Conventions

Use relative Markdown links in repo docs.

Good:
- `[PRD](./docs/product/PRD.md)`
- `[Authoring Guide](./docs/AUTHORING_GUIDE.md)`
- from `docs/`: `[reference deck](../examples/reference-deck)`

Avoid:
- absolute local filesystem paths in repo Markdown

## Practical Advice

If you are unsure what to do next:
- use the committed decks
- prefer tightening the current model over inventing a new subsystem
- check whether the problem is really product behavior, theme behavior, or export behavior
- do not “fix PDF” by damaging the HTML-first experience
