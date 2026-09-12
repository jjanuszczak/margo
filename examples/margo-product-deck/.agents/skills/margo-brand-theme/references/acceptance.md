# Acceptance checklist

- Design direction, brand component contract, and proposed shortcode/API list received explicit approval before implementation.
- Inherited-style audit selected a blank theme or reset layer, and final CSS/rendered output confirms every remaining radius, shadow, gradient, card, or pill style is intentional.
- Every branded selector was checked against rendered HTML or its owning template.
- Every custom component has a proof slide and visible HTML and print HTML/PDF verification.
- HTML, print HTML/PDF, and PPTX were each checked against the proof deck; each component declares required, fallback, or unsupported behavior per output.
- HTML/PDF and PPTX compromises are recorded separately. PDF runtime failures preserve artifacts, identify theme/engine/environment ownership, and record the exact failure.
- Source manifest records design confidence, source authority, asset rights, and supported component specificity. Brand component contract, design decisions, proof matrix, and known limitations are delivered with the theme.
- No feature-specific brand logic was added to the Margo engine.
