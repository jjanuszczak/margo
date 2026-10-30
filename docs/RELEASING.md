# Releasing Margo

Git tags are the release trigger and GitHub Releases are the canonical source
for the CLI and desktop downloads. The CLI remains independently installable;
the desktop app includes its own Go engine. The Homebrew tap and `go install`
remain CLI installation paths.

## Desktop app packaging

Each `v*` tag also builds experimental desktop release artifacts from that exact
checkout:

- macOS disk image (`.dmg`; architecture is included in the asset name)
- x86_64 Arch Linux/Omarchy portable folder archive (`.tar.gz`)

Both include a Go engine built from the same tag. The Arch archive keeps the app
and engine side by side and includes launch/runtime instructions. It requires
WebKitGTK 4.1 from the host; Chromium is optional and supplies slide thumbnails
and PDF export. The CLI archives continue to cover macOS, Linux, and Windows.

The tag workflow tests the Go engine and desktop UI, builds CLI and desktop
artifacts in parallel, verifies a combined SHA-256 manifest, then publishes one
GitHub Release after all package jobs succeed. macOS signing/notarization and
automatic updates are not configured. Omarchy runtime acceptance is still
required; see the [desktop validation boundaries](../desktop/README.md#current-validation-boundaries).

## Before tagging

Run the full test suite and a local package build:

```bash
env GOCACHE="$PWD/.gocache" go test ./...
VERSION=0.3.0 ./scripts/release.sh
```

The package script creates archives for macOS, Linux, and Windows plus a
`checksums.txt` file. Verify the checksums before publishing.

## Publish a release

Use an annotated semantic-version tag:

```bash
git tag -a v0.3.0 -m "Release v0.3.0"
git push origin v0.3.0
```

The `Release` workflow tests the tag, builds all CLI and desktop artifacts,
verifies their combined checksums, and publishes the GitHub Release with
generated notes. It also builds and publishes the Margo product deck to GitHub
Pages. The separate `Deploy Margo product deck` workflow provides the on-demand
equivalent. Tags containing a prerelease suffix, such as `v0.3.0-rc.1`, are
supported.

## Homebrew

After a stable release, update the personal tap formula with the new source
tag and checksum. Open a pull request in `jjanuszczak/homebrew-margo`, then
run:

```bash
brew install --build-from-source ./Formula/margo.rb
brew test margo
brew audit --strict --formula ./Formula/margo.rb
```

## Installation paths

Users can install the CLI with Homebrew, download a matching CLI archive from
GitHub Releases and verify `checksums.txt`, or run:

```bash
go install github.com/jjanuszczak/margo/cmd/margo@v0.3.0
```

Desktop users can download the macOS disk image or the experimental Arch Linux
archive from the same release. The app does not require a separate CLI install.

PDF export may require a separately installed Chrome or Chromium runtime; the
CLI binary does not bundle that browser dependency.
