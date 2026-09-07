# Standalone Theme Archives PRD

## 1. Decision

Margo will support portable, standalone theme archives with the `.margot` extension.

A `.margot` file packages exactly one theme. A deck may contain many themes, but a packaging run always requires the author to choose one. The archive can then be imported into another Margo deck without Git, a registry, or network access.

`.margo` remains the extension for complete, editable deck-project archives. `.margot` means Margo theme archive. The two formats are not interchangeable.

## 2. Problem

Themes are currently portable only by copying a theme directory or publishing it to a Git repository and installing it with `margo theme add`. That works for developers, but it is a poor handoff path for a branded theme created inside a deck project:

- A source deck can contain several themes, while a project archive contains all of them and the deck content.
- Copying a directory is easy to get wrong and does not validate the result at import time.
- Git adds repository setup and network access where the author only needs to hand over one approved theme.

Teams need a way to move one self-contained theme from a source deck to a new or existing deck. The receiving deck must retain local ownership of the installed files and build offline.

## 3. Goals

- Package one selected theme from a Margo deck into a portable `.margot` file.
- Import that archive into another valid Margo deck under `themes/<name>/`.
- Make theme selection explicit when a source deck has more than one theme.
- Preserve the selected theme's complete contract, including templates, partials, shortcodes, assets, metadata, and optional PPTX contract.
- Validate archives before installing them and leave the receiving deck unchanged if validation fails.
- Keep Git installation and standalone archive import as separate, understandable workflows.

## 4. Non-goals

- Packaging an entire deck. Use `.margo` project archives for that.
- Packaging multiple themes in one `.margot` file.
- A theme registry, central cache, signing service, marketplace, or automatic update channel.
- Theme inheritance, composition, or merging archive contents into an existing theme.
- Treating a `.margot` archive as a runnable deck or allowing `margo <archive.margot>`.
- Sandboxing theme templates, shortcodes, or JavaScript.

## 5. Primary User Flows

### Package a theme from a source deck

From a deck containing `themes/brand/`:

```bash
margo theme pack brand
# writes ../brand.margot by default
```

The author may choose an output path:

```bash
margo theme pack brand --output /path/to/brand-v1.2.0.margot
```

The theme may also be selected with a flag:

```bash
margo theme pack --theme brand --output /path/to/brand-v1.2.0.margot
```

`<theme-name>` and `--theme <name>` are equivalent selectors. Supplying both with different values is an error.

When the selector is omitted in an interactive terminal, Margo presents the installed themes and marks the deck's active theme. This is a choice prompt, not a default that silently packages the active theme. In non-interactive use, omitting the selector fails and lists the available theme names.

Margo validates the selected theme before writing the archive. It reports the archive path, theme name, and version after success.

### Import a theme into a receiving deck

From the root of an existing Margo deck:

```bash
margo theme import /path/to/brand-v1.2.0.margot
```

This installs the archive's theme under `themes/brand/`. The import does not change `margo.yaml` or activate the theme automatically. Theme selection stays explicit.

An author may select a different local installation name:

```bash
margo theme import /path/to/brand-v1.2.0.margot --name client-brand
```

An optional activation flag provides the intentional shortcut:

```bash
margo theme import /path/to/brand-v1.2.0.margot --activate
```

`--activate` updates the receiving deck's active-theme setting only after a successful import. It uses the final installed name, including a name supplied with `--name`.

### Relationship to Git themes

Git remains the right path for a shared theme that has a maintained repository and an update lifecycle:

```bash
margo theme add https://example.com/brand-theme.git --ref v1.2.0
```

`.margot` is the offline handoff path. It vendors a fixed copy of the theme into the target deck. An archive-installed theme cannot be refreshed with `margo theme update`; Margo must say that it was installed from an archive and should be re-imported or replaced with a Git-installed theme.

## 6. Archive Format

`.margot` uses a versioned ZIP-based container, separate from the `.margo` project-archive format.

Each archive contains:

```text
margot-theme-archive.yaml
theme.yaml
layouts/
partials/
shortcodes/
assets/
archetypes/              # when provided by the theme
pptx/                    # when provided by the theme
```

The selected theme directory is the archive root. The archive must not include the source deck's `margo.yaml`, slides, deck assets, output, other themes, or source-project metadata.

`margot-theme-archive.yaml` includes, at minimum:

- archive format version
- theme name and version from `theme.yaml`
- the minimum compatible Margo version
- archive creation timestamp
- a SHA-256 checksum of the packaged theme payload

The manifest describes the archive. `theme.yaml` remains the authoritative theme contract and is validated using the normal theme validation rules.

### Inclusion and exclusion rules

The pack command includes all ordinary files beneath the selected theme root that the theme contract may use. It excludes:

