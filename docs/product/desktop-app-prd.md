# Margo Desktop App Product Requirements Document

## 1. Purpose

Define an optional graphical front end for Margo projects. The app helps a person inspect a deck, make small source edits, and see the rendered result quickly. The Margo project on disk remains the source of truth, and the CLI remains the primary interface for automation, agents, builds, and publishing.

This app extends Margo's project and publishing workflow. It does not replace it with a new document format or a separate editor-owned content model.

## 2. Product decision

Build the optional app with Rust + Tauri 2 and a bundled Go engine, targeting macOS
and Omarchy/Arch Linux for the MVP. This runtime choice follows the implementation
direction for issue #72. Go remains responsible for all Margo project semantics;
Monaco provides source editing in the app. The
[runtime decision](../architecture/desktop-runtime.md) records the boundary and
measured evidence. No comparative Go-hosted UI prototype was built.

The app should offer the practical preview and editing capabilities users expect from Hype: slide list and overview, selected-slide Markdown editing, live preview, reordering, add/duplicate/delete, asset insertion, theme/layout controls when the project exposes them, presentation mode, and PDF/PPTX export entry points where Margo supports them.

Margo's differentiator remains the structured project and publishing system: slide bundles, reusable HTML/CSS themes, layouts, partials, shortcodes, notes, assets, validation, build outputs, packaging, agent workflows, and deployment. The app makes that system easier for people to inspect and edit.

### Current implementation status

The MVP implements new-project creation through the Go scaffold, optional Git theme
import, project opening/recent folders, rendered thumbnails and overview, a preview
that fits the pane on resize, Monaco source editing, named Markdown notes and rendered
notes preview, conflict handling, slide operations, media insertion, presentation
review, and HTML/PDF/PPTX export entry points. A graphical tiling control writes
theme-supported nested shortcodes without changing the theme. Agent-skill handoff
builds a contextual prompt for copying to an external agent; the app does not run an
AI agent. The native Projects menu and icon toolbar expose project, export, and view
actions. App appearance supports light/dark/system plus the active Omarchy palette,
with separate editor theme preferences. Files on disk remain the source of truth.

Apple Silicon macOS has local native verification. The debug app has built and
launched in an Omarchy session; full palette, dialog, Wayland, and export acceptance,
performance/memory acceptance, signing/notarization, and automatic updates remain
pending. Deck/theme settings panels, selected-slide presentation start, zoom presets,
and export cancellation are not implemented. Requirements below describe intended
scope, not a claim that every acceptance measure has passed. See the
[desktop guide](../../desktop/README.md) for current usage and limits.

## 3. Target users and jobs

### Primary user

A Margo deck author who works with Markdown and Git, and wants a visual way to inspect a deck, review the effect of source changes, and make small edits without leaving the project workflow.

### Secondary users

- A reviewer who needs to browse slides and leave the source tree ready for an author to edit.
- A theme author who wants to inspect how theme and layout changes affect a real deck.
- An AI-agent user who asks an agent to make larger changes, then reviews and adjusts them in the app.

### Jobs to be done

- Open an existing Margo project and understand its slide order, draft/hidden state, notes, assets, and active theme.
- Preview the deck using Margo's actual renderer and theme, with updates after source changes.
- Make lightweight changes to slide Markdown and project-supported metadata while keeping files human-readable and version-control friendly.
- Add, duplicate, move, and remove slides without damaging bundle assets or notes.
- Add local image/video assets and insert valid references into slide source.
- Inspect diagnostics and jump from an issue to the relevant source.
- Hand the project back to CLI, Git, build, package, and deploy workflows without a conversion step.

## 4. Product principles

- The project files are canonical. The app never stores the only copy of authored content in a proprietary database.
- Margo's Go engine defines parsing, validation, rendering, and export behavior. The desktop shell must not reimplement those rules in Rust or JavaScript.
- The preview must use the same Margo rendering pipeline as `margo serve` and `margo build`; it must not approximate theme output with a second renderer.
- Text remains the default editing surface. Graphical controls are a convenience for supported, explicit fields, and write normal Margo source.
- Agent and CLI changes made on disk must appear in the open app without overwriting unsaved user edits.
- Local-first operation is the default. Opening and editing a deck must not require an account or network connection.
- Keep startup, project loading, and edit-to-preview latency low enough to feel immediate on an ordinary laptop.
- Support the full Margo project model progressively. Unsupported constructs must render through Margo or be clearly identified, never silently rewritten or discarded.

