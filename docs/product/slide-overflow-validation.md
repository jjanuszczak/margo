# Slide Overflow Validation

This document tracks the implementation of GitHub issue #68:
[detect, prevent, and resolve slide content overflow across responsive outputs](https://github.com/jjanuszczak/margo/issues/68).

The issue is intentionally being implemented in phases. Each phase should be
marked complete only after its code, tests, and representative deck checks are
complete.

## Phase status

- [x] Phase 1: browser-backed detection for interactive and print output
- [x] Phase 2: theme layout contracts and responsive profiles
- [x] Phase 3: bounded, theme-approved automatic fitting
- [ ] Phase 4: agent-facing diagnostics and semantic-change proposals
- [ ] Phase 5: explicit and boundary-aware semantic splitting

## Working rules

- Phase 1 is observation-only. It must not rewrite authored content or change
  layout behavior.
- Warnings must identify the artifact, rendering profile, slide, direction, and
  measured overflow where possible.
- Interactive HTML and print HTML are separate surfaces and must be audited
  independently.
- A responsive theme owns reflow and scrolling behavior. Margo owns measurement
  and reporting.
- Future phases must preserve the boundary between engine mechanics, theme
  presentation, and authored content.

## Verification log

| Phase | Implementation | Tests | Representative deck checks | Notes |
| --- | --- | --- | --- | --- |
| 1 | Complete | Complete | Complete | Detection only; warnings do not fail the build. Verified on the reference deck for interactive and print HTML. |
| 2 | Complete | Complete | Complete | Includes contract bootstrap for new themes and explicit migration for older themes; generated contracts are inferred and review-required. |
| 3 | Complete | Complete | Complete | Opt-in fitting is bounded by `min_scale`, re-measured after each adjustment, and preserves warnings when the floor is reached. |
| 4 | Pending | Pending | Pending | Agent proposals must not silently mutate semantic content. |
| 5 | Pending | Pending | Pending | Splitting must use explicit or safe semantic boundaries. |

## Phase 1 exit criteria

- The build can audit generated interactive HTML when HTML output is enabled.
- The build can audit generated print HTML when PDF or PNG output is enabled.
- The audit waits for fonts and images before measuring.
- The audit detects horizontal and vertical overflow, descendant content outside
  the slide boundary, and internal scroll overflow.
- Diagnostics identify the slide index and title when available.
- Missing or unusable browser support produces a clear warning rather than
  changing or blocking the build.
- Tests cover the measurement script, parsing, warning formatting, and the
  browser invocation boundary.
- The reference deck and its intentionally dense overflow cases have been
  checked manually.

## Phase 1 implementation notes

- Added `internal/layoutaudit`, which creates a temporary static audit copy and
  measures rendered slide bounds in headless Chrome.
- `margo build` audits `dist/html/index.html` and `dist/pdf/print.html` when
  those artifacts are enabled.
- Findings are warnings under `layout_overflow`; browser or audit failures are
  warnings and do not change or block the build.
- The audit consumes the DOM snapshot as soon as its result marker appears and
  terminates the temporary Chrome process, so preview timers cannot hold up the
  build.
- Full verification: `go test ./...`.
- Reference verification: `examples/reference-deck` builds successfully and
  reports overflow findings for both interactive and print output.

## Phase 2 implementation notes

- Added optional `layout_contract` metadata for canonical slide geometry,
  reserved theme space, semantic regions, and future fitting/splitting policy.
- Added optional `responsive.profiles` metadata with named viewport dimensions,
  `fixed_canvas` or `reflow` mode, and an explicit vertical-scroll allowance.
- Existing themes without the contract continue to use the Phase 1 desktop
  audit profile at 1920x1080.
- Interactive HTML is audited once per declared profile. Print HTML is audited
  against the declared `desktop` profile, or the first profile when no desktop
  profile is present.
- A reflow profile may explicitly permit vertical scrolling. That permission
  suppresses vertical overflow warnings only; horizontal overflow remains a
  warning.
- Contract validation is intentionally strict for malformed values, but the
  feature remains optional so existing themes are not broken during migration.
- No automatic fitting, font mutation, content rewriting, or slide splitting
  is part of this phase.
- New starter themes include a baseline contract. Existing themes can use
  `margo theme contract init [theme-name]` to generate one explicitly. The
  command refuses to overwrite an existing contract and marks generated values
  as `source: inferred` and `status: review_required`.
- Contract inference uses PPTX slide size, theme CSS aspect ratio, slide
  padding, body font size, layout files, and responsive media queries. These
  are conservative starting assumptions, not a substitute for theme review.
- RFC-wiki migration verification generated a contract for `rfc-corporate` and
  rebuilt the deck successfully. Desktop overflow findings were preserved;
  the inferred mobile reflow profile was audited with vertical scrolling
  explicitly permitted.
- Verification: focused theme, layout-audit, CLI, and scaffold tests pass;
  full-suite and representative deck checks pass for the implementation.

## Phase 3 implementation notes

- Regions with `overflow_policy: warn` remain observation-only. Automatic
  fitting requires an explicit `overflow_policy: fit` declaration.
- Fitting applies a generic region zoom to `.slide-body` or
  `.print-slide-body`, re-measures after each five-percent step, and stops at
  the declared `min_scale` floor.
- The same fitting behavior is injected into interactive HTML and print HTML,
  so preview, PDF, and PNG generation use the approved adjustment.
- A `layout_fit` informational diagnostic reports the resulting scale. Any
  overflow remaining at the floor is still reported as `layout_overflow`.
- Phase 3 does not alter Markdown, rewrite content, reduce semantics, or split
  slides. Those remain later phases.
- RFC-wiki test result: `Control hypotheses` fit at approximately 66% scale;
  `Interview map` reached its 65% floor and retained a 39px warning. Both
  interactive and print surfaces completed successfully.

### Phase 2 contract example

```yaml
layout_contract:
  slide:
    width: 1920
    height: 1080
  layouts:
    default:
      reserved:
        top: 72
        right: 72
        bottom: 72
        left: 72
      regions:
        - name: content
          role: body
          min_font_size: 18
          min_scale: 0.8
          overflow_policy: warn
          flexible: true

responsive:
  profiles:
    - name: desktop
      width: 1920
      height: 1080
      mode: fixed_canvas
    - name: mobile
      width: 390
      height: 844
      mode: reflow
      allow_vertical_scroll: true
```
