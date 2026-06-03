# Chart Shortcode Implementation Plan

## Objective

Add a default-theme `chart` shortcode powered by Chart.js so deck authors can embed charts directly in slide Markdown using simple structured data, with authoring ergonomics modeled on the Hugo Blowfish `chart` shortcode.

This plan is specifically for:

- author-facing shortcode behavior
- shared render/runtime integration in generated HTML
- default-theme support
- regression coverage and documentation

This plan is not for:

- arbitrary third-party chart plugins
- server-side chart rendering
- editable chart GUIs
- PPTX-native chart export

## Product Intent

The target experience is:

```md
{{< chart >}}
type: bar
data:
  labels: ["Tomato", "Blueberry", "Banana", "Lime", "Orange"]
  datasets:
    - label: "# of votes"
      data: [12, 19, 3, 5, 3]
{{< /chart >}}
```

The author writes structured chart configuration in the shortcode body, `margo` validates and serializes it, and Chart.js renders the chart in generated HTML.

This should feel Blowfish-like in practice:

- shortcode body contains chart configuration
- multiple chart types are supported
- chart options live with the chart, not in deck-level JS
- the theme/runtime handles the rendering

## Reference Behavior

Blowfish documents `chart` as a shortcode that:

- uses Chart.js for rendering
- accepts chart parameters between shortcode tags
- supports different chart styles
- relies on Chart.js configuration for deeper customization

Source references:

