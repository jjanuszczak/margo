# Slide Overflow Validation

This document tracks the implementation of GitHub issue #68:
[detect, prevent, and resolve slide content overflow across responsive outputs](https://github.com/jjanuszczak/margo/issues/68).

The issue is intentionally being implemented in phases. Each phase should be
marked complete only after its code, tests, and representative deck checks are
complete.

## Phase status

- [x] Phase 1: browser-backed detection for interactive and print output
- [ ] Phase 2: theme layout contracts and responsive profiles
- [ ] Phase 3: bounded, theme-approved automatic fitting
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
| 2 | Pending | Pending | Pending | Add theme contract before automatic fitting. |
| 3 | Pending | Pending | Pending | Re-measure after every bounded adjustment. |
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
