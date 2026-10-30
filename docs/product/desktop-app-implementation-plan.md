# Margo Desktop App Initial Implementation Plan

## Current implementation status

The implementation uses Rust + Tauri 2, a bundled Go sidecar with protocol v1 over
JSON lines, and Monaco. It targets macOS and Omarchy/Arch Linux. The runtime choice
is resolved by the implementation direction for issue #72; a competing Go-hosted
prototype was not built. See the [runtime decision](../architecture/desktop-runtime.md)
for the actual protocol, isolation boundary, measurements, and native checks.

Working MVP slices include project creation through the canonical scaffold, optional
Git theme import, project lifecycle, snapshots, actual rendered thumbnails, fit-to-pane
preview, source editing and undo, notes creation and preview, safe slide operations,
media import, external-change reconciliation, overview, presentation review, export
entry points, a theme-gated graphical tiling shortcode editor, and contextual
agent-skill prompt handoff. The native Projects menu and icon toolbar expose project,
export, and view actions. App/editor appearance supports light/dark/system and the
active Omarchy palette.

The slices below retain the delivery and acceptance checklist. They are not all
accepted: full Omarchy palette and Wayland acceptance, startup/paint and memory
measurements, full export fidelity, packaging/signing, and updates remain open.
Deck/theme settings panels, selected-slide presentation start, zoom presets, and
export cancellation remain follow-up work.
The [desktop guide](../../desktop/README.md) provides current commands and limits.

## 1. Purpose and delivery rule

This plan breaks the companion desktop app into small, reviewable slices with explicit dependencies and ownership boundaries. It is an initial plan, not a commitment to ship every listed capability in one release.

The desktop UI must call Margo's engine for project interpretation and rendering. Do not begin by implementing a second Markdown/front-matter/theme renderer in the app. Keep the Go CLI and publishing workflow usable throughout.

Product requirements live in [desktop-app-prd.md](./desktop-app-prd.md). The main engine roadmap remains [implementation-plan.md](./implementation-plan.md).

## 2. Delivery strategy

Build a thin vertical slice first:

`open project` -> `read ordered slide model` -> `render through Margo` -> `show slide list + preview + Markdown` -> `save one source edit` -> `refresh preview`

Then add safe slide operations, notes/assets, external-change reconciliation, overview/presentation mode, and export invocation. Validate performance and the desktop runtime boundary before investing in broad UI polish.

Suggested work lanes after the shared interface contracts stabilize:

- Engine contract: Go project snapshot, render preview, diagnostics, and mutation operations.
- Desktop shell: native window, project lifecycle, navigation, and process/service integration.
- Editor experience: Markdown editing, preview, notes, overview, keyboard behavior.
- Quality and packaging: fixture matrix, performance, app packaging, and documentation.

These lanes can proceed in parallel only after the contract slices below have defined stable request/response schemas and filesystem behavior.

## 3. Architecture boundary

### Implemented component boundary

- **Go engine:** canonical project loading, source parsing, validation, render, and safe project operations. Reuse internal packages where possible; avoid UI-specific logic in engine packages.
- **Desktop shell:** Rust + Tauri owns native window lifecycle, dialogs, bundled sidecar lifecycle, preview placement, and IPC. The UI polls for external project changes while idle.
- **Web UI:** presentation/editor panes and interaction state. It consumes a versioned app protocol and never infers Margo semantics from rendered HTML when a canonical model is available.
- **Preview surface:** displays HTML produced by the Margo renderer in a constrained local context, with project assets served from the selected project/build workspace.

### Selected integration