## 5. Scope

### Initial release scope

- Open a project by selecting its root or `margo.yaml`; remember recent projects.
- Create a new project with Margo's canonical scaffold and optionally install a Git-hosted theme.
- Show slide navigation with title/summary, order, and visible state indicators for draft/hidden slides.
- Show a live preview of the selected slide and a Markdown editor for its source.
- Provide a full-deck overview/grid for selection and reordering.
- Create, duplicate, move, and delete slide bundles using Margo's existing project conventions and safe file operations.
- Insert images and other supported local media into the selected slide bundle and create valid Markdown references.
- View and edit slide notes as named Markdown files where supported by the project model.
- Show build/render diagnostics, identify source paths, and offer a way to open them in the editor.
- Detect external file changes, refresh clean buffers, and warn/reconcile when a file changed while its buffer has unsaved edits.
- Provide a presentation preview mode for reviewing the rendered deck.
- Offer a graphical tiling selector that inserts canonical nested tile shortcodes only when the active theme declares support; do not silently modify themes.
- Provide a contextual prompt handoff for repository-local agent skills without embedding an agent runtime or provider.
- Invoke Margo's supported build/export commands and report progress and errors.
- Provide basic undo/redo for edits made in the current session, with clear save status. Source files remain the durable history; Git remains the project history.
- Support the same operating systems targeted by the first Margo desktop release, selected during technical validation.

### Follow-on scope

- Theme and deck settings panels for stable, documented configuration fields.
- Theme development workflow with live refresh when CSS/templates change.
- Richer diagnostics, source navigation, and accessibility checks.
- Batch media import and additional safe authoring helpers.

### Out of scope

- Replacing the CLI or making the desktop app required for a Margo workflow.
- A proprietary deck file format, cloud storage, accounts, or collaborative real-time editing.
- A freeform canvas or drag-to-position editor that bypasses themes and source files.
- Reimplementing arbitrary HTML, CSS, shortcode, or theme semantics in the UI.
- Matching every Hype feature irrespective of Margo's product model. Feature parity is a reference for useful preview/editing interactions, not a mandate to copy product architecture.
- PowerPoint import or fidelity guarantees beyond Margo's existing PPTX capability contract.
- Arbitrary theme-option editing before Margo exposes a stable machine-readable theme contract.

## 6. Default workspace layout

The main editing window should follow the familiar presentation-editor arrangement the user prefers from Hype:

- A slide selector runs down the left side, showing slide order and useful draft/hidden status.
- The selected slide's rendered preview occupies the main upper work area.
- A Markdown editor for that slide sits below the preview.
- The preview/editor divider can be resized so the author can prioritize visual review or source editing.
- A full-deck overview is available as a separate mode for scanning and rearranging slides, while the normal workspace keeps the current slide in focus.

The layout should feel like a presentation editor, not a web admin dashboard. Keep the preview large, the source immediately accessible, and slide navigation continuously visible in the normal editing mode. Keyboard navigation and selection should behave predictably when focus is in the editor.

## 7. Core workflows

### Open, inspect, and edit

1. The author opens a Margo project.
2. The app loads the deck model and renders a preview through Margo.
3. The author selects a slide, edits Markdown or notes, and sees the preview update after a successful render.
4. The app saves to the existing bundle files and reports rendering or validation errors beside the source.

### Organize slides

1. The author selects one or more slides in the list or overview.
2. The author adds, duplicates, moves, or deletes slides.
3. Margo's safe project operations preserve bundle assets and notes and keep slide order consistent.
4. The app refreshes the model and preview from disk after each operation.

### Review agent or CLI changes

1. An agent or CLI edits project files while the deck is open.
2. The app detects changed paths and refreshes any clean buffers.
3. For dirty buffers, the app presents a clear conflict choice and does not silently overwrite either version.
4. The author reviews the new render and continues editing.

### Export and publish

1. The author uses the app to run a Margo build or supported export.
2. The app reports the exact output location and any diagnostics.
3. Publishing configuration and deployment remain ordinary Margo project/CLI workflows. The desktop app may expose these commands once their interface is stable, but it does not create a separate publishing system.

## 8. Functional requirements

### Project lifecycle

- Detect a valid project by `margo.yaml` and show actionable errors for invalid or unsupported projects.
- Preserve existing file names, front matter, comments, whitespace where practical, bundle-local assets, notes, and unknown metadata when editing a supported field.
- Never write generated `dist/` artifacts as authored source.
- Do not make assumptions based on the process working directory; every operation is scoped to the selected project root.

