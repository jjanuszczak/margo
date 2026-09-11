---
title: Publish versioned decks to GitHub Pages
order: 13
layout: content
section: Workflow
---

## Ship the same deck you reviewed.

{{< columns >}}
{{< column >}}
### Generate the workflow

```sh
margo deploy github-pages \
  --margo-version vX.Y.Z
```

Margo writes a transparent GitHub Actions workflow that pins the release version your deck passed review on.
{{< /column >}}
{{< column >}}
### Release or dispatch

```sh
git tag vX.Y.Z
git push origin vX.Y.Z
```

The workflow builds `dist/html` and deploys it on `v*` tags. You can also run it on demand from GitHub Actions.
{{< /column >}}
{{< /columns >}}

{{< callout tone="info" >}}
Commit the generated workflow, then select GitHub Actions as the repository's Pages source. Margo configures the deck workflow, not your repository settings.
{{< /callout >}}
