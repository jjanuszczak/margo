# AGENTS.md

This file is for a new agent or developer picking up `margo` for the first time.

It is intentionally practical. Read this before making changes so you understand:
- what this repo is
- what is already implemented
- how to run and verify it
- where the important code lives
- what work is still in flight

## What This Repo Is

`margo` is a Go prototype for building slide decks from Markdown with a Hugo-like project model.

It is currently:
- a local CLI tool
- HTML-first
- able to scaffold, build, and serve decks
- able to render a meaningful default theme
- partially along the path to release packaging

It is not yet a finished product. The repo is past the initial prototype stage, but still in active product-definition-through-implementation mode.

## Read These First

Start with these files:
- [README.md](./README.md)
- [PRD.md](./PRD.md)
- [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md)
- [SPRINT_02.md](./SPRINT_02.md)
- [docs/AUTHORING_GUIDE.md](./docs/AUTHORING_GUIDE.md)

These explain:
- the product intent
- the implementation shape
- the current sprint
- the supported user-facing workflow

## Current Product State

### Working CLI surface

- `margo new <deck-name>`
- `margo init`
- `margo build`
- `margo serve`
- `margo clean`
- `margo new slide <name>`
- `margo new theme <name>`

### Implemented content/platform pieces

- YAML config parsing
- YAML slide front matter
- deck/project discovery via `margo.yaml`
- theme loading from `themes/<name>/`
- per-slide layouts
- slide archetypes
- draft filtering
- hidden/skip filtering
- notes preservation
- sections and synthetic section divider slides
- deck-local shortcodes
- project-local include-style Markdown reuse
- deck-level snippet injection at approved locations
- bundle-local and deck-level asset staging
- HTML output generation
- initial PDF path
- benchmark and fixture coverage
- release archive script

### Not done / not mature

- robust PDF validation across environments
- PPTX export
- presenter mode
- plugin architecture
- multi-deck repo workflow
- final theme/API stability

## Current Sprint

The active sprint is [SPRINT_02.md](./SPRINT_02.md).

As of now, most P0 work is effectively implemented:
- deck-local shortcodes
- project-local includes
- deck-level snippet injection
- theme option ergonomics
- stronger theme contract validation

The main remaining Sprint 02 work is P1:
- starter-deck polish
- a real dogfood deck

Before starting a new feature, compare it to the sprint and decide whether it belongs in:
- the current sprint
- a follow-up sprint
- explicit future work outside sprint scope

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
  command dispatch and user-facing CLI flows

- [internal/config](./internal/config)
  `margo.yaml` parsing

- [internal/project](./internal/project)
  deck-root discovery

- [internal/content](./internal/content)
  slide discovery, front matter parsing, notes extraction, include resolution

- [internal/theme](./internal/theme)
  theme metadata loading and option validation

- [internal/shortcode](./internal/shortcode)
  shortcode rendering and resolution

- [internal/output/html](./internal/output/html)
  main HTML rendering pipeline, asset staging, fixture/benchmark tests

- [internal/output/pdf](./internal/output/pdf)
  HTML-to-PDF path

- [internal/scaffold](./internal/scaffold)
  deck/theme/slide scaffolding

- [internal/serve](./internal/serve)
  preview server

## Fixtures and Example Decks

These matter a lot:

- [examples/reference-deck](./examples/reference-deck)
  the main committed regression/acceptance fixture

- [examples/benchmark-deck](./examples/benchmark-deck)
  the committed 20-slide benchmark deck for performance checks

Use them for:
- manual QA
- regression tests
- validating author-facing behavior

Do not create a pile of throwaway deck folders again unless there is a strong reason. Prefer:
- committed fixtures
- temp directories in tests

## How to Build and Run

Build the CLI:

```bash
go build -o ./bin/margo ./cmd/margo
```

Create and use a test deck:

```bash
./bin/margo new my-deck
cd my-deck
../bin/margo build
../bin/margo serve
```

If you need the repo’s committed examples:

```bash
cd examples/reference-deck
go run ../../cmd/margo build
```

## Testing and Verification

### Full suite

```bash
env GOCACHE=/Users/johnjanuszczak/Projects/margo/.gocache go test ./...
```

The repo often needs a local `GOCACHE` in restricted environments. If Go tries to use a system cache path and fails, set `GOCACHE` explicitly.

### Benchmark

```bash
go test -run '^$' -bench BenchmarkBenchmarkDeckBuildFlow ./internal/output/html
```

This uses the committed 20-slide benchmark deck.

### Focused test areas

If you are working on:
- content parsing: `./internal/content`
- themes/options: `./internal/theme`
- rendering: `./internal/output/html`
- CLI workflows: `./internal/cli`

### Manual checks

When a change affects user-facing output, prefer checking:
- `examples/reference-deck`
- `examples/benchmark-deck`
- a fresh scaffolded deck from `margo new`

## Release / Packaging

There is now a release build script:
- [scripts/release.sh](./scripts/release.sh)

Example:

```bash
VERSION=0.1.0 ./scripts/release.sh
```

This builds versioned archives for common targets and bundles:
- the binary
- [README.md](./README.md)
- [docs/AUTHORING_GUIDE.md](./docs/AUTHORING_GUIDE.md)

Version stamping comes from:
- [internal/version/version.go](./internal/version/version.go)

Current caveat:
- archive creation works
- Homebrew formula and `.pkg` packaging are not yet implemented in-repo

## Documentation Conventions

Use relative Markdown links in repo docs.

Good:
- `[PRD](./PRD.md)`
- `[Authoring Guide](./docs/AUTHORING_GUIDE.md)`
- from `docs/`: `[reference deck](../examples/reference-deck)`

Avoid:
- absolute local filesystem paths in repo Markdown

## Common Sharp Edges

### 1. PDF may fail in restricted environments

The PDF path currently depends on launching Chrome/Chromium headlessly. In remote or sandboxed environments this may fail even when HTML is fine.

Treat PDF failures carefully:
- distinguish real renderer bugs from environment/browser issues
- do not break HTML trying to “fix” PDF blindly

### 2. `margo` source repo is not a deck

The repo root is the CLI source tree, not a presentation deck.

Do not assume `margo build` should work from repo root.

### 3. Theme behavior is partly convention-driven

The default theme is the main real product surface right now. Be careful when changing:
- layout names
- theme metadata contract
- option names
- shortcode behavior

Changes there ripple into:
- scaffolding
- fixtures
- docs
- tests

### 4. Keep features disciplined

The product direction deliberately avoids turning authoring into a second programming language.

Prefer:
- explicit, constrained features
- simple include behavior
- theme-owned customization

Avoid:
- broad new DSLs
- arbitrary extension points unless clearly justified

## How to Work Safely

When implementing a change:

1. identify whether it is:
   - product behavior
   - theme behavior
   - scaffold behavior
   - regression/test-only behavior

2. update all affected layers together:
   - implementation
   - scaffold output
   - reference fixture if needed
   - tests
   - docs if user-facing

3. verify with the smallest correct test scope first

4. verify with a real deck flow if output changes

## Recommended Next Work

If you are picking up immediately after the current state, the most obvious next items are:
- finish Sprint 02 P1 starter-deck polish
- create a real dogfood deck
- then reassess whether the next sprint should focus on:
  - export quality
  - product polish

## Final Advice

This repo has enough history now that it is easy to accidentally solve the wrong problem.

When in doubt:
- check the PRD
- check the sprint
- use the committed decks
- prefer tightening the existing model over inventing a new one
