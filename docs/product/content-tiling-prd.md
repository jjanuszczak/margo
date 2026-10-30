# Content Tiling Product Requirements Document

Status: proposed structured tiling model; a separate nested-shortcode composition
and desktop insertion UI are implemented in limited form.
Date: 2026-10-30.

Related: [implementation plan](./content-tiling-implementation-plan.md),
[Margo PRD](./PRD.md), [desktop PRD](./desktop-app-prd.md),
[authoring guide](../AUTHORING_GUIDE.md),
[desktop runtime](../architecture/desktop-runtime.md).

## 1. Purpose

Let authors define several content areas within a content slide and have Margo
allocate the available space automatically. Adding or removing an area recalculates
the composition. Authors can choose presets, placement, and proportions without
writing HTML or CSS.

The interaction takes inspiration from desktop tiling window managers: content
occupies allocated areas and changes to the set of areas redistribute space.
This proposal does not require integration with a window manager or an OS service.
It must work through Markdown and the headless CLI as well as the desktop app.

## 2. Product direction and decision status

### Current implementation boundary

The desktop app currently generates editable nested `tiles` and `tile` shortcodes
for Stack, BSP, Columns, Grid, and Track arrangements. It enables the control only
when the active theme provides both shortcode templates and CSS selectors for the
composition. It does not edit an existing tile tree, expose a structured tile model,
change theme files, or provide tile diagnostics and overflow validation. This is a
limited integration built on the existing shortcode system, not implementation of
the structured contract proposed below. The proposed syntax, settings, and acceptance
criteria remain unimplemented until the model and theme API are explicitly adopted.

Add **content tiling** as an optional composition inside `layout: content`.
Existing untiled slides keep their current behavior. Tiles occupy the body region;
titles, subtitles, sources, branding, and footers remain owned by the slide layout.

The syntax and defaults below are proposed contracts for implementation review,
not existing supported features. Two interaction decisions remain open:

| Decision | Recommended first-release default | Alternative |
| --- | --- | --- |
| How a new tile changes placement | Rebalance the whole composition using its selected preset | Split the selected tile and preserve surrounding areas |
| How authors resize tiles in desktop | Preset and proportion controls | Draggable dividers required at first release |

This PRD and the implementation plan assume the recommended defaults. Confirm
these decisions before implementation; choosing the alternatives changes scope.

## 3. Users and jobs

- A Markdown author creates three content areas, fills them, and gets a usable
  composition without theme edits.
- A desktop author chooses "Create 3 tiles", then adds, removes, reorders, and
  adjusts areas while inspecting the actual Margo preview.
- An agent or automation process authors the same source and builds it without
  desktop state or interactive prompts.
- A theme author styles tiled content through a documented structural contract.

## 4. Scope

### First release

- One to six explicit tiles per tiled content slide.
- Ordinary Markdown, local images, tables, and supported shortcodes inside tiles.
- Automatic arrangement and a small set of named presets.
- Stable tile IDs, source order, primary-tile selection, placement, proportional
  sizing, spacing, and constrained alignment controls.
- Add/remove/reorder operations that recalculate available body space.
- Canonical source shared by desktop, CLI, previews, and exports.
- Tile-level validation and rendered overflow diagnostics.
- Default-theme HTML and print support, PDF verification, and basic editable PPTX
  tile geometry and content support.
- Theme capability declarations and clear diagnostics for unsupported themes.

### Deferred

- Draggable dividers, unless selected as a first-release requirement.
- Splitting a selected tile and preserving a manually authored split tree.
- Nested tiling, more than six tiles, arbitrary coordinates, overlapping areas,
  floating elements, and freeform canvas editing.
- Content-aware or AI-based placement and implicit tile creation from headings.
- New automatic text-fitting behavior or automatic content rewriting.
- Full PPTX parity with arbitrary HTML, custom shortcodes, and theme decoration.

## 5. Authoring contract

Use explicit tile blocks in Markdown and composition settings in front matter.
Proposed syntax:

```markdown
---
title: Why this market
layout: content
tiling:
  preset: auto
---

{{< tile id="opportunity" >}}
## Opportunity

A large market with fragmented distribution.
{{< /tile >}}

{{< tile id="traction" >}}
## Traction

- 12 enterprise customers
- Three active markets
{{< /tile >}}

{{< tile id="economics" >}}
## Economics

![Unit economics](economics.png)
{{< /tile >}}
```

The number of blocks determines tile count. Do not require a second count field.
Desktop's "Create 3 tiles" action inserts three ordinary editable blocks.
An explicitly authored empty tile retains its space until removed.

IDs must be nonempty and unique within the slide. Preserve them through reorder
and content edits. Headings remain Markdown content rather than tile boundaries.

Example composition customization:

```yaml
tiling:
  preset: primary-stack
  primary: opportunity
  placement: left
  ratio: [2, 1]
  gap: medium
```

This assigns roughly two-thirds of usable width to the primary area and stacks
the remaining tiles in the supporting area, accounting for spacing. Ratios express
relative space, not absolute pixels.

Keep the initial configuration constrained. The implementation contract must
document which presets accept placement, ratios, and alignment settings, their
defaults, and invalid combinations. Do not expose arbitrary CSS through this API.

Recommended syntax boundaries:

- `tiling` explicitly enables the feature for a content layout.
- Tile blocks are top-level body content; nested tiles are invalid.
- Outside tile blocks, permit whitespace and comments. Reject other body content
  with a source diagnostic rather than silently losing or relocating it.
- Tile blocks without tiling enabled produce a diagnostic.
- Preserve existing notes and include behavior. Specify how included tile content
  maps to source before enabling structured editing of that content.

## 6. Arrangement and customization

