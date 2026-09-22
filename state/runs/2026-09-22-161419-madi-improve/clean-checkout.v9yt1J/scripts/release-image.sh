#!/usr/bin/env bash
set -euo pipefail

MADI_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$MADI_ROOT"
MADI_VERSION="$(tr -d '\r\n' < VERSION)"
MADI_VERSION="${MADI_VERSION#v}"
if [[ ! "$MADI_VERSION" =~ ^[0-9]+\.[0-9]+\.[0-9]+([-.][A-Za-z0-9.]+)?$ ]]; then
  printf 'Invalid VERSION: %s\n' "$MADI_VERSION" >&2
  exit 1
fi
MADI_IMAGE="madi:v${MADI_VERSION}"
MADI_ARCHIVE="$MADI_ROOT/dist/madi-v${MADI_VERSION}.tar.gz"
mkdir -p "$MADI_ROOT/dist"
MADI_TEMP_ARCHIVE="$(mktemp "$MADI_ROOT/dist/.madi-image.XXXXXXXX.tar.gz")"
trap 'if [[ -f "$MADI_TEMP_ARCHIVE" ]]; then rm -- "$MADI_TEMP_ARCHIVE"; fi' EXIT
docker build --pull --build-arg "VERSION=${MADI_VERSION}" -t "$MADI_IMAGE" .
docker save "$MADI_IMAGE" | gzip -9 > "$MADI_TEMP_ARCHIVE"
gzip -t "$MADI_TEMP_ARCHIVE"
mv -- "$MADI_TEMP_ARCHIVE" "$MADI_ARCHIVE"
printf 'Image: %s\nArchive: %s\n' "$MADI_IMAGE" "$MADI_ARCHIVE"
sha256sum "$MADI_ARCHIVE"
