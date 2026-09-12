# Margo Deck Agent Guide

This directory is a Margo deck project. Read this file before changing deck content, themes, or build artifacts.

## Working rules

- margo.yaml is the deck configuration entry point. It selects the active theme and configured outputs.
- Keep slide content in Markdown bundles under slides/<slide-id>/index.md. Put slide-local assets beside that file and shared assets under assets/.
- Use an existing archetype before inventing a new slide shape. Archetypes create authoring files; layouts render slides.
- Keep presentation markup, CSS, layouts, partials, and theme shortcodes under themes/<theme-name>/. Keep the Margo engine generic.
- Do not edit generated dist/ output. Change source files and run Margo again.
- Treat imported themes as trusted project code. Templates, shortcodes, and JavaScript can run during local builds and previews.

## Agent resources

Read .agents/README.md to choose the right repository-local skill. The deck-authoring skill covers normal slide work, upgrades, and packaging. The theme-authoring skill covers custom theme work. The brand-theme skill covers evidence-backed brand-theme creation. The GitHub Pages skill covers deployment setup. The error-triage skill covers reported build, rendering, preview, and export failures.
