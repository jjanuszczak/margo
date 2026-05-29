# Sprint 01

## Goal

Establish the minimum foundation needed to start building `margo` as a real product:
- a concrete sprint backlog
- initial Go project structure
- CLI entrypoint
- project discovery
- config loading foundation
- diagnostics primitives

This sprint does not aim to render slides yet. It aims to remove ambiguity and make Milestone 1 implementation straightforward.

## Sprint Outcome

At the end of this sprint, the repository should contain:
- a coherent package layout
- a compilable CLI skeleton
- strict project/config discovery rules
- a first-pass diagnostics model
- documented next tasks for the thin HTML slice

## Constraints

- Keep dependencies at zero or near-zero until the thin skeleton is in place.
- Do not overbuild command frameworks or plugin abstractions.
- Do not start PDF work.
- Do not start generalized theme inheritance or deck-local template override systems.

## Backlog

### P0: Repository foundation
- Add `go.mod` with an initial module path placeholder.
- Create `cmd/margo` entrypoint.
- Create initial internal package structure for CLI, config, project, diagnostics, and versioning.
- Establish a stable error/exit model for command failures.

Acceptance criteria:
- The codebase is ready to compile once `go` is available.
- There is a single clear CLI entrypoint.

### P0: CLI skeleton
- Implement root command help output.
- Implement command dispatch for `build`, `serve`, `new`, `init`, `clean`, `new slide`, and `new theme`.
- Return explicit "not implemented" errors for commands not started yet.
- Add a `version` command for basic sanity and future release plumbing.

Acceptance criteria:
- Command surface matches the PRD at a structural level.
- Unimplemented commands fail clearly rather than silently doing nothing.

### P0: Project discovery
- Define the canonical root config filename as `margo.yaml`.
- Implement discovery of `margo.yaml` from the working directory.
- Add strict failure if config is missing for commands that require a project.
- Keep path resolution logic isolated from command code.

Acceptance criteria:
- `build` and `serve` can locate a deck root through project discovery.
- Discovery errors are structured and user-readable.

### P0: Raw config loading
- Implement raw root-config loading.
- Separate file discovery from parsing.
- Leave YAML schema parsing as the next slice once the dependency choice is locked.

Acceptance criteria:
- The CLI can locate and read `margo.yaml`.
- Config bytes and path are available to downstream services.

### P0: Diagnostics foundation
- Create a `Diagnostic` type with severity, code, message, path, and line.
- Create a report/collection type.
- Add helpers for formatting diagnostics to terminal output.

Acceptance criteria:
- Early project discovery and config-loading failures use the diagnostics model rather than ad hoc strings where practical.

### P1: Thin HTML slice preparation
- Define first-pass domain structs for project config, slide bundle, and deck model.
- Stub the renderer and output packages so Milestone 1 has obvious insertion points.
- Document recommended libraries for YAML, Markdown, file watching, and syntax highlighting.

Acceptance criteria:
- The next sprint can begin rendering without package churn.

## Out Of Scope For Sprint 01

- Markdown rendering
- theme template execution
- file watching
- browser auto-open
- PDF generation
- archetype scaffolding behavior

## Recommended Libraries To Confirm Next

- YAML: `gopkg.in/yaml.v3`
- Markdown: `github.com/yuin/goldmark`
- Syntax highlighting: likely `github.com/alecthomas/chroma` via Goldmark integration
- File watching: `github.com/fsnotify/fsnotify`

These are recommendations, not yet integrated in this sprint.

## Exit Checklist

- [x] `go.mod` exists
- [x] `cmd/margo/main.go` exists
- [x] root CLI dispatch exists
- [x] project discovery exists
- [x] raw config loading exists
- [x] diagnostics primitives exist
- [x] implementation notes for the thin HTML slice are preserved

## Next Sprint Preview

The next sprint should target the first end-to-end HTML output:
- parse `margo.yaml`
- discover slide bundles
- parse front matter
- define minimal theme metadata contract
- render a basic deck to `dist/html/index.html`
