# Brand-to-theme workflow

## Inputs

Prefer official brand guides, logo kits, font licences, and design systems. A live website, existing deck, screenshots, product UI, and campaign imagery can supply directional evidence. Record source URLs, supplied files, access date, and usage restrictions. Do not package third-party assets unless the user supplied them or confirms use rights.

## Website component inventory

Before design review, inventory reusable website patterns, not only colors and type. Identify notices, feature media panels, content cards, CTAs, data containers, navigation-like controls, and media treatment. For each pattern, decide whether it becomes a layout, a shortcode, a CSS primitive, or is intentionally excluded. Include the proposed shortcode/API name, parameters, visual behavior, and HTML/PDF/PPTX support in the review.

## Inherited-style audit

Before using a base theme, inventory its border-radius, box-shadow, gradients, cards, pill controls, and image treatment. Use a blank theme when the brand diverges materially, otherwise add an explicit reset layer. Before delivery, scan final CSS and rendered output; every remaining rounded, elevated, or decorative style must be intentional and listed in the decision log.

## Required review gate

Before changing themes, present: the evidence confidence (high-fidelity reproduction or inferred brand expression), source authority, asset rights, audience and deck purpose, color and type tokens, hierarchy and density rules, image/data treatment, title/content/metric/image/section/closing storyboard, the component contract, proposed shortcode/API list, output-support matrix, inherited-style audit, and open assumptions. Wait for explicit approval.

## Build and handoff

Use the standard Margo layout vocabulary. Maintain one token system across CSS and PPTX theme metadata. Test a fixed proof deck with title, dense content, image, section, two-column comparison, metric, chart/table, quote, and closing slides. Every custom component needs a proof slide and visible HTML and print HTML/PDF verification. Verify each CSS selector against rendered markup or its owning template. Check interactive HTML first, print HTML/PDF second, and editable PPTX independently.

If PDF generation fails, preserve the HTML and print artifacts, classify the fault as theme, engine, or environment, validate unaffected outputs, ask before changing output configuration, restore any approved temporary configuration, and record the exact failure. Ship brand-sources.yaml, BRAND_COMPONENT_CONTRACT.yaml, BRAND_DECISIONS.md, a component proof matrix, known limitations, and a .margot archive.
