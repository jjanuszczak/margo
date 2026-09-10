---
title: How to contribute
order: 98
layout: content
section: How to Contribute
---

## Make the right layer responsible for the job.

Contribute to the core margo repo for the engine. Make your own theme and deck repos.

{{< columns >}}
{{< column >}}
### Engine

Parsing, validation, asset resolution, staging, model shaping, and reusable template primitives belong in Go.
{{< /column >}}
{{< column >}}
### Theme and deck

Markup, class composition, visual hierarchy, and component presentation belong in templates and CSS.
{{< /column >}}
{{< /columns >}}

{{< callout tone="info" >}}
Do not turn authoring into a second programming language. Prefer explicit conventions and small, useful extension points.
{{< /callout >}}
