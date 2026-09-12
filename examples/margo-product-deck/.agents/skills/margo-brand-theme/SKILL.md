---
name: margo-brand-theme
description: Turn approved brand guidelines, websites, or visual references into an editable Margo theme. Use only for brand-theme creation or refreshes. Do not create theme files until the user approves the proposed design direction.
---

1. Read AGENTS.md, references/workflow.md, references/evidence.md, references/component-contract.md, and the local margo-theme-authoring skill before acting.
2. Classify design confidence, source authority, asset/packaging rights, and supported component specificity separately. Do not claim high fidelity without authoritative evidence.
3. Audit inherited theme styles before copying a base theme: border radii, shadows, gradients, card/pill controls, and media treatment. Choose a blank theme or a reset layer, then record intentional exceptions.
4. Produce a concise design-direction review with the brand component contract, proposed shortcode/API list, token sheet, layout storyboard, output-support matrix, assumptions, and licensing constraints. Stop for explicit approval.
5. After approval, create an editable theme under themes/<brand>/ with layouts, partials, shortcodes, assets, theme.yaml, and a PPTX contract. Keep presentation decisions in templates and CSS. Do not add brand-specific behavior to the Margo engine.
6. Verify each branded selector against rendered HTML or the owning template, scan final CSS for non-compliant inherited styles, and prove every custom component visually in HTML and print HTML/PDF.
7. For each component, implement a PPTX-safe equivalent or record a visible fallback/unsupported decision before delivery. Classify PDF runtime failures by owner, preserve unaffected artifacts, and do not change output configuration without approval.
8. Deliver the evidence manifest, brand component contract, design decision log, proof matrix, exact runtime limitations, and a .margot archive.
