# Evidence collection

Collect the minimum evidence needed to explain the failure. Keep deck content local unless the user authorizes sharing it.

## Build and preview failures

- Capture the exact command and complete stderr.
- Identify the source file and line from Margo diagnostics when present.
- Check margo.yaml and the active theme only when they are relevant to the error.
- Use a committed fixture or a minimal copy when reproducing a suspected Margo defect.

## Rendering and export failures

Treat interactive HTML, print HTML, and PDF as separate checkpoints. A defect in PDF can arise from print HTML, the browser or PDF runtime, fonts, or the theme's print rules. Compare the earliest incorrect artifact before changing source.

Do not edit generated output to test a theory. Change source only after the user approves a repair.
