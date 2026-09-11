---
name: margo-github-pages
description: Configure or review GitHub Pages deployment for a Margo deck. Use when the task concerns the generated Pages workflow, release-gated publishing, manual dispatch, or the published deck artifact. Do not use for normal deck content or theme work.
---

1. Read AGENTS.md before changing deployment files.
2. Confirm the deck is in a Git repository and identify the Margo release version already verified for the deck.
3. Generate the workflow with margo deploy github-pages --margo-version <version>. Do not replace an existing workflow unless the task explicitly authorizes it.
4. The generated workflow deploys dist/html on v* tags and manual dispatch. It does not publish directly from the local machine or configure repository settings.
5. After generation, inspect the workflow and build the deck locally. Tell the user to commit the workflow and set the repository Pages source to GitHub Actions.
