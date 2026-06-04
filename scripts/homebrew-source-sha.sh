#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <tag>" >&2
  echo "example: $0 v0.1.0" >&2
  exit 1
fi

tag="$1"

if [[ "$tag" != v* ]]; then
  tag="v${tag}"
fi

url="https://github.com/jjanuszczak/margo/archive/refs/tags/${tag}.tar.gz"

echo "==> fetching $url" >&2
curl -fsSL "$url" | shasum -a 256 | awk '{print $1}'
