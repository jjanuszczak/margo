# Homebrew Packaging

This document is for maintainers setting up and shipping Homebrew support for `margo`.

## Recommended v1 Approach

For the current Homebrew implementation, use:

- a personal tap, not `homebrew-core`
- source builds from a tagged GitHub release tarball
- a formula update pull request after each tagged GitHub release

GitHub Releases are the canonical source for versioned binaries and source
archives. Homebrew is a convenience installation path, not the release
system itself.

Why:

- `margo` does not need `homebrew-core` review overhead yet
- Homebrew can build the CLI directly from source with the existing Go toolchain
- tagged releases provide stable source URLs and checksums

## Packaging Shape

Use a tap repository named:

- `jjanuszczak/homebrew-margo`

Put the formula in:

- `Formula/margo.rb`

Users should then be able to install with:

```bash
brew install jjanuszczak/margo/margo
```

Homebrew will automatically tap `jjanuszczak/homebrew-margo` when needed.

## Formula Strategy

The first formula should:

- download the GitHub source tarball for a tagged release
- build with Homebrew's Go conventions
- stamp the CLI version during build
- verify installation with `margo version`

The in-repo template lives at [packaging/homebrew/Formula/margo.rb](../packaging/homebrew/Formula/margo.rb).

## Release Checklist

For a release like `v0.1.0`:

1. Confirm the repo is in a releasable state and cut a Git tag using the `vX.Y.Z` format.
2. Build local archives if you want direct-download release assets:
   ```bash
   VERSION=0.1.0 ./scripts/release.sh
   ```
3. Create the GitHub release for the tag.
4. Compute the source tarball checksum:
   ```bash
   ./scripts/homebrew-source-sha.sh v0.1.0
   ```
5. Update `url`, `sha256`, and any version-specific text in the tap's `Formula/margo.rb`.
6. In the tap checkout, verify the formula:
   ```bash
   brew install --build-from-source ./Formula/margo.rb
   brew test margo
   brew audit --strict --formula ./Formula/margo.rb
   ```
7. Push the tap update.
8. After the tap is live, keep the main README install section aligned with the real command.

## Notes On Scope

Do not block the first Homebrew release on:

- bottled binaries
- GitHub Actions-based tap automation
- GoReleaser integration
- `homebrew-core` submission

Those can follow once the source-based install path is stable and the release cadence is clearer.
