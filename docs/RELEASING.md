# Releasing Margo

Git tags are the release trigger and GitHub Releases are the canonical source
for prebuilt binaries and checksums. The Homebrew tap and `go install` are
secondary installation paths for the same tagged source.

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

The `Release` workflow tests the tag, builds all archives, verifies checksums,
and publishes the GitHub Release with generated notes. Tags containing a
prerelease suffix, such as `v0.3.0-rc.1`, are supported.

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

Users can install with Homebrew, download a matching archive from GitHub
Releases and verify `checksums.txt`, or run:

```bash
go install github.com/jjanuszczak/margo/cmd/margo@v0.3.0
```

PDF export may require a separately installed Chrome or Chromium runtime; the
CLI binary does not bundle that browser dependency.
