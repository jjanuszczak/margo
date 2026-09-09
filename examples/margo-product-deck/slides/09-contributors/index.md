---
title: A contribution model with boundaries
order: 10
layout: content
section: Workflow
---

## Make the right layer responsible for the job.

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
