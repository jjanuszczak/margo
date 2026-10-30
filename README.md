<picture>
  <source media="(prefers-color-scheme: dark)" srcset="./assets/new-logo-dark-bg.png">
  <source media="(prefers-color-scheme: light)" srcset="./assets/new-logo-light-bg.png">
  <img alt="Margo logo" src="./assets/new-logo-light-bg.png" width="600">
</picture>

Markdown presentation engine | Inspired by [Hugo](https://gohugo.io/) | Developer *and* agent friendly

This project's overview is a [slide deck](https://jjanuszczak.github.io/margo/) built with **margo**!

# Margo

Margo builds slide decks from Markdown with a Hugo-like project model. Use the Go
CLI for scaffolding, automation, builds, and publishing, or the optional Rust + Tauri
desktop app to browse slides and edit their source with a live Margo preview. Both
use the same projects, themes, and Go rendering engine.

The current repo contains:
- the Go engine and CLI implementation
- the desktop app in [desktop](./desktop), targeting macOS and Omarchy/Arch Linux
- the product docs in [docs/product/PRD.md](./docs/product/PRD.md) and [docs/product/implementation-plan.md](./docs/product/implementation-plan.md)
- the user guide in [docs/AUTHORING_GUIDE.md](./docs/AUTHORING_GUIDE.md)
- a working local deck scaffold/build/serve flow
- one committed reference fixture deck in [examples/reference-deck](./examples/reference-deck)
- one committed benchmark deck in [examples/benchmark-deck](./examples/benchmark-deck)
- an installable Omarchy palette theme and preview deck in [examples/omarchy-theme-showcase](./examples/omarchy-theme-showcase)

## Current Status

Margo is under active development. The CLI and desktop app have working local
authoring flows; the desktop debug app has launched on Omarchy, while full native
acceptance and desktop distribution remain pending.

The release-gated [Margo product deck](./examples/margo-product-deck) deploys to the repository's [GitHub Pages site](https://jjanuszczak.github.io/margo/) with each versioned release. The published site is built from the same release tag as the shipped binary. Its source is also a working deck and a reference for users and contributors.

Working today:
- `margo new <deck-name>`
- `margo init`
- `margo build`
- `margo serve`
- `margo pack <deck-dir>`
- `margo unpack <archive.margo> [destination]`
- `margo theme add <repo>`
- `margo theme pack <theme-name>`
- `margo theme import <archive.margot>`
- `margo theme list`
- `margo clean`
- `margo upgrade --plan|--apply`
- `margo skills install brand-theme --scope user|project [--plan]`
- `margo deploy github-pages`
- `margo new slide <name>`
- `margo slide insert <name> (--before <slide> | --after <slide> | --position <n>)`
- `margo slide move <slide> (--before <slide> | --after <slide> | --position <n>)`
- `margo slide delete <slide>`
- `margo new note <name> --slide <bundle>`
- `margo new theme <name>`
- `margo new theme <name> blank`
- `margo theme update <name>`
- `margo theme contract init [theme-name]`

Implemented in the content model:
- YAML config parsing
- YAML slide front matter
- Markdown rendering with Goldmark
- theme loading from `themes/<name>/`
- per-slide layout selection
- draft filtering
- hidden/skip filtering
- section context
- bundle-based Markdown notes, including speaker scripts and research, with optional interactive HTML viewing

Not finished:
- advanced PPTX theme recipes and full shortcode parity
- presenter mode
- browser-refresh injection outside the scaffolded default theme

## Desktop companion (MVP)

The app creates new projects with the Go scaffold or opens an existing folder
containing `margo.yaml`, then writes directly to its Markdown bundles. It provides:

- actual rendered slide thumbnails, an overview, and a preview that fits on resize
- Monaco source editing with syntax highlighting, Find, auto-save, manual save, and undo/redo
- named Markdown notes with a rendered notes preview
- add, duplicate, reorder, delete-to-trash, and local image/video insertion
- presentation review and HTML, PDF, and PPTX export actions
- graphical tiling options that write theme-supported nested shortcodes
- task-specific agent-skill prompts with project and source context; the app does
  not run an AI agent
- a Projects menu for New/Open, recent projects, and export, with icon toolbar actions
- light, dark, system, or active Omarchy palette appearance plus independent editor themes
- external-change detection and conflict handling when CLI, agents, or other editors change files

Build tools: Go 1.22+, Node.js 22+, stable Rust, and
[Tauri platform prerequisites](https://v2.tauri.app/start/prerequisites/).
From the repository root:

```bash
cd desktop
npm ci
npm run dev
```

For a local macOS app:

```bash
npm run build -- --debug --bundles app
```

Open `desktop/src-tauri/target/debug/bundle/macos/Margo.app` from Finder. Quit and
reopen it after rebuilding. The app bundles its Go engine; a separate installed CLI
is unnecessary. Chrome/Chromium supplies thumbnails and PDF export.

Apple Silicon macOS has local build and native-workflow verification. A debug build
has launched in an Omarchy session; broader native acceptance of palette matching,
dialogs, Wayland geometry/focus, and exports remains pending. Desktop signing/notarization
and automatic updates are not configured. CLI release archives and Homebrew
installation do not install the app.

See the [desktop guide](./desktop/README.md) for Arch prerequisites, packaging,
notes, recovery, and limits; the [runtime decision](./docs/architecture/desktop-runtime.md)
for architecture; and the [desktop PRD](./docs/product/desktop-app-prd.md) for scope.

## CLI build and installation

Build a local binary:

```bash
go build -o ./bin/margo ./cmd/margo
```

Run directly without installing:

```bash
go run ./cmd/margo help
```

Install the latest tagged version with Go:

```bash
go install github.com/jjanuszczak/margo/cmd/margo@latest
```

For prebuilt binaries, download the archive for your platform from the
[GitHub Releases](https://github.com/jjanuszczak/margo/releases) page and
verify it with the published `checksums.txt` file. macOS and Linux users can
also install through the personal Homebrew tap:

```bash
brew install jjanuszczak/margo/margo
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

Maintainer release procedures are documented in
[docs/RELEASING.md](./docs/RELEASING.md).

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

Insert a slide into an existing deck:

```bash
../bin/margo slide insert roadmap --after 02-why --archetype agenda
```

When every existing bundle uses a positional name such as `01-title`, Margo
renumbers later bundles automatically. For decks with descriptive bundle names,
it preserves those paths and only updates slide order. Add `--renumber` when
you explicitly want to convert those bundle paths to positional names.

Move or remove a slide with the same sequencing rules:

```bash
../bin/margo slide move 12-market-size --after 04-product
../bin/margo slide delete 04-product
```

Deletion moves the bundle into the deck-local `.margo-trash/` directory so its
content, notes, and assets can be recovered.

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

Upgrade Margo-managed agent guidance in an older deck without replacing
authored slides, configuration, assets, or themes:

```bash
margo upgrade --plan
margo upgrade --apply
```

Configure GitHub Pages deployment for a deck already stored in a Git
repository. The generated workflow deploys when a `v*` tag is pushed and can
also run on demand from GitHub Actions:

```bash
margo deploy github-pages --margo-version v0.3.0
```

New deck scaffolds include repository-local agent guidance for deck authoring,
theme work, tiled-content adoption, brand-theme creation, GitHub Pages deployment,
and error triage. Run `margo upgrade --plan` on an older deck to preview missing
guidance, then `margo upgrade --apply` to add it. Ask the deck's agent to use
`margo-theme-tiling` to add tiled-content support to its active theme.
Install the complete user-global brand-theme workflow with `margo skills install
brand-theme --scope user`.

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

- The repository root contains the engine, CLI, and desktop app; it is not a deck project.
- Generated test decks and local caches are ignored by `.gitignore`.
- `examples/reference-deck` is the committed fixture deck for manual checks and regression work.
- `examples/benchmark-deck` is the committed 20-slide deck for manual performance and density checks.
- `margo serve` uses `127.0.0.1:1313` by default. If that port is unavailable, interactive runs now prompt for another port and non-interactive runs should use `--port <port>`.
- Some local preview verification required running outside the sandbox because binding local ports may be restricted in this environment.

Desktop checks are `npm test --prefix desktop` and `npm run web:build --prefix desktop`.
The Go suite covers the private desktop protocol as well as the engine and CLI.
Use a disposable fixture copy for app editing tests. See [AGENTS.md](./AGENTS.md)
for code ownership and verification guidance.

Benchmark the committed 20-slide deck build path with:

```bash
go test -run '^$' -bench BenchmarkBenchmarkDeckBuildFlow ./internal/output/html
```
