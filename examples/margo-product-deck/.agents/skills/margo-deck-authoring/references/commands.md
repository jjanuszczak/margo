# Margo commands

Run these from the deck root when margo is on your PATH. If it is not, replace margo with the path to the local binary.

~~~bash
# Build and preview
margo build
margo build --include-drafts
margo serve
margo serve --port 1414
margo clean

# Safely refresh Margo-managed agent guidance
margo upgrade --plan
margo upgrade --apply

# Configure GitHub Pages (requires a Git repository and a released Margo version)
margo deploy github-pages --margo-version v0.3.0

# Add deck content
margo new slide roadmap --archetype agenda
margo new note speaker-script --slide 02-why

# Create and manage themes
margo new theme custom
margo new theme minimalist --blank
margo theme add https://example.com/brand-theme.git --ref v1.2.0 --name brand
margo theme update brand
margo theme list

# Transfer one theme
margo theme pack brand --output ../brand.margot
margo theme import ../brand.margot --name client-brand --activate

# Transfer the complete editable deck
margo pack .
margo unpack ../my-deck.margo restored-deck
~~~

Use .margo for a complete editable deck project. Use .margot for one theme only. Imported themes are local copies and are not updated with margo theme update.
