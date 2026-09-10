# Margo product deck

This is Margo's public product deck. It is a complete Margo project, including a deck-local `margo-brand` theme, source slides, assets, and speaker notes.

The source is deliberately part of the product story. It shows users how to structure a deck and gives contributors a compact acceptance fixture for the authoring, theme, output, and notes model.

## Preview and build

From the repository root:

```sh
go build -o ./bin/margo ./cmd/margo
cd examples/margo-product-deck
../../bin/margo serve
```

Build the static artifact with:

```sh
../../bin/margo build
```

Generated output is written to `dist/` and is not committed.

## Testimonial shortcode

The `margo-brand` theme includes a compact testimonial card. `quote` and `name` are required. Add `role` for attribution details, and use `avatar` plus `avatar_alt` when the testimonial needs a portrait.

```md
{{< testimonial
  quote="Margo keeps our story in Markdown, where the team can actually maintain it."
  name="Avery Chen"
  role="Product lead, Acme"
  avatar="assets/avery-chen.jpg"
  avatar_alt="Portrait of Avery Chen"
/>}}
```

## Column variants

Columns render as cards by default. Use `variant="plain"` when the column should keep its spacing but not its border or background.

```md
{{< columns >}}
{{< column variant="plain" >}}
Content without a card surface.
{{< /column >}}
{{< /columns >}}
```

## Release publishing

GitHub Pages is release-gated. The release workflow runs only for version tags, builds this deck at that tag, and deploys `dist/html` to the repository's GitHub Pages site.

Before creating a release tag, maintainers should:

1. Review slides that describe user-facing capabilities and commands.
2. Review the named speaker notes, especially the release checklist on “What ships today.”
3. Run the deck build and inspect the interactive HTML artifact.
4. Ensure GitHub Pages is configured to use GitHub Actions in the repository settings.

The release tag is the public deck's source of truth. Do not publish an untagged branch version.
