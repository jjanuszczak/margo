# Brand component contract

Create this machine-readable contract and obtain approval before theme implementation. Every rule needs a source, rendered selector or shortcode, proof slide, and output behavior.

~~~yaml
geometry:
  container_radius: 0
  image_radius: 0
  decorative_shadows: prohibited
  gradients: prohibited
components:
  cards:
    treatment: square-bordered
    selector_or_shortcode: content-card
    proof_slide: 05-components
    outputs: { html: required, pdf: required, pptx: fallback }
  notices:
    treatment: blue-information
    selector_or_shortcode: notice
    proof_slide: 05-components
    outputs: { html: required, pdf: required, pptx: unsupported }
  ctas:
    treatment: coral-outline
    selector_or_shortcode: cta
    proof_slide: 05-components
    outputs: { html: required, pdf: required, pptx: fallback }
~~~

Include media treatment, information states, CTA behavior, cards, feature panels, data containers, and any brand-specific controls. Validation must flag a rule with no selector/shortcode, no proof slide, or no declared HTML/PDF/PPTX behavior. CSS rules that target selectors absent from rendered markup are defects, not completed styling.
