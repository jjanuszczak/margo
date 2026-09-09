# Theme contract

Themes live under themes/<theme-name>/. The usual entry points are theme.yaml, assets/, layouts/, partials/, and shortcodes/.

- Layouts own slide and deck markup.
- Partials are reusable template fragments.
- Shortcodes are content components expanded inside slide Markdown.
- Theme assets contain CSS, fonts, JavaScript, and images needed by the theme.

Deck-local partials and shortcodes override theme entries with the same name. Keep the engine responsible for generic parsing, validation, asset resolution, and model shaping. Keep presentation-specific markup, class composition, and styling in theme templates and CSS.

Select the active theme in margo.yaml. Create a default-inspired theme before editing a shared default theme. Use Git installation for maintained shared themes; use .margot import for a fixed offline handoff. Do not import a theme from an untrusted source.
