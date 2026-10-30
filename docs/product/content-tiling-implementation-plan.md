# Content Tiling Implementation Plan

Status: plan for the structured tiling model; limited nested-shortcode presets and
desktop insertion UI exist separately. The canonical structured model is not implemented.
Date: 2026-10-30.

Related: [tiling PRD](./content-tiling-prd.md),
[core implementation plan](./implementation-plan.md),
[desktop implementation plan](./desktop-app-implementation-plan.md),
[desktop runtime](../architecture/desktop-runtime.md).

## 1. Objective and assumptions

The current desktop **Tile content** action inserts nested `tiles`/`tile` shortcode
source for Stack, BSP, Columns, Grid, and Track when the active theme already provides
the shortcode templates and CSS. It does not mutate themes or implement the canonical
tile model, source-safe tile editing, geometry contract, validation, or overflow
diagnostics described in this plan. Treat that UI as an interim shortcode composer;
the work below remains proposed and needs the product decisions in the PRD resolved.

Implement optional content-body tiling using canonical Markdown and the Go engine,
with consistent CLI, desktop, HTML, print/PDF, and basic editable PPTX behavior.

For the proposed structured model, the plan assumes whole-composition rebalancing
when tiles change and preset/proportion controls before draggable dividers. These
remain open product decisions in the PRD and do not describe the interim nested
shortcode composer. Resolve them before beginning structured-model implementation.

Keep the first release constrained to one through six tiles, top-level tile blocks,
and named presets. Do not introduce a general layout DSL or an editor-owned model.

## 2. Existing foundations

- [Content parsing](../../internal/content/slides.go) loads slide front matter,
  notes, includes, and bundle assets.
- [Shortcode rendering](../../internal/shortcode/render.go) expands content
  components before Markdown rendering.
- [Shared rendering](../../internal/output/render/render.go) prepares slide bodies
  for interactive and print pipelines.
- [Theme metadata](../../internal/theme/metadata.go) describes layout regions,
  responsive profiles, fitting bounds, and structural policies.
- [Layout auditing](../../internal/layoutaudit) provides rendered observation;
  [split proposals](../../internal/layoutsplit) are a separate existing concern.
- [PPTX output](../../internal/output/pptx/build.go) builds native shapes separately
  from HTML and needs explicit tile support.
- [Desktop engine](../../internal/desktop), [CLI integration](../../internal/cli),
  and [desktop UI](../../desktop/src) provide source revisions, mutations, buffers,
  preview isolation, and export workflows.
- [Scaffolding](../../internal/scaffold) owns the default theme and author guidance.

Weighted column shortcodes are useful precedent but do not provide a structured
tile model for safe desktop edits or native PPTX geometry.

## 3. Architecture and ownership

### Canonical model

Add a structured tiling configuration and ordered tile collection to the canonical
slide model. Each tile needs a stable ID, Markdown content, and source provenance.
Represent source spans or equivalent edit locations separately from rendered HTML.
Retain original source for lossless edits rather than serializing the entire slide
from a reduced model.

Parse tile structure before generic shortcode expansion discards its boundaries.
Render tile contents through existing Markdown, asset, include, and shortcode
mechanisms. The tile notation must have one semantic implementation, not competing
engine and theme shortcode interpretations.

Audit parser behavior for tags inside fenced/inline code, nested shortcodes, raw
HTML, include expansion, and existing notes extraction. Define processing order
and source mapping explicitly. Do not split final HTML with regular expressions.

Included content may render inside a tile using existing include semantics. Before
allowing controls to edit tile structure originating in an include, require reliable
provenance; otherwise provide source editing and an explicit unsupported-operation
diagnostic. Never flatten includes into the parent file as an incidental edit.

### Structural resolver

Use a small pure Go resolver, in a focused package if needed, to validate settings
and map ordered tiles to tracks, spans, splits, and ratios. Inputs include the
declared theme capabilities; outputs carry no component-specific HTML or CSS.

Resolve exact count-specific presets deterministically. Specify positive finite
ratios, valid array lengths, valid primary IDs, supported placements, allowed
alignment values, zero/maximum counts, and malformed block handling. Reject
conflicting or unsupported settings rather than guessing.

