---
title: Chart QA
order: 8
layout: content
footer_text: Chart Regression Check
---

## Chart.js charts should upgrade cleanly

{{< chart caption="Reference-deck Chart.js smoke test" height="280px" >}}
type: doughnut
data:
  labels: ["Markdown", "Themes", "HTML", "PDF"]
  datasets:
    - label: "Margo outputs"
      data: [30, 25, 25, 20]
      backgroundColor:
        - "#4db6ac"
        - "#74c0b8"
        - "#d7ccb6"
        - "#6d655c"
options:
  plugins:
    legend:
      position: bottom
{{< /chart >}}

- Keep the configuration readable if Chart.js fails to load
- Render the upgraded canvas in served and built HTML
- Preserve the same slide content in print HTML