All arrangements fill the usable body area after theme-owned slide regions and
spacing have been accounted for. The resolver must be deterministic for the same
source, settings, and theme contract.

### Automatic preset

| Tile count | Proposed arrangement |
| --- | --- |
| 1 | Full body |
| 2 | Equal columns |
| 3 | Primary tile left, two tiles stacked right |
| 4 | Equal 2 × 2 grid |
| 5–6 | Balanced two-row arrangement |

Source order determines placement unless an explicit primary ID overrides it.
Without an explicit primary, the first tile is primary where applicable. The
implementation plan must freeze exact five-tile placement and reading order before
shipping; this proposal does not define an arbitrary hole or spanning behavior.

In auto mode, adding a fourth tile changes the three-tile composition to a grid.
Removing it reverses the change. This is intentional rebalancing.

### Explicit presets

- `columns`: divide the available width among all tiles.
- `rows`: divide the available height among all tiles.
- `grid`: balanced grid with documented count-specific placement.
- `primary-stack`: keep a primary area beside or above/below supporting tiles;
  divide the supporting area among the remaining tiles.

An explicit preset retains its identity when tile count changes. Invalid counts,
missing primary references, invalid ratios, and unsupported combinations produce
actionable validation errors. Never silently change an explicit preset.

Support relative track sizing appropriate to each preset, primary placement,
theme-defined gap choices, and constrained content alignment. Stable source order
must also provide a sensible reading order. Existing column shortcodes remain
supported; no automatic migration is required.

## 7. Content fitting and diagnostics

Resizing tile areas does not guarantee that their content will fit. Preserve theme
typography by default and report overflow rather than silently deleting content,
cropping media, or shrinking all text.

- Images default to contain behavior; cropping requires an explicit choice.
- Any existing opt-in fitting behavior must respect theme minimum font sizes and
  scaling bounds. It must not conceal a tile overflow diagnostic.
- Diagnostics identify slide, tile ID, source location where available, output or
  viewport profile, and overflow direction.
- Desktop identifies the affected tile and supports locating its source.
- Headless auditing reports equivalent findings using the canonical rendered output.
- Structural validation works without a browser. Accurate rendered overflow
  measurement uses browser-backed auditing and reports an unavailable environment
  explicitly when it cannot run.

An empty tile is an intentional area, not automatically an error. A tiled slide
with no tiles is invalid. Removing the last tile requires an explicit desktop
choice to remove tiling or replace its content; never discard content implicitly.

## 8. Desktop requirements

Provide controls to create a tiled composition, add/remove/reorder tiles, select a
preset and primary tile, adjust placement and proportions, and inspect diagnostics.
Preserve undo/redo and use the existing source conflict workflow.

Every authored change must become ordinary Markdown/front matter. The app must
preserve unrelated source, unknown fields, notes, assets, and comments where
possible. It must reconcile dirty editor buffers before structural operations;
an operation must not overwrite unsaved or externally changed content.

Controls belong in the trusted editor UI. The rendered preview remains unprivileged
and receives no native mutation capabilities. Resizing the app window scales the
fixed-canvas preview without changing tile arrangement. Published mobile reflow is
a separate theme profile and must not change fixed-canvas export geometry.

If draggable dividers are added later, they persist the same proportional settings
available to CLI authors. They introduce no desktop-only layout state.

## 9. CLI and output requirements

- `margo build` and `margo serve` consume tiled source through the canonical engine.
- HTML composition does not require a browser, a desktop session, network access,
  or an OS window manager.
- Browser dependencies for auditing, PDF, and thumbnails remain explicit.
- Any tile mutation commands reuse the Go operations used by desktop and provide
  complete non-interactive arguments. Exact command grammar is an implementation
  decision; normal source authoring does not depend on these commands.
- Interactive HTML and print HTML use the same resolved composition. PDF preserves
  its placement through the print pipeline.
- Editable PPTX uses shared structural geometry for supported text and images.
  Unsupported components produce clear diagnostics rather than apparent parity.

## 10. Theme compatibility and architecture

Go owns source interpretation, validation, structural composition, revisions, and
safe mutations. Themes own markup, CSS layout, typography, spacing, and decoration.
Rust owns native lifecycle and preview placement; JS owns buffers and interaction.

The default theme implements the first tiling contract. Other themes explicitly
declare tiling support and supported presets. Unsupported-theme diagnostics apply
only when a slide requests the feature. Existing projects need no migration.

Reuse the existing content-region and layout-audit concepts. Do not add narrow
engine flags that force theme presentation decisions, or duplicate parsing and
rendering in the desktop app.

## 11. Acceptance criteria

1. Source alone can define one through six tiles with deterministic placement.
2. Adding/removing tiles reallocates space according to the documented preset;
   reordering preserves IDs and content.
3. Placement, proportions, spacing, and alignment work through supported source
   settings and desktop controls.
4. Equivalent source and options yield equivalent CLI and desktop compositions.
5. Titles, chrome, annotations, and footers remain outside the tiled body region.
6. HTML and print/PDF preserve intended geometry. Supported PPTX text and images
   remain editable inside corresponding tile areas.
7. Invalid source and unsupported capabilities produce actionable diagnostics.
   Rendered overflow identifies the affected tile without hidden content loss.
8. Desktop operations preserve unrelated source and handle undo, dirty buffers,
   revision conflicts, and external edits safely.
9. App/window resizing preserves the fixed-canvas composition and preview isolation.
10. Untiled slides and existing column usage retain their behavior.

Verify committed fixtures, a fresh scaffold, dense-content examples, and actual
desktop behavior on macOS and Omarchy. Build configuration alone is not native
platform acceptance evidence.
