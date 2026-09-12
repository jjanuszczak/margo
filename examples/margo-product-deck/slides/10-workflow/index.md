---
title: A small deliberate CLI
order: 10
layout: content
section: Workflow
---

## Start fast. Keep the project readable.

{{< columns >}}
{{< column >}}
### Create and maintain

```sh
# new deck
margo new roadmap

# new slide
margo new slide launch-plan

# review managed guidance updates
margo upgrade --plan

# install the global brand-theme workflow
margo skills install brand-theme --scope user
```

Use archetypes when a slide needs a known **content** template. Review a safe scaffold update before applying it.
{{< /column >}}
{{< column >}}
### Preview and share

```sh
# preview with hot reloads
margo serve

# generate shareable formats
margo build
```

Preview as you write, then produce static output for a release, a review, or an archive.
{{< /column >}}
{{< /columns >}}