The theme supplies the body container and gap tokens. Templates convert the
structural result into CSS Grid markup and CSS custom properties. A generic
geometry conversion can map the same structure into an export body rectangle,
accounting for that output's declared spacing and reserved regions.

Shared topology does not imply pixel-identical text metrics between browsers and
PowerPoint. Document supported geometry and content fidelity separately.

### Theme contract

Extend theme metadata with explicit tiling capability/preset declarations. Final
field names must be agreed with the PRD syntax before implementation. Expose stable
mechanism-level data to templates, not a collection of feature-specific booleans.

Implement the default-theme body wrapper/partial, tile markup, CSS Grid rules,
spacing/alignment tokens, and print styles. Use bounded tracks such as
`minmax(0, 1fr)` where appropriate so intrinsic content size does not expand the
slide. Detect resulting content overflow; do not hide it as a substitute for fitting.

Make styling reusable across interactive and print layouts. Respect each theme's
body boundaries; never hardcode title/footer offsets into the resolver.

### Desktop boundary and mutation safety

Expose composition summaries, capabilities, tile IDs, source locations, and
diagnostics through the Go protocol. Add shared operations for create/add/remove/
reorder/configure as needed. Rust forwards requests; JS presents controls.

Choose an explicit dirty-buffer strategy before building controls: either request
a revision-aware source transformation applied to the Monaco model as one undoable
edit, or safely save/reconcile the buffer before a disk mutation and integrate the
result with editor history. Verify the strategy against existing undo behavior.

Use expected revisions, scoped source edits, atomic saves, and current conflict
handling. Preserve unknown front matter, comments, other tile bodies, notes, and
assets. Deleting a tile is an explicit content-removal action and must be undoable.
Removing the last tile requires the PRD's explicit handling.

Keep all edit controls in the privileged editor. The separate preview continues to
serve read-only generated output without native capabilities. Any tile highlighting
must respect this boundary. Never create an authoring API on the preview server.

## 4. Delivery slices

### Slice 1: Freeze the contract and implement the content model

Work:

- Resolve the two open product decisions.
- Finalize front matter, tile syntax, supported presets/options, and diagnostics.
- Define exact five- and six-tile placement, source/reading order, and transitions
  for every supported count.
- Define theme capability behavior and preset-specific ratio/placement semantics.
- Implement canonical parsing, source provenance, structural validation, and the
  deterministic resolver.

Acceptance:

- Valid source yields a stable model and composition for every supported count.
- Invalid nesting, duplicate IDs, missing primary references, malformed settings,
  stray body content, and unsupported themes produce useful source diagnostics.
- Code examples remain literal; notes, includes, assets, and nested ordinary
  shortcodes retain their intended behavior.
- Existing untiled source and column shortcodes behave as before.

### Slice 2: Default-theme rendering, print output, and diagnostics

Depends on Slice 1.

Work:

- Integrate resolved compositions into shared rendering.
- Add default-theme markup/CSS, preset support, spacing/alignment controls, and
  print rules in scaffold output.
- Extend existing audit machinery to measure each tile and associate findings with
  tile IDs and source locations.
- Preserve typography by default and check interactions with existing opt-in fit
  policies so global fitting cannot silently mask tile overflow.
- Confirm fixed-canvas geometry and keep published mobile profiles separate.

Acceptance:

- `build` and `serve` render tiled HTML without a browser requirement.
- Interactive HTML and print HTML allocate corresponding areas; generated PDF
  preserves the intended placement.
- Dense text, wide tables, tall images, and nested shortcodes produce tile-specific
  overflow evidence when rendered auditing runs.
- Missing audit-browser dependencies are explicit; structural validation still works.
- Header/chrome/annotations/footer remain outside tile allocation.

### Slice 3: Shared mutations and desktop controls

Depends on Slices 1–2 and a reviewed source/undo strategy.

Work:

- Add Go source operations and protocol responses with expected revisions.
- Provide desktop create-count, add/remove/reorder, preset, primary, placement,
  proportion, spacing, and alignment controls for declared capabilities.
- Connect diagnostics to tile/source identification.
- Expose CLI mutation commands only where useful, using those same operations and
  non-interactive arguments. Finalize command grammar before documenting it.
- Preserve selected tile by stable ID across operations and external changes where
  that ID still exists.

Acceptance:

