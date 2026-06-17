---
title: Math QA
order: 9
layout: content
footer_text: Math Regression Check
---

## Block math should upgrade cleanly

{{< math caption="Reference-deck KaTeX smoke test" >}}
\operatorname{Var}(X) = \mathbb{E}[X^2] - \left(\mathbb{E}[X]\right)^2
{{< /math >}}

{{< math caption="Integral smoke test" >}}
\int_0^{2\pi} \sin(x)\,dx = 0
{{< /math >}}

- Keep the TeX source readable if KaTeX fails to load
- Render the upgraded equation in served and built HTML
- Preserve the same slide content in print HTML