- Blowfish shortcode docs: [Shortcodes · Blowfish](https://blowfish.page/docs/shortcodes/)
- Blowfish chart samples: [Charts · Blowfish](https://blowfish.page/samples/charts/)
- Chart.js configuration model: [Configuration | Chart.js](https://www.chartjs.org/docs/latest/configuration/)

## Recommended Authoring Model

### Recommendation

Use YAML-in-shortcode-body as the canonical Margo authoring format.

Example:

```md
{{< chart caption="Quarterly broker growth" >}}
type: line
data:
  labels: ["Q1", "Q2", "Q3", "Q4"]
  datasets:
    - label: "Active brokers"
      data: [12, 18, 27, 35]
      borderColor: "#b88b2d"
      tension: 0.3
options:
  plugins:
    legend:
      display: true
{{< /chart >}}
```

### Why YAML instead of raw JS object literals

Blowfish examples use JavaScript-like object syntax, but Margo should prefer validated structured data over raw JS strings.

YAML is the right compromise because it:

- stays author-friendly inside Markdown
- is close to the “simple structured data” goal
- supports arrays and nested objects cleanly
- can be validated in Go
- avoids `eval`/raw inline-JS parsing

This is slightly stricter than Blowfish’s literal example format, but better aligned with Margo’s product principles and validation model.

## Shortcode Surface

### Body

Required structured chart config with these top-level fields:

- `type`
- `data`
- optional `options`
- optional `plugins`

### Parameters

Recommended initial shortcode params:

- `caption`
- `class`
- `height`
- `width`
- `id`

Notes:

- `id` should be optional; the renderer should generate a stable unique ID if absent
- `height` and `width` should control container sizing, not bypass Chart.js config
- `caption` should be rendered in theme markup, consistent with other shortcodes

### Validation Rules

MVP validation should require:

- non-empty shortcode body
- body parses as YAML object
- top-level `type` exists and is a string
- top-level `data` exists and is an object
- `data.datasets` exists and is a non-empty array
- `type` is one of the supported chart types for MVP

Supported MVP chart types:

- `bar`
- `line`
- `pie`
- `doughnut`
- `radar`

Optional later extension:

- `polarArea`
- `bubble`
- `scatter`
- mixed charts

## Rendering Architecture

### Go-side responsibilities

Go should:

- parse shortcode params
- parse shortcode body YAML
- validate required structure
- serialize the validated config to canonical JSON
- render safe HTML scaffolding with no raw JS config interpolation

Go should not:

- interpret Chart.js semantics deeply beyond basic shape validation
- reimplement chart rendering logic
- encode chart-type-specific presentation rules in the engine

### Theme/runtime responsibilities

The default theme should:

- render a chart wrapper
- render a `<canvas>` target
- embed serialized chart JSON in a safe data carrier
- load Chart.js only when at least one chart exists
- instantiate charts on page load

Recommended markup shape:

```html
<figure class="shortcode-chart">
  <div class="shortcode-chart-frame" style="...">
    <canvas id="margo-chart-1"></canvas>
    <script type="application/json" class="shortcode-chart-config">
      { ...validated config json... }
    </script>
  </div>
  <figcaption class="shortcode-chart-caption">...</figcaption>
</figure>
```

Recommended runtime shape:

- scan `.shortcode-chart`
- read sibling JSON config
- create `new Chart(canvas, config)`

This avoids inline JS object templating and keeps serialization boundaries clear.

## Asset Strategy

### Recommendation

Vendor Chart.js into theme assets and stage it into built output.

Do not use a CDN as the primary runtime path.

### Why

Margo’s product requirements already establish that built HTML should work offline after generation. CDN-only Chart.js would violate that.

Recommended approach:

- add a versioned Chart.js asset under theme assets or internal scaffold assets
- stage it with the rest of theme output
- reference it locally from deck HTML and print HTML when charts are present

## HTML And Print Behavior

### Interactive HTML

Interactive HTML should:

- render charts after runtime initialization
- preserve readable fallback markup if JS does not execute
- respect responsive container widths

### Print HTML / PDF

Print HTML should also initialize Chart.js so browser-based PDF generation can capture the rendered canvas output.

Recommended approach:

- include the same local Chart.js asset in print HTML when charts are present
- initialize charts before print/PDF capture
- keep print layout sizing explicit so charts do not overflow slides

This is similar in spirit to the existing Mermaid print/runtime split.

## Theme Integration Plan

### Default theme

Add:

- `themes/default/shortcodes/chart.html`
- chart CSS in interactive deck layout
- chart CSS in print deck layout
- chart runtime bootstrapping in interactive deck layout
- chart runtime bootstrapping in print deck layout

### Scaffold

Mirror the default-theme implementation into scaffold generation in:

- `internal/scaffold/deck.go`

### Example decks

Update:

- `examples/reference-deck`
- optionally `examples/authoring-guide-deck`

The reference deck should include at least one chart slide used for regression coverage.

## Runtime Design Details

### Presence detection

Do not load Chart.js globally for all decks.

Recommended detection options:

1. Render a deck-level boolean during slide shaping when chart shortcode HTML is present.
2. Simpler MVP: let deck JS query for `.shortcode-chart` and early-return if none exist.

For asset loading, prefer server-rendered presence detection if practical so the script tag itself is omitted when unused.

### Error handling

If runtime initialization fails:

- leave the fallback frame visible
- log a console warning
- do not break slide navigation or other deck JS

### Fallback behavior

If JS does not run:

- the chart area should still reserve reasonable space
- a readable fallback should remain visible

Recommended fallback:

- render a `<pre>` block with the canonical JSON config behind a `shortcode-chart-fallback` class
- hide it only after successful chart initialization

This mirrors the current Mermaid fallback model and improves debugging without leaking raw source into final successful render.

## Validation Strategy

### Go-side validation

MVP validation should be structural, not exhaustive:

- required top-level fields
- supported `type`
- object/array shape sanity
- numeric `height`/`width` if provided

It should not attempt to fully validate every Chart.js option subtree.

### Runtime validation

Let Chart.js remain the final authority on deeper option correctness.

If Chart.js rejects a config:

- log the error
- keep fallback visible

## Testing Plan

### Unit tests

Add shortcode rendering tests for:

- valid chart shortcode body renders expected wrapper/config
- missing body errors
- invalid YAML errors
- missing `type` errors
- unsupported chart type errors

### HTML fixture tests

Add coverage in `internal/output/html` for:

- chart wrapper HTML appears in rendered output
- serialized JSON config is present
- chart runtime bootstrap is included when charts exist
- chart runtime bootstrap is omitted when charts do not exist

### Print HTML tests

Add coverage in `internal/output/printhtml` for:

- print artifact includes chart wrapper/config
- print artifact includes Chart.js runtime only when needed

### Manual QA

Verify in:

- `examples/reference-deck`
- a fresh scaffolded deck
- `examples/arca-investor-memo` only if desired for dogfooding, but not required for MVP

Manual checks:

- desktop HTML
- mobile HTML
- print HTML
- PDF generation with chart slides

## Documentation Plan

Update:

- `docs/AUTHORING_GUIDE.md`

Add:

- a simple bar chart example
- a line chart example
- notes on supported types
- notes on body syntax and validation expectations
- notes on fallback/print behavior if relevant

## Risks

### 1. Syntax drift from Blowfish

Risk:

- users may expect Blowfish’s JavaScript-like body syntax verbatim

Mitigation:

- document that Margo uses validated structured config
- keep examples concise and Blowfish-like in spirit
- explicitly call out YAML body format

### 2. Offline output regression

Risk:

- CDN-hosted Chart.js breaks offline HTML

Mitigation:

- vendor the runtime locally

### 3. PDF timing issues

Risk:

- charts may not finish rendering before PDF capture

Mitigation:

- reuse the existing print/runtime initialization pattern
- add explicit readiness handling if required during PDF export

### 4. Over-validation

Risk:

- trying to model all Chart.js options in Go slows progress and creates brittle validation

Mitigation:

- keep Go validation structural
- let Chart.js validate deep semantics at runtime

## Phased Execution

### Phase 1

- add shortcode template
- add Go-side YAML parsing and structural validation
- render chart wrapper plus JSON config

### Phase 2

- add local Chart.js asset
- add interactive runtime bootstrap
- add print runtime bootstrap

### Phase 3

- add reference-deck example slide
- add tests
- add authoring-guide documentation

### Phase 4

- manual QA for mobile and PDF
- tune sizing, fallback, and theme styling

## Recommended Review Questions

Before implementation starts, confirm:

1. Should the body syntax be YAML-only, or do we want to also support JSON?
2. Is Blowfish-like authoring ergonomics sufficient, or is literal JS-object compatibility a hard requirement?
3. Do we want the fallback to show the parsed config when JS fails, or should it degrade to a simpler placeholder/message?
4. Are the MVP chart types limited to `bar`, `line`, `pie`, `doughnut`, and `radar`, or should `scatter`/`bubble` be included immediately?

## Recommendation

Proceed with:

- YAML body syntax
- locally vendored Chart.js
- structural Go validation
- conditional runtime loading
- default-theme and print-theme support in the first implementation

That is the cleanest path that preserves Margo’s architecture while still delivering the Blowfish-style authoring experience the enhancement is aiming for.
