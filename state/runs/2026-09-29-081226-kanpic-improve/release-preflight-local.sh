#!/usr/bin/env bash
# Checks everything that must already be true before a release is published.
# Publishing is irreversible: a tag people have fetched cannot be quietly
# corrected, so the cheap checks run before the build rather than after it.
set -euo pipefail

release_version="${1:-}"
project_root='/home/hkjang/.cache/auto-improve-wt/kanpic-release'
failures=()

fail() { failures+=("$1"); }

if [[ ! "$release_version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "usage: $0 vMAJOR.MINOR.PATCH" >&2
  exit 2
fi

notes="$project_root/docs/releases/$release_version.md"
[[ -s "$notes" ]] || fail "릴리즈 노트가 없습니다: docs/releases/$release_version.md"

declared="$(grep -m1 '^VERSION=' "$project_root/README.md" | cut -d= -f2- || true)"
if [[ "$declared" != "$release_version" ]]; then
  fail "README.md의 VERSION이 다릅니다: '$declared' (릴리즈: $release_version)"
fi

if [[ -n "$(git -C "$project_root" status --porcelain)" ]]; then
  fail "커밋되지 않은 변경이 있습니다. 릴리즈 아티팩트가 커밋과 달라집니다."
fi

if git -C "$project_root" rev-parse -q --verify "refs/tags/$release_version" >/dev/null; then
  fail "태그 $release_version 이(가) 이미 있습니다."
fi

# Detached HEAD / no-push runner: upstream publication gate is deferred.
if (( ${#failures[@]} > 0 )); then
  echo "릴리즈를 중단합니다 ($release_version):" >&2
  for item in "${failures[@]}"; do echo "  - $item" >&2; done
  exit 1
fi

echo "preflight ok: $release_version"
