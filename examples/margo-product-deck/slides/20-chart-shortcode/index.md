---
title: Shortcodes are reusable and theme-defined
order: 20
layout: content
section: Themes
---

## When Markdown alone cannot express the visual.

{{< chart caption="Example monthly source activity and cumulative deck adoption" height="260px" width="100%" >}}
type: bar
data:
  labels: ["Jan", "Feb", "Mar", "Apr", "May", "Jun", "Jul", "Aug", "Sep", "Oct", "Nov", "Dec"]
  datasets:
    - label: "Core components"
      data: [24, 33, 28, 37, 31, 26, 35, 30, 39, 34, 42, 36]
      backgroundColor: "#69e3b1"
      stack: "components"
      order: 2
    - label: "Deck-specific components"
      data: [8, 5, 11, 7, 13, 9, 6, 12, 8, 14, 10, 7]
      backgroundColor: "#82b8ff"
      stack: "components"
      order: 1
    - type: line
      label: "Cumulative deck adoption"
      data: [31, 75, 111, 163, 204, 239, 288, 328, 383, 430, 491, 541]
      borderColor: "#f0fff8"
      backgroundColor: "#f0fff8"
      pointBackgroundColor: "#f0fff8"
      pointRadius: 3
      tension: 0.3
      yAxisID: "y1"
      order: 0
options:
  responsive: true
  maintainAspectRatio: false
  plugins:
    legend:
      position: bottom
      labels:
        color: "#f0fff8"
  scales:
    x:
      stacked: true
      grid:
        display: false
      ticks:
        color: "#f0fff8"
    y:
      stacked: true
      beginAtZero: true
      grid:
        color: "rgba(240, 255, 248, 0.14)"
      ticks:
        color: "#f0fff8"
    y1:
      position: right
      beginAtZero: true
      grid:
        drawOnChartArea: false
      ticks:
        color: "#f0fff8"
{{< /chart >}}
