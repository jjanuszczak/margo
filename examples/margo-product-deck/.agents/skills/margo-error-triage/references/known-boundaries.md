# Ownership boundaries

Keep the Margo engine generic. Content and theme problems should be repaired in the deck or theme that owns them, not by adding feature-specific behavior to Margo.

- Do not modify generated dist/ output. Change the deck, theme, or Margo source and rebuild.
- Do not reset a theme or overwrite authored files as a diagnostic shortcut.
- Do not upload logs or deck material without explicit approval.
- Do not create, comment on, or subscribe to an external issue without explicit approval.
- If browser, font, or PDF tooling is implicated, report that uncertainty and give the next factual check instead of guessing at a source change.
