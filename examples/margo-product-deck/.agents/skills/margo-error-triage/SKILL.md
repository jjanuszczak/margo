---
name: margo-error-triage
description: Diagnose a reported Margo build, serve, rendering, or export failure. Use before proposing a repair, workaround, or upstream report. Do not use for ordinary deck editing or proactive reviews.
---

Use this skill only when a user reports a failure or unexpected output. It is demand-driven; it does not watch processes or invoke an agent from Margo.

1. Read AGENTS.md and identify the command, exact failure, Margo version, selected theme, requested output, and relevant source path.
2. Read references/triage.md. Classify the problem before suggesting a change.
3. Read references/evidence.md when collecting or reproducing evidence. For visual or export defects, compare interactive HTML, print HTML, and PDF when those outputs apply.
4. Read references/reporting.md only after evidence indicates a Margo bug and the user wants upstream action.
5. Read references/known-boundaries.md whenever ownership is unclear.

Do not edit source, configuration, themes, generated output, or external systems while diagnosing. Ask for explicit approval before making a change, uploading logs, or creating or commenting on an external issue.
