---
title: A small, deliberate CLI surface
order: 7
layout: content
section: Workflow
---

## Start fast. Keep the project readable.

{{< columns >}}
{{< column >}}
### Create and author

```sh
margo new roadmap
margo new slide launch-plan
```

Use archetypes when a slide needs a known shape.
{{< /column >}}
{{< column >}}
### Preview and share

```sh
margo serve
margo build
```

Preview as you write, then produce static output for a release, a review, or an archive.
{{< /column >}}
{{< /columns >}}

{{< callout tone="info" >}}
The happy path is intentionally narrow. It should not require a custom build system or a separate publishing stack.
{{< /callout >}}