- `.git/`
- generated output and local caches
- `.margo` and `.margot` archives
- backup directories
- `.DS_Store`
- symbolic links

Margo rejects symbolic links rather than following them. This prevents an archive from silently including files outside the chosen theme.

## 7. Import Rules and Safety

Margo treats a theme archive as trusted code, not as passive design data. Theme templates, shortcodes, and JavaScript can run during local build or serve. The CLI and documentation must warn users not to import `.margot` files from untrusted sources.

Before writing to the receiving deck, Margo must:

1. Confirm that the command runs inside a valid Margo project.
2. Confirm the `.margot` extension and open the archive as the theme-archive format.
3. Validate every member before extraction: relative paths only, no path traversal, no absolute paths, no duplicate unsafe paths, and no symlinks.
4. Enforce bounded file-count and expanded-size limits appropriate to theme assets.
5. Read and validate the archive manifest, including format and compatibility versions.
6. Verify the payload checksum.
7. Extract to a temporary staging directory inside the receiving deck's `themes/` directory.
8. Validate the staged theme against the normal theme contract.

Only then may Margo move the staged directory into `themes/<installed-name>/`. If any check fails, the existing deck and themes remain unchanged.

If `themes/<installed-name>/` already exists, import fails. It never overwrites, merges, or deletes an existing theme. The author can use `--name` to choose a free local name.

## 8. Provenance and Updates

After import, Margo writes local installation provenance into the installed `theme.yaml` without changing the archive's original theme content:

```yaml
source:
  type: archive
  archive_format: margot
  archive_sha256: <payload checksum>
  imported_theme_name: brand
  imported_theme_version: 1.2.0
```

The installed theme does not depend on the original archive path. The target deck contains the full vendored theme and remains usable after the archive is moved or deleted.

If the packaged theme already contains Git provenance, Margo may preserve it as original-source metadata for inspection. It must not treat that metadata as authority to run `margo theme update` unless the user explicitly converts or reinstalls the theme through the Git workflow.

## 9. CLI Requirements

Add two commands:

```text
margo theme pack [<theme-name> | --theme <name>] [--output <path>]
margo theme import <archive.margot> [--name <local-name>] [--activate]
```

Requirements:

- `theme pack` requires a Margo project root and an installed local theme.
- The pack selector is always explicit through an argument, a flag, or an interactive selection.
- `theme import` requires a Margo project root and exactly one `.margot` path.
- Both commands must produce action-oriented errors that identify the file, theme, and next valid action where practical.
- `theme list` must identify archive-installed themes and show their imported version and checksum prefix when provenance is available.
- `theme update <name>` must reject archive-installed themes with a clear re-import or Git-install instruction.
- `--activate` must update configuration atomically with installation. If configuration update fails, Margo restores the pre-import state.

The existing `margo theme add <repo>` contract remains Git-only. Margo must not guess whether an arbitrary path is a Git repository or an archive.

## 10. Acceptance Criteria

1. Given a source deck with `themes/default/` and `themes/brand/`, `margo theme pack` prompts the user to choose one and never packages both.
2. Given the same deck in non-interactive mode, `margo theme pack` without a selector fails and lists `default` and `brand`.
3. `margo theme pack brand` creates a `.margot` archive containing the validated `brand` theme only, with no deck slides or other theme directories.
4. Importing that archive into a clean deck places it at `themes/brand/`, validates it, and leaves the active theme unchanged unless `--activate` is supplied.
5. `--name client-brand --activate` installs at `themes/client-brand/` and sets `client-brand` as the active theme.
6. Importing into an occupied target directory fails without altering the existing theme.
7. Archives with traversal paths, symlinks, missing or invalid manifests, checksum mismatches, excessive size, or invalid theme contracts fail before a theme becomes visible in the destination.
8. A deck using an imported theme builds and serves with no network access and after the original `.margot` file is deleted.
9. `margo theme update` reports that an archive-installed theme is not updateable through Git.
10. `.margo` continues to package full decks, and `.margot` cannot be opened, unpacked, or served as a deck project.

## 11. Documentation Requirements

Update the authoring guide with:

- a comparison of `.margo`, `.margot`, and Git theme installation
- package and import examples
- the selection behavior for source decks with multiple themes
- the explicit activation behavior
- a clear trust warning for imported themes
- guidance to use Git for ongoing shared-theme maintenance and `.margot` for fixed offline handoffs

## 12. Open Product Questions

- Should Margo support signing `.margot` archives once third-party theme distribution becomes common?
- Should a future `theme replace` command provide an explicit, backed-up replacement flow for an installed archive theme?
- Should the manifest contain a human-readable description and release notes for `theme list` and import confirmation?

These questions do not block the initial archive and import workflow.