### Preview and rendering

- Render with the installed or project-selected Margo version/engine appropriate to the project.
- Preview must represent interactive HTML output, including theme CSS/JS and local assets. Clearly distinguish print/PDF preview if exposed.
- Show drafts in authoring mode and visually mark them. Hidden/skipped slides should be inspectable without being mistaken for normal published slides.
- Keep the selected slide stable when slides are inserted, removed, or reordered elsewhere.
- Show a stale/failed-render state rather than presenting old output as current.
- Support preview resizing and a full-window presentation review mode. Keyboard navigation must not steal keystrokes from text inputs.

### Editing

- Provide a Markdown editing surface with syntax-aware affordances and standard text editing keys.
- Save to canonical files, with visible dirty/saved state and explicit conflict handling for external modifications.
- Support slide body and front matter editing; the initial structured controls should be limited to fields with stable schemas.
- Support notes editing without mixing notes into slide-body Markdown.
- Offer safe operations for slide creation, duplication, reordering, and deletion, including confirmation for destructive deletion and honoring Margo's recoverability behavior.
- Media insertion must copy or link assets according to documented Margo conventions and insert a path that resolves from the bundle.

### Diagnostics and operations

- Display parser, build, and render errors in a persistent diagnostics area with source location when available.
- Expose build, PDF, and PPTX actions only when supported by the project and local environment.
- Show progress, cancellation where supported, exit status, and output paths.
- Keep operational logs accessible for bug reports without requiring a cloud service.

## 9. Non-functional requirements

- Startup and open-project performance are explicit acceptance measures in the implementation plan; avoid unnecessary runtime duplication and large asset copies.
- Preview should update quickly for common slide edits and make any slower full-deck rebuild visible.
- The UI must remain responsive during builds and asset processing.
- The app must not execute shell strings assembled from project content. Process invocation must use argument arrays and a controlled working directory.
- Treat project themes and their HTML/JS as executable local content, consistent with Margo's existing trust model. Show the project identity and give users a clear choice before opening an unfamiliar project if needed.
- Use atomic writes or equivalent protection for source updates where practical; never truncate an existing source file on a failed save.
- Maintain compatibility with the active Margo project format and report version/contract mismatches clearly.
- Accessibility: keyboard operation for core navigation/editing, meaningful focus order, and readable diagnostics.

## 10. Success measures

- A user can open a representative Margo project, inspect all slides, edit Markdown, and see the exact Margo-rendered result without using a browser or terminal.
- A project edited in the app remains buildable through the CLI with no conversion or cleanup step.
- App edits preserve notes, assets, unrecognized metadata, and unrelated files in the touched slide bundle.
- External CLI/agent edits are reflected without losing unsaved work.
- Common edit-to-preview latency and cold-start time meet agreed budgets on a defined reference machine.
- A first-time user can identify current diagnostics and the associated source without needing to read logs.
- The app remains useful when a deck uses advanced Margo templates or shortcodes that have no GUI controls, because preview delegates to Margo and source editing remains available.

## 11. Resolved decisions and remaining acceptance

- Separate native desktop app, with a Go sidecar bundled from the same checkout.
- Private versioned JSON-lines protocol on stdin/stdout; no engine mutation API over HTTP.
- Random-token loopback server for generated artifacts, displayed in a separate unprivileged webview.
- macOS and Omarchy/Arch Linux MVP targets; actual Omarchy session acceptance remains required.
- HTML and PPTX export actions, plus PDF when Chrome/Chromium is detected.
- Locally bundled Monaco source editor, with no CDN or remote language service.
- Native startup/paint, memory, focus, and export-fidelity acceptance remain open; engine measurements alone do not establish them.

Distribution signing, update delivery, and complete platform acceptance must be
resolved before presenting the MVP as a finished desktop release.

## 12. Relationship to existing product docs

This app is an optional companion to the Margo engine described in [PRD.md](./PRD.md). It does not change the canonical source model, output contracts, CLI, packaging, or publishing requirements. Desktop-app implementation work should follow [desktop-app-implementation-plan.md](./desktop-app-implementation-plan.md) and coordinate engine/API changes with the main [implementation-plan.md](./implementation-plan.md).

Hype is a useful reference for rapid slide navigation, overview, live Markdown preview/editing, media insertion, presentation review, and export affordances. Margo should adopt those user benefits where they fit its source-project model while retaining its theme-driven rendering and automation-first project workflow.
