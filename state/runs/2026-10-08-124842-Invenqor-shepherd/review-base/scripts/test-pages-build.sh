#!/usr/bin/env bash
# Build every published document with the GitHub Pages production build command.
# Usage: bash scripts/test-pages-build.sh /absolute/path/to/build-output
set -euo pipefail

root=$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)
output=${1:?Pass a build output directory outside the repository}
mkdir -p "$output"
output=$(CDPATH= cd -- "$output" && pwd)

# This image contains github-pages 232, Jekyll 3.10.0 and Liquid 4.0.4.
# Keep the source read-only and the generated site outside the checkout.
docker run --rm \
  --user "$(id -u):$(id -g)" \
  --entrypoint /usr/local/bundle/bin/github-pages \
  --volume "$root/docs:/github/workspace/docs:ro" \
  --volume "$output:/github/workspace/_site" \
  --env JEKYLL_ENV=production \
  --env PAGES_REPO_NWO=hkjang/Invenqor \
  ghcr.io/actions/jekyll-build-pages@sha256:6791ebfd912185ed59bfb5fb102664fa872496b79f87ff8b9cfba292a7345041 \
  build --verbose --source /github/workspace/docs --destination /github/workspace/_site

test -s "$output/index.html"
test -s "$output/RELEASE_NOTES_v0.2.45.html"
echo 'GitHub Pages build and published page checks passed.'
