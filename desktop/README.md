# Margo desktop (MVP)

Rust + Tauri 2 desktop companion for creating and editing Margo projects. The
bundled Go engine owns project interpretation, rendering, slide operations, and exports.
No separately installed Margo CLI or account is required. The app and bundled
editor work locally; a deck's theme or content can still request external resources.
See the [authoring guide](../docs/AUTHORING_GUIDE.md) for the shared project model
and the [runtime decision](../docs/architecture/desktop-runtime.md) for the protocol.

## Status

The MVP targets macOS and Omarchy/Arch Linux. Apple Silicon macOS has local native
verification for project opening, previews/resizing, source save/undo, app/editor
appearance, and named notes. The debug app has built and launched in an Omarchy
session; full native acceptance remains pending. Tagged
releases attach an experimental macOS disk image and Arch Linux portable archive
alongside the independently usable CLI archives. The desktop app includes its
matching Go engine and does not require the CLI to be installed separately.

## Develop

Install Go 1.22+, a current stable Rust toolchain, and Node.js 22+. Install the
[Tauri system prerequisites](https://v2.tauri.app/start/prerequisites/) for your OS.

```sh
cd desktop
npm ci
npm run dev
```

The app uses the approved `assets/new-logo-dark-bg.png` and
`assets/new-logo-light-bg.png` wordmarks. The native icon derives from the dark version.
Run `npm run icons` to regenerate packaged icons after changing that source.

`npm run dev` builds the matching Go sidecar automatically, then starts Vite and
Tauri. `npm run web:build` builds only the web assets. `npm test` checks the editor
reconciliation model. Run `go test ./...` from the repository root for engine tests.

## macOS

Install the Xcode command line tools. Build a local app:

```sh
cd desktop
npm ci
npm run build -- --bundles app
```

The result is `src-tauri/target/release/bundle/macos/Margo.app`. For a faster local
test build, add `--debug`; that result is under `target/debug/bundle/macos/`.
Open the `.app` from Finder. Quit and reopen an existing instance after rebuilding;
replacing the bundle does not update an already running process.
The app and its Go engine must stay together inside the bundle. Signing and
notarization credentials are not configured in this MVP.

## Omarchy / Arch Linux

Use the native Arch build for the MVP, so WebKitGTK and the desktop portals match
the installed system. Start with Tauri's Arch prerequisites:

```sh
sudo pacman -S --needed webkit2gtk-4.1 base-devel curl wget file openssl appmenu-gtk-module libappindicator-gtk3 librsvg xdotool
```

Install `chromium` if no Chrome-compatible browser is present; it supplies rendered
slide thumbnails and PDF export. The file picker uses the desktop portal. An Omarchy/Hyprland session needs a
working `xdg-desktop-portal` setup with a file chooser backend (normally
`xdg-desktop-portal-gtk` alongside the Hyprland portal). Keep your normal Omarchy
portal configuration; do not replace it just for Margo.

```sh
cd desktop
npm ci
npm run engine
npm run web:build
cd src-tauri
cargo build --locked --release --features custom-protocol
mkdir -p target/release/margo-app
cp target/release/margo-desktop target/release/margo-app/
cp binaries/margo-engine-x86_64-unknown-linux-gnu target/release/margo-app/margo-engine
./target/release/margo-app/margo-desktop
```

Keep both executables in the same directory. On ARM64, use the
`margo-engine-aarch64-unknown-linux-gnu` binary instead. `npm run build` also
supports Linux packages; native Arch builds are the preferred Omarchy test path.
CI includes macOS, Ubuntu/WebKitGTK, and an Arch container build. A container build
does not establish working Hyprland input, file dialogs, or GPU rendering.

## Use

- **Open project folder** opens an existing deck in place. **Open .margo archive**
  opens a file picker and extracts the selected project beside the archive. If a
  folder with that name already exists, the app asks for a different project folder
  name. It does not overwrite existing content.
- **Project themes** in the toolbar or native Projects menu lists installed themes
  and marks the active one. Switching themes updates `margo.yaml`, keeps compatible
  option values, uses the new theme's defaults for incompatible values, and refreshes
  the Margo-rendered preview. Add
  themes from Git, a `.margot` archive, another project
  folder, or a `.margo` archive. Project imports copy only the selected theme and
  leave it inactive. Theme templates and scripts run during local rendering, so
  import only from sources you trust.
- **New project** creates a standard deck with the Go scaffold. Choose a parent
  folder and name; optionally enter a Git theme repository and revision during
  setup. If theme installation fails, the scaffolded default-theme deck remains.
- **Tile content** offers graphical cards for Stack, BSP, Columns, Grid, and
  Track. Choose 2–6 areas; Stack also sets the primary width, while Track sets
  the number of vertical lanes. The selected arrangement is inserted as nested
  shortcodes in the current Markdown selection. Scroll is omitted because slides
  use a fixed canvas. Tiling is enabled only when the active theme supplies both
  `tiles` and `tile` shortcodes and their CSS; custom themes are never modified
  automatically.
- **Agent skills** lists project-local skills from `.agents/skills/` and builds a
  task-specific prompt with the deck root, theme, selected slide and source paths,
  the skill instructions, and its Markdown references. YAML skill metadata is
  omitted from the copied prompt. The target agent must have access to the same
  project folder; the app does not provide an AI runtime.
- Use the native **Projects** menu beside Edit, Window, and Help for New/Open,
  recent projects, and HTML/PDF/PPTX export. Icon-only New, Open, and AI handoff
  buttons sit on the ribbon with hover tooltips; Overview and Present also use
  tooltip-labeled icons. The welcome screen presents New project and Open existing
  project side by side.
- The left filmstrip and overview show captured images of the actual interactive
  slides. Thumbnails fill in progressively in the background. Chrome/Chromium is
  required for capture, using the same browser discovery as PDF/PNG export.
- Select a slide in the left list. The full source, including front matter, appears
  below its Margo-rendered preview. The default view fits the complete rendered slide
  inside the pane without page margins or scrollbars, scaling on window/divider
  resize. Drag or use the keyboard on the divider to resize. Zoom presets are
  deferred to a later version.
- Auto-save waits 800 ms after typing. Turn it off for manual saves with Cmd/Ctrl+S.
  Undo/redo uses standard editor keys and retains history across saves in this session.
- Add a note with **+ Note**, enter a name, then press Enter or **Create**. This
  creates `slides/<bundle>/notes/<name>.md` and opens it in the editor and rendered
  notes preview. Choose saved notes from the source menu. **Preview notes** shows
  all slide notes when editing the slide source; **Back to slide** restores the
  slide preview. Saving a note refreshes its preview. Legacy inline notes remain
  editable in the slide's full source and can be inspected with **Preview notes**.
  Published slide controls still follow `presentation.navigation.notes: true`
  and the theme's support for notes; the desktop reader works independently.
- **Overview** shows rendered thumbnails of the ordered deck. Drag a card or list item onto
  another position to reorder; arrows move the selected slide by one position.
- **Duplicate** copies the entire bundle, including unknown front matter, assets,
  and notes. **Delete** asks for confirmation and moves the bundle to `.margo-trash`.
  Restore a deleted bundle manually and repair manifest/order using the CLI.
- **Insert media** copies a selected local image/video into the current bundle using
  a collision-resistant filename, inserts a relative reference, then saves.
- **Present** opens the Margo interactive deck in the large preview. Use the theme's
  navigation. Click **Back to editor** to return. Draft slides are included, hidden
  slides are excluded. Selected-slide previews can inspect either state.
- **Export** invokes Margo's HTML, PDF, or PPTX renderer. The status area reports the
  artifact path. PDF appears only when Chrome/Chromium is detected. Export excludes
  drafts/hidden slides, like normal CLI builds. PPTX retains Margo's existing limits.

## Appearance and source editor

Choose **Dark**, **Light**, **System**, or **Omarchy** in the header. Omarchy reads
the active theme's `colors.toml` palette, applying its background, foreground,
accent, selection, and muted colors to the app surface and controls. It refreshes
while the app is open when the active theme changes. If the palette is unavailable,
it falls back to system colors. The source toolbar has an independent editor theme:
match the app, light, dark, or either high-contrast theme. Both choices persist
across restarts. App appearance does not change the deck theme.

The source editor is [Monaco](https://github.com/microsoft/monaco-editor), licensed
under MIT. It edits Markdown and YAML directly, with syntax highlighting, find,
folding, and per-file session undo history. Raw HTML, front matter and shortcodes
remain source text. Monaco and its worker ship inside the app, without a CDN or
remote language service. The dependency lock pins Monaco; a DOMPurify override
keeps its transitive sanitizer on the patched 3.4.17 release.

## External changes and recovery

The app polls project inputs every 1.5 seconds while idle. Changes to clean sources
reload automatically. Dirty sources keep your text and display a conflict. Compare,
reload disk, or explicitly keep your text and press Save. There is no force-save
endpoint. The engine rechecks a file's SHA-256 before atomic replacement.

Saving replaces only the requested source bytes and preserves file permissions.
It does not regenerate front matter or discard unknown metadata. Structural slide
operations share the CLI's ordering, positional bundle renaming, and manifest rules;
they can therefore rename several bundles just as CLI move/insert/delete does.
Symlinked project content is rejected by this initial desktop implementation.

Preview failure keeps the last successful output with a stale label and diagnostics.
Source can still be repaired in the editor after a bad save. Closing/switching saves
ordinary dirty text first; unresolved conflicts must be resolved or explicitly
abandoned before closing. Git remains the durable history.

Opening a project trusts its theme, shortcodes, snippets, and assets as executable
local content. Preview is a separate native webview with no desktop capabilities,
not an iframe inside the privileged editor. External navigation and popups are
blocked in that webview. Its HTTP server binds only to loopback and serves generated
artifacts beneath a random session URL. It has no mutation or engine API endpoints.

Engine stderr is saved to Tauri's platform log directory (`engine.log`). Build and
render errors appear in the persistent diagnostics panel. On macOS logs live under
`~/Library/Logs/dev.margo.desktop/`; on Linux Tauri uses the platform data/log path.
Recent-project metadata is in the platform app-data directory. Removing the app
never removes projects. An engine crash requires reopening the app.

## Verification

From the repository root:

```sh
npm test --prefix desktop
npm run web:build --prefix desktop
go test ./internal/desktop ./internal/cli
cargo fmt --manifest-path desktop/src-tauri/Cargo.toml -- --check
```

The engine tests cover source revisions and conflicts, scoped paths, note creation
and rendering, preview isolation, and fixture output. Real Chrome thumbnail tests
are opt-in with `MARGO_DESKTOP_THUMBNAIL_TEST=1 go test ./internal/desktop -run Thumbnail`.
Use a disposable copy of a committed deck for manual saves, media insertion, notes,
and structural slide operations. Check window/divider resizing, theme choices,
source selection, external edits, and exports in the native app on each target OS.

The [Desktop CI workflow](../.github/workflows/desktop.yml) defines macOS app,
Ubuntu package, and Arch compile jobs. Tagged [releases](../.github/workflows/release.yml)
build the macOS disk image and Arch archive. Review run results separately; a
successful compile does not establish working native dialogs, focus, or Wayland
rendering.

## Current validation boundaries

The first local build targets Apple Silicon macOS. Engine rendering tests cover
reference, benchmark, Omarchy showcase, and available ArCa fixtures. The app has
also built and launched in an Omarchy session, but runtime checks of palette
matching, native dialogs, Wayland geometry, keyboard/focus, and export fidelity
remain user acceptance checks.

Presentation starts at the beginning of the interactive deck. Escape from a focused theme webview follows
the theme's behavior; the visible Back button always returns to the editor. Theme
settings panels are follow-on scope in the PRD. Cancellation of exports is not yet
available. Uninstall/update automation and notarized distribution are not implemented.
