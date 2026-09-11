# Margo error-triage workflow

Start with a concise incident report, not a speculative fix.

## Establish the facts

- Read the deck AGENTS.md and use the narrowest applicable Margo skill.
- Record the Margo version, operating system and relevant runtime, exact command, exact stderr, selected theme, requested output, and relevant source path.
- Reproduce with the smallest safe command. Do not edit dist/ or reset source files to make the error disappear.

## Classify ownership

- Deck-owned: Markdown, front matter, margo.yaml, assets, includes, or deck-local shortcodes.
- Theme-owned: layouts, partials, theme shortcodes, CSS, theme assets, or print templates.
- Margo-owned: reproducible CLI or engine behavior that persists in a clean or committed fixture.
- Environment-owned: browser, Chromium, fonts, PDF tooling, filesystem permissions, or port availability, unless Margo configuration caused it.

When evidence supports more than one owner, say so. Do not call an environment problem a Margo defect merely because Margo exposed it.

## Return this shape

## Finding

State the most likely owner and problem in one sentence.

## Evidence

List only the commands, diagnostics, output comparison, and source locations needed to support the finding.

## Recommended next action

Name the smallest safe repair or the next diagnostic check. State whether it needs user approval.

## Upstream status

State whether existing Margo issues were searched and whether a matching report exists.
