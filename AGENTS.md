# AGENTS.md

This file is for a new developer or agent picking up `margo`.

It is intentionally practical. Read this first so you understand:
- what the repo is
- what works today
- where the important code lives
- how to verify changes safely
- which areas are still in flux

## What This Repo Is

`margo` is a Go CLI for building slide decks from Markdown with a Hugo-like project model.

Current shape:
- local CLI tool
- HTML-first output
- scaffold/build/serve workflow
- default theme plus deck-local custom themes
- initial PDF pipeline via print-oriented HTML
- Git-based vendored theme install workflow

This is not a finished product. It is beyond prototype scaffolding, but still actively evolving in product scope, theme contract, and export quality.

## Read These First

Start with:
- [README.md](./README.md)
- [docs/product/PRD.md](./docs/product/PRD.md)
- [docs/product/implementation-plan.md](./docs/product/implementation-plan.md)
- [docs/AUTHORING_GUIDE.md](./docs/AUTHORING_GUIDE.md)

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
- `margo new theme <name>`
- `margo theme add <repo> [--ref <rev>] [--name <local-name>]`
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

### Implemented but still rough

- PDF fidelity and environment reliability
- theme print overrides beyond the default/productized examples
- theme portability UX beyond the current repo-root install assumption
- manifest-driven sequencing beyond the current loader path

### Not done

- PPTX export
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

Recent additions that matter:
- theme-aware print templates
- vendored theme installation via `theme add`
- `serve --port` for non-default preview ports

## Repo Layout

Top-level directories:

```text
cmd/
docs/
examples/
internal/
scripts/
```

### Important code areas

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

Current caveat:
- archive creation works
- Homebrew formula and `.pkg` packaging are not implemented in-repo

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

### 4. The print and interactive pipelines are related but not identical

`internal/output/html` and `internal/output/printhtml` share render logic, but they are not the same shell.

When changing deck-level wrappers or CSS assumptions, verify both pipelines.

### 5. Keep features disciplined

The product direction avoids turning authoring into a second programming language.

Prefer:
- explicit, constrained features
- deck-local and theme-local customization
- minimal extension points with clear ownership

Avoid:
- broad DSLs
- implicit magic behavior
- arbitrary generalization unless the product needs it

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
