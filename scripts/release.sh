#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${DIST_DIR:-$ROOT_DIR/dist/release}"
VERSION="${VERSION:-}"
GOCACHE_DIR="${GOCACHE:-$ROOT_DIR/.gocache}"
APP_NAME="margo"
PACKAGE_NAME="${APP_NAME}-${VERSION}"

if [[ -z "$VERSION" ]]; then
  echo "VERSION is required (for example VERSION=0.3.0)" >&2
  exit 1
fi
if [[ ! "$VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
  echo "invalid VERSION '$VERSION' (expected semver without the leading v)" >&2
  exit 1
fi

TARGETS=()
if [[ $# -gt 0 ]]; then
  TARGETS=("$@")
else
  TARGETS=(
    "darwin/arm64"
    "darwin/amd64"
    "linux/amd64"
    "linux/arm64"
    "windows/amd64"
  )
fi

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"
mkdir -p "$GOCACHE_DIR"

build_target() {
  local target="$1"
  local goos="${target%/*}"
  local goarch="${target#*/}"
  local archive_base="${APP_NAME}-${goos}-${goarch}"
  local work_dir="$DIST_DIR/$archive_base"
  local package_dir="$work_dir/$PACKAGE_NAME"
  local binary_name="$APP_NAME"

  if [[ "$goos" == "windows" ]]; then
    binary_name="${APP_NAME}.exe"
  fi

  mkdir -p "$package_dir/docs"

  echo "==> building $archive_base"
  (
    cd "$ROOT_DIR"
    GOCACHE="$GOCACHE_DIR" GOOS="$goos" GOARCH="$goarch" \
      go build \
      -trimpath \
      -ldflags="-s -w -X github.com/jjanuszczak/margo/internal/version.Version=$VERSION" \
      -o "$package_dir/$binary_name" \
      ./cmd/margo
  )

  cp "$ROOT_DIR/README.md" "$package_dir/README.md"
  cp "$ROOT_DIR/docs/AUTHORING_GUIDE.md" "$package_dir/docs/AUTHORING_GUIDE.md"
  cp "$ROOT_DIR/LICENSE" "$package_dir/LICENSE"

  if [[ "$goos" == "windows" ]]; then
    (
      cd "$work_dir"
      zip -rq "${archive_base}.zip" "$PACKAGE_NAME"
    )
    cp "$work_dir/${archive_base}.zip" "$DIST_DIR/${archive_base}.zip"
    echo "created $DIST_DIR/${archive_base}.zip"
  else
    (
      cd "$work_dir"
      tar -czf "${archive_base}.tar.gz" "$PACKAGE_NAME"
    )
    cp "$work_dir/${archive_base}.tar.gz" "$DIST_DIR/${archive_base}.tar.gz"
    echo "created $DIST_DIR/${archive_base}.tar.gz"
  fi
}

for target in "${TARGETS[@]}"; do
  if [[ "$target" != */* ]]; then
    echo "invalid target '$target' (expected GOOS/GOARCH)" >&2
    exit 1
  fi
  build_target "$target"
done

checksum_command=(shasum -a 256)
if command -v sha256sum >/dev/null 2>&1; then
  checksum_command=(sha256sum)
fi
(
  cd "$DIST_DIR"
  find . -maxdepth 1 -type f \( -name '*.tar.gz' -o -name '*.zip' \) -print0 |
    sort -z |
    xargs -0 "${checksum_command[@]}" > checksums.txt
)

echo
echo "release artifacts and checksums written to $DIST_DIR"
