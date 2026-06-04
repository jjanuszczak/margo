<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./assets/margo-logo-dark-bg-flat.png">
  <source media="(prefers-color-scheme: light)" srcset="./assets/margo-logo-light-bg-flat.png">
  <img alt="Margo logo" src="./assets/margo-logo-glow.png" width="600">
</picture>

# Margo

Margo is a Go prototype for building slide decks from Markdown with a Hugo-like project model.

The current repo contains:
- the CLI implementation
- the product docs in [docs/product/PRD.md](./docs/product/PRD.md) and [docs/product/implementation-plan.md](./docs/product/implementation-plan.md)
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
- `margo theme add <repo>`
- `margo theme list`
- `margo clean`
- `margo new slide <name>`
- `margo new theme <name>`
- `margo new theme <name> blank`
- `margo theme update <name>`

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

Homebrew packaging notes for maintainers live in [docs/HOMEBREW.md](./docs/HOMEBREW.md).

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

If the default preview port is unavailable, choose another port interactively or pass one explicitly:

```bash
../bin/margo serve --port 1414
```

Create a slide:

```bash
../bin/margo new slide roadmap
```

Create a theme:

```bash
../bin/margo new theme custom
```

Install a vendored theme from a Git repo:

```bash
../bin/margo theme add https://example.com/brand-theme.git --ref v0.1.0
../bin/margo theme list
```

Update an installed vendored theme from its recorded Git source:

```bash
../bin/margo theme update brand
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
- `margo serve` uses `127.0.0.1:1313` by default. If that port is unavailable, interactive runs now prompt for another port and non-interactive runs should use `--port <port>`.
- Some local preview verification required running outside the sandbox because binding local ports may be restricted in this environment.

Benchmark the committed 20-slide deck build path with:

```bash
go test -run '^$' -bench BenchmarkBenchmarkDeckBuildFlow ./internal/output/html
```