- "Create 3 tiles" writes three ordinary tile blocks and previews the result.
- Adding a fourth tile rebalances auto mode; removal and reorder preserve intended
  content and documented preset behavior.
- Undo/redo, dirty buffers, unsaved changes, external edits, and stale revisions
  work without silent source loss.
- CLI edits are reflected in the app and app edits build through CLI.
- Preview-pane resizing scales the slide without changing the composition.
- Preview isolation and read-only serving remain intact.

### Slice 4: Editable PPTX, documentation, and release acceptance

Depends on the structural contract from Slice 1; final acceptance follows Slice 3.

Work:

- Consume shared composition structure in PPTX, using the export theme's declared
  body geometry and spacing to produce tile rectangles.
- Place supported editable text and local images within tile bounds; retain
  contain behavior unless explicit supported cropping is selected.
- Define and document the exact supported text subset. Diagnose unsupported tables,
  shortcodes, media, or decoration rather than silently flattening tile placement.
- Add authored fixtures, scaffold examples, authoring guidance, and theme-contract
  documentation. Update portable agent guidance if it describes supported layouts.
- Verify desktop behavior on macOS and actual Omarchy/Hyprland.

Acceptance:

- Basic text/image tile geometry is preserved in PPTX and content remains editable.
- Unsupported content has clear output-specific diagnostics.
- All PRD acceptance criteria have recorded verification evidence.
- Documentation distinguishes supported features from deferred drag/split behavior.

## 5. Verification strategy

Start with focused Go tests for parsing, resolver invariants, and source mutations.
Assert positive bounded geometry, no overlapping tile areas, stable order/IDs,
spacing-aware ratios, and documented behavior for count transitions. Test
round trips against source with comments, unknown metadata, notes, assets, includes,
and external revision changes.

Add rendering/audit integration cases for one through six tiles, every preset,
primary placements, ratio changes, empty tiles, dense Markdown, wide tables, local
images, and supported shortcodes. Browser-backed tests must wait for fonts/assets
and separate missing dependencies from layout failures.

Use committed fixtures and disposable copies for mutation/native QA:

- [Reference deck](../../examples/reference-deck): supported content and regression.
- [ArCa deck](../../examples/arca-investor-memo): theme/body/print compatibility.
- [Benchmark deck](../../examples/benchmark-deck): rebuild/preview regression.
- [Omarchy showcase](../../examples/omarchy-theme-showcase): theme compatibility.
- A fresh scaffold and a focused committed tiling fixture or equivalent additions
  to the reference deck, chosen during implementation.

Do not claim legacy custom themes support tiling without explicit implementation.
Verify their existing slides remain unchanged and unsupported tiled slides receive
the declared diagnostic. Add custom-theme tiling acceptance only when that theme
has adopted the contract.

Run focused packages first, then the Go suite with the repo's explicit `GOCACHE`.
For desktop changes, run UI tests, the production web build, and Rust formatting
checks. Inspect actual HTML, print HTML/PDF, and PPTX outputs. Native macOS and
Omarchy tests must cover controls, keyboard operation, conflicts, preview geometry,
and export; compilation does not establish native acceptance.

Measure representative preview/rebuild performance against the existing baseline.
Do not add browser layout measurement to the ordinary HTML build path.

## 6. Risks and controls

| Risk | Control |
| --- | --- |
| Feature grows into a layout language | Constrain presets, counts, options, and nesting |
| Source corruption or lost unsaved edits | Provenance, scoped transformations, revision checks, undo integration |
| Theme/engine coupling | Structural model in Go; markup and visual decisions in templates/CSS |
| Areas shrink but content becomes unreadable | Preserve typography, report tile overflow, honor explicit fitting bounds |
| HTML/PDF/PPTX drift | Shared topology, declared output geometry, separate fidelity verification |
| Existing themes change unexpectedly | Explicit capability opt-in; no migration of untiled slides |
| Desktop duplicates semantics | Shared Go operations; Rust/JS remain interaction and lifecycle layers |
| Preview gains authoring privileges | Trusted editor controls and existing read-only preview boundary |

## 7. Definition of done

The first release supports the PRD's complete source-driven and desktop workflows,
passes output and source-safety acceptance, documents supported theme/export limits,
and leaves existing projects usable through CLI without conversion. No required
behavior depends on hidden desktop state, a window manager, or new online services.