A long-lived bundled Go process runs the private `desktop-engine` command using
versioned JSON-lines requests and replies on stdin/stdout. The app consumes machine
responses, not human CLI output. Rust launches it with an argument array. HTTP is
reserved for read-only generated artifacts on loopback beneath a random session
prefix; it exposes no source-edit or mutation endpoints. Every operation is scoped
to the selected project. See [protocol v1](../architecture/desktop-runtime.md#protocol-v1).

### Protocol principles

- Version every request/response envelope and event.
- Include project identity and revision/generation in snapshots and render results.
- Make save/mutation operations explicit and atomic where practical.
- Return structured diagnostics with severity, message, source path, line/column when available, and a stable code.
- Emit file-change events or provide a clear refresh mechanism.
- Preserve unknown metadata and make lossless source edits a Go-side responsibility.
- Never trust paths supplied by the UI without resolving and validating them under the open project root.

## 4. Dependency-ordered implementation slices

Each slice should end with a reviewable artifact and acceptance evidence. “Tests” below are implementation verification tasks, not an instruction to run an unrelated repository-wide suite for every slice.

### Slice 0: Product/technical spike and decision record

**Goal:** Select the fastest maintainable desktop architecture for the Hype-style workspace: a persistent slide selector on the left, rendered current-slide preview above, Markdown editor below, and a resizable preview/editor divider.

**Decision:** Rust + Tauri 2 with a bundled Go engine, targeting macOS and
Omarchy/Arch Linux. The implementation direction superseded the proposed two-runtime
prototype comparison. No comparative performance claim is made.

**Evidence:** The [runtime decision](../architecture/desktop-runtime.md) records
engine timings, preview isolation, platform/build dependencies, and macOS native
checks. It also identifies missing native cold-start, paint, memory, and Omarchy
acceptance. Complete those measurements before declaring platform acceptance.

### Slice 1: Define the app protocol and stable deck snapshot

**Goal:** Give UI work and engine work a shared, small contract.

**Work:** Specify protocol version, project identity, ordered slide summary, source paths, front matter fields needed for navigation, draft/hidden status, notes references, theme identity, operation capabilities, revision IDs, and structured diagnostics. Add JSON fixtures from representative projects.

**Dependencies:** Slice 0 choice of process boundary.

**Acceptance:** Schema and fixture examples are reviewed; contract says which fields are authoritative and how unsupported values are preserved. UI and engine developers can implement against the same fixtures.

**Parallelism:** Schema and fixture authoring can split by concern, with one owner resolving names and compatibility rules.

### Slice 2: Engine read-only project snapshot and diagnostics API

**Goal:** Load a project through Go and return the deck outline without UI parsing.

**Work:** Add the selected protocol endpoint/command to discover a project root, load its canonical model, return slide ordering, labels, state flags, notes metadata, active theme, and parser/validation diagnostics. Include source revision information.

**Dependencies:** Slice 1 contract.

**Acceptance:** Reference, benchmark, and ArCa projects produce stable snapshots; malformed projects return actionable structured diagnostics; no project files are changed.

**Parallelism:** Engine implementation and UI fixture-backed navigation can proceed in parallel after contract freeze.

### Slice 3: Desktop shell and project lifecycle

**Goal:** Open and switch among local projects safely.

**Work:** Create the production desktop workspace using the runtime selected in Slice 0; implement the app window, project picker, recent projects, project-root validation, engine process/service lifecycle, loading/error states, and app logs.

**Dependencies:** Slice 0 ADR and Slice 1 protocol; may use fixture transport until Slice 2 is ready.

**Acceptance:** User can open a valid project, see project identity and load errors, close/switch projects cleanly, and the app shuts down child processes/services without leaving orphaned listeners.

**Parallelism:** Shell scaffolding and UI design system can proceed in parallel; both consume the protocol fixtures.

### Slice 4: Read-only slide browser and full-deck overview

**Goal:** Make deck structure inspectable.

**Work:** Slide sidebar with selection, ordered labels, draft/hidden markers, keyboard navigation, and an overview grid with selection. Preserve selected slide identity through snapshot updates.

**Dependencies:** Slices 2 and 3; protocol fixtures from Slice 1.

**Acceptance:** 20-slide benchmark and larger decks remain responsive; selection stays stable across refreshed snapshots; hidden/draft states are clear; keyboard shortcuts do not interfere with text entry.

**Parallelism:** Sidebar and overview UI can be built separately once selection state ownership is agreed.

### Slice 5: Margo-backed preview rendering

**Goal:** Display the actual Margo-rendered result for the selected slide and deck.

**Work:** Add a preview request/result contract with render generation, HTML/asset location or safe preview URL, diagnostics, and stale state. Embed the preview surface; handle navigation and local assets safely. Support responsive preview sizing.

**Dependencies:** Slice 0 preview security decision; Slice 1 protocol; Slices 2 and 3.

**Acceptance:** Default theme, custom theme, shortcode, image, draft, and unsupported/error fixtures preview through Margo output; failed renders cannot be mistaken for current output; project-authored theme JS does not gain access to desktop IPC.

**Parallelism:** Go rendering endpoint and UI preview frame can proceed independently against a stable preview fixture contract.

### Slice 6: Markdown editor and one-file save/reload loop

**Goal:** Edit the selected slide's body and see the saved result.

**Work:** Add source loading/saving, Markdown editor, dirty/saved state, atomic save behavior, debounced preview requests, build/render diagnostics, and external file watcher with clean-buffer reload.

**Dependencies:** Slices 2, 3, and 5; explicit source read/write contract.

**Acceptance:** Editing one slide writes only its intended file, updates preview from the same Go engine, preserves front matter/unknown metadata, and reports errors without clearing the last valid preview. External changes update a clean buffer.

**Parallelism:** Editor pane and engine save/render endpoints may proceed in parallel after save semantics are fixed.

### Slice 7: Conflict-safe editing, undo/redo, and source navigation

**Goal:** Make the app safe alongside agents, editors, and CLI operations.

**Work:** Detect dirty-buffer external changes using file hashes/revisions; present reload/keep/compare choices; prevent silent overwrites; add session undo/redo for app edits; map diagnostics to source locations.

**Dependencies:** Slice 6 save/reload loop and revision protocol.

**Acceptance:** Simulated agent edits during dirty and clean buffers never disappear; overwrite requires an explicit action; diagnostics open the correct source and location where available.

**Parallelism:** Conflict resolution and diagnostics navigation are separable once revision and diagnostic schemas are stable.

### Slice 8: Safe slide authoring and ordering operations

**Goal:** Provide Hype-like overview organization using Margo bundle semantics.

**Work:** Add create, duplicate, move, and delete operations backed by Go project operations or a versioned CLI API. Preserve assets and notes. Respect `.margo-trash` recoverability and manifest/front-matter ordering rules. Refresh selected identity and preview after changes.

**Dependencies:** Slices 2, 4, 6, and stable mutation API; reuse existing CLI operations where appropriate.

**Acceptance:** Operations produce the same valid project state as CLI equivalents; asset/note preservation is verified; deletion behavior is clear and recoverable; sequence is correct with and without a manifest.

**Parallelism:** Go operations and overview interaction can be built in parallel against operation fixtures.

### Slice 9: Notes and media insertion

**Goal:** Support common small edits beyond slide-body text.

**Work:** Edit named Markdown notes in their own files; import/paste/drop local image assets; copy media into the correct bundle location; insert valid relative Markdown syntax; show resolved/missing media diagnostics.

**Dependencies:** Slices 6 and 7; canonical note and asset contracts.

**Acceptance:** Existing note metadata and body remain intact; imported media survives slide moves/duplication according to defined copy semantics; paths resolve in Margo output; large asset operations do not block the UI.

**Parallelism:** Notes editor and media workflow are independent after source/asset APIs settle.

### Slice 10: Presentation review mode and shortcuts

**Goal:** Let authors inspect a deck in a clean, large preview.

**Work:** Full-window presentation review using the Margo interactive output, slide navigation, escape/back-to-editor behavior, selected-slide start, keyboard and optional click navigation, and shortcut reference.

**Dependencies:** Slice 5 preview; Slice 4 slide identity/navigation.

**Acceptance:** Presentation shortcuts work when preview has focus and do not steal editor keystrokes; drafts/hidden slides follow a clear review policy; returning to edit restores selection and scroll state.

**Parallelism:** Can proceed after the preview and navigation contracts are stable.

### Slice 11: Build/export actions and publishing handoff

**Goal:** Connect the app to Margo's existing artifact workflows.

**Work:** Invoke supported build/PDF/PPTX actions through structured engine APIs; show progress, cancellation if supported, diagnostics, and artifact paths. Add entry points for documented CLI publishing flows only when stable.

**Dependencies:** Slice 0 boundary; Slice 3 process lifecycle; engine build/export contracts.

**Acceptance:** App build output matches CLI output for the same project/options; unsupported environment/export combinations produce a clear explanation; app never substitutes a separate exporter.

**Parallelism:** Build/PDF/PPTX commands may be separate tasks if the contract identifies capability and progress events.

### Slice 12: Theme and deck settings panels

**Goal:** Expose stable settings without turning the app into a theme-template editor.

**Work:** Read machine-readable theme options and deck metadata; render controls only for declared supported fields; write through lossless Margo config updates; show unknown options read-only or in source.

**Dependencies:** Stable theme contract in Margo engine and Slices 2, 6, and 7.

**Acceptance:** Controls appear only for declared options, preserve unknown fields/comments where feasible, and the preview reflects saved changes through the Margo renderer.

**Parallelism:** UI control families may split by schema section after contract and file-update behavior are stable.

### Slice 13: Packaging, update compatibility, and release readiness

**Goal:** Deliver a maintainable companion app across selected platforms.

**Work:** Package the app and engine strategy selected in Slice 0; define app/engine compatibility; sign/notarize where available; document installation, project trust, logs, supported projects, and recovery; validate clean install and upgrade paths.

**Dependencies:** Slices 0-12 as applicable; platform matrix from Slice 0.

**Acceptance:** Reproducible release artifacts for supported platforms; app can identify incompatible engine/protocol versions; uninstall/upgrade does not touch user projects; release guide is complete.

**Parallelism:** Platform packaging can be developed per OS once the engine distribution contract is fixed.

## 5. Suggested milestone cuts

### Milestone A: Visual inspection prototype

Slices 0-5. Open a project and inspect exact Margo-rendered slides. No source mutation yet.

### Milestone B: Safe small edits

Slices 6-7. Edit one slide, save, see preview, and safely reconcile external agent/CLI edits.

### Milestone C: Practical authoring companion

Slices 8-10. Organize slides, edit notes/assets, and review presentation mode.

### Milestone D: Workflow integration and release

Slices 11-13. Build/export handoff, stable settings, packaging, compatibility, and docs.

Ship a milestone only when the project remains fully usable through the CLI and all authored changes remain ordinary reviewable project files.

## 6. Cross-cutting verification matrix

Use committed fixtures and temporary copies for destructive operations. Do not test mutations against the working fixture itself.

- Minimal scaffolded deck with default theme.
- `examples/reference-deck` for front matter, layouts, notes, assets, and shortcodes.
- `examples/benchmark-deck` for large slide lists and responsiveness.
- `examples/arca-investor-memo` for real-deck theme and output fidelity.
- `examples/omarchy-theme-showcase` for palette/theme behavior.
- A fixture with a custom theme, deck-level shortcode, missing asset, draft/hidden slide, and malformed Markdown/front matter.
- A fixture edited externally during app session, including while the app buffer is dirty.

At each relevant milestone verify the same source can be built by the CLI, compare app preview with `margo serve` output, and confirm app operations leave `git diff` scoped to intended source files.

## 7. Main risks and mitigations

### Renderer duplication

**Risk:** UI previews differ from published output because Rust/JS recreates Margo parsing or rendering.

**Mitigation:** Go engine owns project model and render; app consumes rendered output and structured model APIs.

### Source corruption or lost concurrent edits

**Risk:** editor save rewrites unsupported front matter, comments, assets, notes, or agent changes.

**Mitigation:** scoped lossless updates, revisions/hashes, atomic writes, conflict UI, and fixture-based round-trip checks.

### Preview privilege boundary

**Risk:** project theme scripts are arbitrary local code and could access native app capabilities.

**Mitigation:** isolate preview from privileged app APIs, constrain navigation and protocol scope, apply the selected shell's least-privilege controls, and document that themes are trusted project code.

### Engine distribution drift

**Risk:** app invokes an incompatible or stale Margo CLI.

**Mitigation:** versioned protocol, explicit capability negotiation, deterministic engine discovery/bundling, and actionable mismatch diagnostics.

### Scope creep toward a general slide editor

**Risk:** GUI controls displace the source-project and publishing value proposition.

**Mitigation:** each UI feature must write canonical Margo files and use the Margo engine; defer canvas positioning and non-source state.

### Preview latency

**Risk:** full project rebuild on every keystroke feels slow.

**Mitigation:** measure early; debounce requests; preserve last good preview; optimize the existing engine only when profiling points to a real bottleneck. Avoid a UI-owned renderer as a shortcut.

## 8. Definition of done for the first public companion release

- User can open a supported Margo project and inspect its entire slide order.
- User can edit Markdown and notes, organize slides, insert media, and review exact rendered output.
- External CLI/agent changes are reconciled without silent loss.
- Build/export actions use Margo and report accurate outcomes.
- All app-created content is represented in ordinary Margo project files.
- Documented install, compatibility, project-trust, and recovery behavior exists for the chosen platforms.
- Verification covers representative fixtures and confirms CLI build parity.
