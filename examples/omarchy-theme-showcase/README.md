# Omarchy theme for Margo

This is a complete, deck-local Margo theme that brings the Omarchy palette
picker to slide decks. It uses the color values served by
[omarchy.org](https://omarchy.org/) on September 12, 2026, and borrows the
site's square controls, slim rules, terminal-style metadata, and high-contrast
type treatment.

Body copy uses the Omarchy site's `JetBrains Mono` stack. Headings use its
`Geist Variable` stack at regular weight. The local header icon comes from the
[Omarchy dashboard icon](https://cdn.jsdelivr.net/gh/homarr-labs/dashboard-icons/png/omarchy.png).
Its transparent cutout acts as an alpha mask, so the header mark uses the
active palette's brand color.
The title wordmark uses the supplied Omarchy artwork as an alpha mask, so it
keeps that exact shape while taking the active palette's color bands.

Build the included preview:

```bash
go run ../../cmd/margo build
```

To use it in another deck, copy `themes/omarchy/` into that deck's `themes/`
directory and activate it in `margo.yaml`:

```yaml
theme:
  name: omarchy
  palette: catppuccin
```

`palette` accepts: `catppuccin`, `catppuccin-latte`, `ethereal`, `everforest`,
`flexoki-light`, `gruvbox`, `hackerman`, `kanagawa`, `last-horizon`, `lumon`,
`lupine`, `matte-black`, `miasma`, `nord`, `osaka-jade`, `retro-82`,
`ristretto`, `rose-pine`, `solitude`, `tokyo-night`, `vantablack`, and `white`.

For a portable archive, run `margo theme pack omarchy` from this example deck.
