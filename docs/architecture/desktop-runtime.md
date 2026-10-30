# Desktop runtime decision and protocol v1

Status: implemented MVP, native platform acceptance pending.
Related: [issue #72](https://github.com/jjanuszczak/margo/issues/72),
[desktop PRD](../product/desktop-app-prd.md),
[implementation plan](../product/desktop-app-implementation-plan.md),
[desktop development and use](../../desktop/README.md).

## Decision

Use Rust + Tauri 2 with a bundled Go sidecar on macOS and Omarchy/Arch Linux.
The user's implementation instruction resolves the PRD's runtime/platform spike;
a competing Go UI prototype has not been built and no comparative performance
claim is made.

The sidecar is the Margo executable from this checkout, invoked with
`desktop-engine`. It speaks serialized JSON lines on stdin/stdout. Go owns the
canonical model, source revisions, lossless saves, slide sequencing, and all output
renderers. Rust uses argument-array process invocation and owns native dialogs,
engine lifetime, the editor window, and preview placement. Monaco supplies
Markdown/YAML syntax, find, folding, and per-file session undo/redo. UI state contains buffers, never the only
durable authored content.

Bundle the engine from the same commit rather than searching PATH. This avoids
silent version drift and works offline without a separate CLI installation. The
protocol handshake refuses incompatible app/engine versions. Development and
packaging build the sidecar for the Tauri target triple.

## Preview boundary

A separate child webview shows generated Margo output. It receives no capability
assignment; custom commands additionally reject callers whose webview label is not
`main`. The editor's CSP forbids iframes and external scripts. A preview cannot
navigate outside the Go session prefix or create another window. This design follows
[Tauri's Linux iframe caveat](https://tauri.app/security/capabilities/) and uses its
native child-webview API, currently behind Tauri's `unstable` feature.

The Go HTTP listener binds to `127.0.0.1:0`. Its URL contains a random 192-bit token.
It serves only completed generated files, checks Host, rejects writes, symlinks and
directory listings, sets no-referrer/no-store headers, and has no JSON/API endpoints.
The engine removes its temporary workspace and closes the listener on stdin EOF.
Eight generations are retained for in-flight asset requests. Opening another project
changes the source scope; preview generation numbers continue within the same session.

Templates retain Margo's trusted local-code model, including their external resource
requests. No renderer/parser is reproduced in JS/Rust. Single-slide inspection passes
one canonical slide to the interactive renderer so hidden/draft slides are inspectable.
Presentation review filters hidden slides and includes drafts, with synthetic section
slides as in CLI serve. Publication exports use normal filtering.

## Protocol v1

Every request and response contains `protocol: 1`. One response follows each request.
Messages are capped at 16 MiB. Errors carry `error`, structured `diagnostics`, and
`conflict: true` when applicable. Errors never masquerade as successful preview URLs.

| Request `op` | Input | Success result |
| --- | --- | --- |
| `hello` | protocol | compatible response |
| `open` | absolute `root` folder or margo.yaml | `snapshot` |
| `close` | | clear selected project scope |
| `snapshot` | | `snapshot` |
| `read` | project-relative `path` | `file` |
| `save` | `path`, full `content`, expected file `revision` | new `file` |
| `preview` | `slide`, optional `presentation` or `notes`, selected note `path` | `preview` URL and diagnostics |
| `create` | `name`, expected project `revision` | snapshot, `selected` ID |
| `duplicate` | `slide`, `name`, project revision | snapshot, selected ID |
| `move` | `slide`, one-based `position`, project revision | snapshot, selected ID |
| `delete` | `slide`, project revision | snapshot, nearest selected ID |
| `note` | `slide`, `name`, project revision | newly created file |
| `media` | `slide`, picker `source`, project revision | Markdown reference |
| `thumbnails` | project revision | start background captures |
| `thumbnail-status` | `position` (image cursor) | new PNG data URLs, revision, completion/error |
| `build` | `format`: html/pdf/pptx | artifact path, diagnostics, log |

A snapshot contains canonical root identity, title, active theme, source-tree SHA-256
revision, ordered slide summaries, and available export formats. Each summary includes
bundle ID, title, source path, draft/visibility/layout, named note references, and asset
paths. Notes/assets are always arrays, including when empty. File replies contain the
exact source text and its SHA-256. Inline legacy notes have an empty path and remain
in their parent Markdown file. Paths are project-relative except the canonical root,
external media source explicitly selected by the user, diagnostics, and export artifacts.

Project revision hashes authored inputs, including manifest, partials, shortcodes and
includes. Generated `dist`, hidden tool directories, node_modules and target are
excluded. Clean editor buffers reload external changes; dirty buffers preserve their
base revision until explicit conflict reconciliation. Saving uses a same-directory
temporary file, fsync, hash recheck, and atomic rename. Like ordinary local file editors,
this is optimistic concurrency, not a filesystem lock against uncooperative writers.

## Performance evidence

Local Apple Silicon macOS measurements on 2026-10-29, bundled Go engine, five repeated
single-slide previews per deck. `desktop/scripts/measure-engine.py` reproduces the
measurement. These times exclude webview paint and the 800 ms editing debounce.

| Fixture | Open | Median preview | Slowest preview |
| --- | ---: | ---: | ---: |
| Reference | 42.7 ms | 6.1 ms | 12.7 ms |
| 20-slide benchmark | 17.1 ms | 5.3 ms | 7.6 ms |
| Omarchy showcase | 7.3 ms | 4.5 ms | 5.2 ms |
| ArCa dogfood | 40.9 ms | 11.2 ms | 155.7 ms |

Cold process-to-protocol handshake: 333.5 ms. Development app bundle before the Monaco integration: 42.78 MiB.
Initial budgets: native cold start <2 s, open-to-preview <1 s, preview generation
p95 <300 ms for benchmark-sized projects, external changes visible within 2.5 s while
idle. Native cold-start/paint and memory need platform measurements; engine timings
alone do not establish those acceptance criteria. Full asset hashing on every poll
is intentionally conservative and may need optimization for very large projects.

## Verification and remaining acceptance

The Go suite includes source-byte preservation, conflict rejection, path/symlink
scope, protocol mismatch, notes/media, fixture preview, last-good-preview retention,
read-only loopback HTTP, EOF cleanup, and shared CLI slide operations preserving notes,
assets and metadata through duplication/move/trash. UI model tests cover external
changes and selection preservation. Native macOS inspection confirmed initial window,
folder picker, project model, and separate rendered preview. It exposed a null-assets
selection bug, now corrected on both sides of the contract.

CI is configured to build macOS and WebKitGTK/Linux plus Arch. Those newly added jobs
have not run yet. Actual Omarchy/Hyprland acceptance must verify dialogs, Wayland
placement/scaling, preview keyboard focus, source reconciliation, presentation, and
exports. Native distribution signing, updates, presentation starting at selection, and export cancellation remain incomplete.

## Filmstrip and preview sizing

The slide selector and overview display 320×180 PNG thumbnails captured from Margo's
interactive output. A cancellable Go worker uses the existing Chrome/Chromium capture
pipeline; it does not recreate slide markup. Small PNG data URLs cross the trusted
stdio boundary incrementally. No remote images or iframe privileges are added to
the editor. Capture errors are explicit, and normal editing/preview works while
images load. Chrome/Chromium is required for this MVP thumbnail path on both platforms.

The preview is letterboxed to the current 16:9 deck format within the frame's interior.
Rust maps DOM bounds through the editor webview's native origin and size, then updates
child bounds together. On macOS, native frame height minus the scaled DOM viewport
height supplies the titlebar inset. An unprivileged preview shell scales a sandboxed
1280×720 frame visually, preserving desktop theme breakpoints at every pane size.
This frame never lives in the privileged editor and has no native capabilities.
Layout requests are serialized/coalesced during resizing. Background project polling
never marks navigation controls inert.

### macOS visual verification (2026-10-30)

Verified the approved wordmark in the header, empty state, and generated native icon.
The reference deck preview stayed inside its border at 1360×900, 1000×700, and
1400×1000 window sizes. Divider keyboard changes from 55% to 46% and drag changes
to 29% preserved the complete slide and left the toolbar unobscured. The preview
layout log measured native height 900 versus DOM height 872 at the initial size,
confirming the 28-point macOS content inset. The fixed viewport shell kept the
theme layout constant while scaling. Omarchy visual verification remains pending.

### Fit-to-window follow-up

The shell measures the rendered stage (`main .deck`, a declared stage, or an active
slide fallback) in the fixed source viewport, then scales and offsets its frame to
fit that stage. This excludes browser-page padding without changing the theme's
layout. Scrolling is disabled in both the shell and preview document, so incidental
page overflow cannot create a persistent scrollbar. Fonts, image loads and stage
resizes trigger remeasurement. Inspected the user's jj theme in a resized QA window
with no scrollbar and its complete slide/footer contained in the preview. Explicit
125%/150%/200% zoom controls remain future scope.

## App appearance and editor boundary

Dark, light and system appearance are local preferences, independent of the deck
theme. Monaco has its own persisted theme choice, including both high-contrast
variants. Switching appearance changes neither the buffer nor its model/undo stack.
The app chooses the approved dark/light wordmark to match its background.

Monaco 0.57.0 (MIT) is bundled locally with selected editor contributions and
Markdown/YAML tokenizers. Its worker uses a locally constructed blob URL, allowed
by `worker-src 'self' blob:`. Script execution still uses `script-src 'self'`; no
CDN, remote language service or `unsafe-eval` is introduced. The main window alone
receives the native set-theme capability. Preview capabilities remain empty.
DOMPurify is overridden to 3.4.17 to address transitive dependency advisories.

### Monaco and appearance verification (2026-10-30)

A separate macOS test app opened a disposable copy of the reference deck. Verified
light app colors and the light-background logo, independent dark editor colors,
preferences retained after restart, source edits saved through the Go engine, Find,
and Undo/Redo after saving. Undo also survived switching to another slide and back.
The test source matched the original fixture byte for byte after undo and save.
The platform Edit menu routes Undo/Redo to Monaco's model history; ordinary input
fields retain document history. Production web build, native app packaging, UI model
tests, Rust formatting and `npm audit` passed (zero known npm vulnerabilities).
Monaco's standalone bundle produces a Vite chunk-size warning; the packaged debug
app measured 43.94 MiB. Omarchy runtime/editor acceptance remains pending.

## Named notes follow-up

Desktop note paths convert the content model's bundle-relative `notes/*.md` paths
to project-relative `slides/<bundle>/notes/*.md` paths for source reads and saves.
The Add Note dialog submits Create on Enter; Cancel is an explicit non-submit
button. Creation refreshes the snapshot and opens the new source and notes preview.

The `preview` request accepts `notes: true` and an optional project-relative note
path belonging to the selected slide. Without a path it inspects all notes,
including inline, hidden and draft notes. The standalone reader uses canonical
Markdown rendering in an embedded HTML template, served through the same read-only
loopback server and unprivileged webview. Script execution and forms are disabled
by its CSP. This view does not depend on theme note controls. Published theme note
controls still require the project option and theme support. Regression tests cover
creation, source selection paths, saving, duplicate rejection, rendered Markdown,
wrong-slide path rejection, and canonical theme output with notes enabled.

Native macOS verification used a disposable reference-deck copy: pressing Enter in
Add Note created `slides/01-title/notes/speaker-notes.md`, selected it in Monaco, and
opened its rendered preview. Saving headings and bold text refreshed the reader.
Switching back to slide source showed the theme Notes control; selecting the saved
note again reopened the correct file and rendered content. The user's deck was not
modified during these checks.
