#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
DIST_DIR="${DIST_DIR:-$ROOT_DIR/dist/release}"
VERSION="${VERSION:-0.0.0-dev}"
GOCACHE_DIR="${GOCACHE:-$ROOT_DIR/.gocache}"
APP_NAME="margo"
PACKAGE_NAME="${APP_NAME}-${VERSION}"

TARGETS=()
if [[ $# -gt 0 ]]; then
  TARGETS=("$@")
else
  TARGETS=(
    "darwin/arm64"
    "darwin/amd64"
    "linux/amd64"
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
      -ldflags="-X 'margo/internal/version.Version=$VERSION'" \
      -o "$package_dir/$binary_name" \
      ./cmd/margo
  )

  cp "$ROOT_DIR/README.md" "$package_dir/README.md"
  cp "$ROOT_DIR/docs/AUTHORING_GUIDE.md" "$package_dir/docs/AUTHORING_GUIDE.md"

  if [[ "$goos" == "windows" ]]; then
    (
      cd "$work_dir"
      zip -rq "${archive_base}.zip" "$PACKAGE_NAME"
    )
    echo "created $DIST_DIR/$archive_base/${archive_base}.zip"
  else
    (
      cd "$work_dir"
      tar -czf "${archive_base}.tar.gz" "$PACKAGE_NAME"
    )
    echo "created $DIST_DIR/$archive_base/${archive_base}.tar.gz"
  fi
}

for target in "${TARGETS[@]}"; do
  if [[ "$target" != */* ]]; then
    echo "invalid target '$target' (expected GOOS/GOARCH)" >&2
    exit 1
  fi
  build_target "$target"
done

echo
echo "release artifacts written to $DIST_DIR"
