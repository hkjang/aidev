#!/usr/bin/env bash
set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "usage: $0 <version>" >&2
  exit 2
fi

release_version="${1#v}"
if [[ ! "$release_version" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]]; then
  echo "version must be semantic versioning (example: 1.0.0)" >&2
  exit 2
fi

release_commit="$(git rev-parse --short=12 HEAD)"
release_built_at="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
# 이미지 태그는 서비스명:v버전, 배포 파일은 서비스명-v버전.tar.gz 로 맞춘다.
image_tag="seaton:v${release_version}"
artifact="SeatOn-v${release_version}.tar.gz"

docker build \
  --platform linux/amd64 \
  --build-arg "VERSION=${release_version}" \
  --build-arg "COMMIT=${release_commit}" \
  --build-arg "BUILT_AT=${release_built_at}" \
  -t "$image_tag" \
  -t "seaton:latest" .
docker save "$image_tag" | gzip -9 > "$artifact"
echo "$artifact"
