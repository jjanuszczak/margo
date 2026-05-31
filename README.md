<img width="1200" height="500" alt="margo-logo" src="https://github.com/user-attachments/assets/398419fa-c516-4401-901a-a35b9e34a95b" />

# Margo

Margo is a Go prototype for building slide decks from Markdown with a Hugo-like project model.

The current repo contains:
- the CLI implementation
- the product docs in [PRD.md](./PRD.md) and [IMPLEMENTATION_PLAN.md](./IMPLEMENTATION_PLAN.md)
- the user guide in [docs/AUTHORING_GUIDE.md](./docs/AUTHORING_GUIDE.md)
- a working local deck scaffold/build/serve flow
- one committed reference fixture deck in [examples/reference-deck](./examples/reference-deck)
- one committed benchmark deck in [examples/benchmark-deck](./examples/benchmark-deck)

## Current Status

This is an early prototype, not a release-ready tool.

Working today:
- `margo new <deck-name>`
- `margo init`
- `margo build`
- `margo serve`
- `margo clean`
- `margo new slide <name>`
- `margo new theme <name>`
- `margo new theme <name> blank`

Implemented in the content model:
- YAML config parsing
- YAML slide front matter
- Markdown rendering with Goldmark
- theme loading from `themes/<name>/`
- per-slide layout selection
- draft filtering
- hidden/skip filtering
- section context
- notes preservation without rendering notes into normal slide HTML

Not finished:
- PDF export
- PPTX export
- manifest-driven sequencing beyond the current loader path
- presenter mode
- browser-refresh injection outside the scaffolded default theme
- polished CLI flags and non-interactive ergonomics

## Build

Build a local binary:

```bash
go build -o ./bin/margo ./cmd/margo
```

Run directly without installing:

```bash
go run ./cmd/margo help
```

Build versioned release archives:

```bash
VERSION=0.1.0 ./scripts/release.sh
```

Override the default targets by passing explicit `GOOS/GOARCH` pairs:

```bash
VERSION=0.1.0 ./scripts/release.sh darwin/arm64 linux/amd64
```

## Example Workflow

Create a deck:

```bash
./bin/margo new my-deck
cd my-deck
../bin/margo build
```

Serve the deck locally:

```bash
../bin/margo serve
```

Create a slide:

```bash
../bin/margo new slide roadmap
```

Create a theme:

```bash
../bin/margo new theme custom
```

## Project Shape

A generated deck currently looks like:

```text
my-deck/
  margo.yaml
  slides/
    01-title/index.md
    02-why/index.md
  themes/
    default/
      theme.yaml
      layouts/
        deck.html
        slide-default.html
        slide-title.html
  archetypes/
    default/
      archetype.yaml
  assets/
```

## Development Notes

- The repository root is the CLI source repo, not a deck project.
- Generated test decks and local caches are ignored by `.gitignore`.
- `examples/reference-deck` is the committed fixture deck for manual checks and regression work.
- `examples/benchmark-deck` is the committed 20-slide deck for manual performance and density checks.
- Some local preview verification required running outside the sandbox because binding `127.0.0.1:1313` is restricted in this environment.

Benchmark the committed 20-slide deck build path with:

```bash
go test -run '^$' -bench BenchmarkBenchmarkDeckBuildFlow ./internal/output/html
```
