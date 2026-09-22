#!/usr/bin/env bash
# Verify the built image on a Docker network with no external egress.
set -euo pipefail
MADI_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$MADI_ROOT"
MADI_IMAGE="${1:-madi:v$(tr -d '\r\n' < VERSION)}"
MADI_RUN_ID="madi-check-$(date +%s)-${RANDOM}"
MADI_NETWORK_ID=""
MADI_DB_CONTAINER=""
MADI_APP_CONTAINER=""
MADI_CLIENT_CONTAINER=""
MADI_TEMP_DIR="$(mktemp -d -t madi-image-check.XXXXXXXX)"
cleanup() {
  if [[ -n "$MADI_CLIENT_CONTAINER" ]]; then docker rm -fv "$MADI_CLIENT_CONTAINER" >/dev/null 2>&1 || true; fi
  if [[ -n "$MADI_APP_CONTAINER" ]]; then docker rm -fv "$MADI_APP_CONTAINER" >/dev/null; fi
  if [[ -n "$MADI_DB_CONTAINER" ]]; then docker rm -fv "$MADI_DB_CONTAINER" >/dev/null; fi
  if [[ -n "$MADI_NETWORK_ID" ]]; then docker network rm "$MADI_NETWORK_ID" >/dev/null; fi
}
trap cleanup EXIT
docker image inspect "$MADI_IMAGE" >/dev/null
test "$(docker image inspect --format '{{.Config.User}}' "$MADI_IMAGE")" = '10001:10001'
test "$(docker image inspect --format '{{index .Config.Entrypoint 0}}' "$MADI_IMAGE")" = '/usr/local/bin/madi'
docker run --rm --network none --read-only --cap-drop ALL --security-opt no-new-privileges \
  --entrypoint sh "$MADI_IMAGE" -c 'cd /usr/share/madi/sources && sha256sum -c SHA256SUMS >/dev/null && test -f tesseract/recipe/APKBUILD && test -f tesseract/tesseract-5.5.2.tar.gz && test -f pdfjs-liberation/liberation-fonts-1.07.4.tar.gz && test -f pdfjs-liberation/REBUILD.txt && test -f pdfjs-liberation/manifest.json && test -f /usr/share/tessdata/eng.traineddata && test -f /usr/share/tessdata/kor.traineddata'
printf 'PASS bundled corresponding sources and offline OCR model files.\n'
MADI_ARCH="$(docker image inspect --format '{{.Architecture}}' "$MADI_IMAGE")"
CGO_ENABLED=0 GOOS=linux GOARCH="$MADI_ARCH" go build -trimpath -o "$MADI_TEMP_DIR/image-smoke" ./tests/image-smoke
# Client binary has no embedded secrets and is retained for runner cleanup.
chmod 755 "$MADI_TEMP_DIR" "$MADI_TEMP_DIR/image-smoke"
# PostgreSQL is test infrastructure and never becomes a release attachment.
if ! docker image inspect postgres:17-alpine >/dev/null 2>&1; then docker pull postgres:17-alpine; fi
MADI_NETWORK_ID="$(docker network create --internal "$MADI_RUN_ID")"
MADI_DB_PASSWORD="$(openssl rand -hex 24)"
MADI_ADMIN_PASSWORD="$(openssl rand -hex 24)"
MADI_KEY="$(openssl rand -base64 32)"
MADI_DB_CONTAINER="$(docker run -d --network "$MADI_NETWORK_ID" --network-alias postgres \
  -e POSTGRES_DB=madi -e POSTGRES_USER=madi -e "POSTGRES_PASSWORD=$MADI_DB_PASSWORD" postgres:17-alpine)"
for MADI_ATTEMPT in $(seq 1 40); do
  if docker exec "$MADI_DB_CONTAINER" pg_isready -h 127.0.0.1 -U madi -d madi >/dev/null 2>&1; then break; fi
  sleep 1
done
docker exec "$MADI_DB_CONTAINER" pg_isready -h 127.0.0.1 -U madi -d madi >/dev/null
MADI_APP_CONTAINER="$(docker run -d --network "$MADI_NETWORK_ID" --network-alias madi \
  --cpus 2 --memory 4g --memory-swap 4g --pids-limit 512 \
  --read-only --tmpfs /tmp:rw,noexec,nosuid,size=2g --cap-drop ALL --security-opt no-new-privileges \
  -e "POSTGRES_DSN=postgres://madi:${MADI_DB_PASSWORD}@postgres:5432/madi?sslmode=disable" \
  -e BOOTSTRAP_ADMIN=admin@example.internal -e "BOOTSTRAP_ADMIN_PASSWORD=$MADI_ADMIN_PASSWORD" \
  -e "ENCRYPTION_KEY=$MADI_KEY" "$MADI_IMAGE")"
MADI_CLIENT_CONTAINER="$(docker create --network "$MADI_NETWORK_ID" \
  --read-only --cap-drop ALL --security-opt no-new-privileges \
  --mount "type=bind,src=$MADI_TEMP_DIR/image-smoke,dst=/image-smoke,readonly" \
  -e MADI_IMAGE_SMOKE=disposable-internal-network -e "MADI_IMAGE_PASSWORD=$MADI_ADMIN_PASSWORD" \
  -e "MADI_IMAGE_VERSION=$(tr -d '\r\n' < VERSION)" \
  --entrypoint /image-smoke "$MADI_IMAGE")"
docker start --attach "$MADI_CLIENT_CONTAINER"
test "$(docker inspect --format '{{.State.ExitCode}}' "$MADI_CLIENT_CONTAINER")" = 0
test "$(docker network inspect --format '{{.Internal}}' "$MADI_NETWORK_ID")" = true
test "$(docker exec "$MADI_APP_CONTAINER" id -u)" = 10001
docker inspect --format 'Verified limits: NanoCPUs={{.HostConfig.NanoCpus}} Memory={{.HostConfig.Memory}} ReadOnly={{.HostConfig.ReadonlyRootfs}}' "$MADI_APP_CONTAINER"
printf '\nPASS service image: internal network, four app variables, non-root/read-only runtime and full data recovery.\n'
