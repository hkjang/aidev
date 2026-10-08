#!/bin/bash
set -euo pipefail
run_dir=/mnt/c/Users/USER/projects/aidev/state/runs/2026-10-08-104905-ai-admin-improve
pg_name=ai-admin-release-20261008-pg
kc_name=ai-admin-release-20261008-keycloak
cleanup() {
  docker rm -f "$pg_name" "$kc_name" > "$run_dir/release-container-cleanup.log" 2>&1 || true
}
trap cleanup EXIT
docker run --detach --rm --name "$pg_name" --publish 127.0.0.1::5432 --env POSTGRES_HOST_AUTH_METHOD=trust --env POSTGRES_USER=ai_admin --env POSTGRES_DB=ai_admin_test postgres:16-alpine@sha256:cf78e76683b9ca8c5733cbbdce6c9262b45b6767934dd0a95e671f9a0fc20685 > "$run_dir/release-pg-start.log" 2>&1
docker run --detach --rm --name "$kc_name" --publish 127.0.0.1::8080 --volume "$PWD/internal/server/testdata/keycloak-e2e-realm.json:/opt/keycloak/data/import/ai-admin-e2e-realm.json:ro" quay.io/keycloak/keycloak:26.7.2@sha256:9d1f1b2b7261ff53c66cb1092dfcdc34a5fb77e81f9e6a6e75b8b6a795de8067 start-dev --import-realm > "$run_dir/release-keycloak-start.log" 2>&1
pg_port=$(docker port "$pg_name" 5432/tcp | cut -d: -f2)
kc_port=$(docker port "$kc_name" 8080/tcp | cut -d: -f2)
export TEST_POSTGRES_DSN="postgres://ai_admin@127.0.0.1:$pg_port/ai_admin_test?sslmode=disable"
export TEST_KEYCLOAK_ISSUER="http://127.0.0.1:$kc_port/realms/ai-admin-e2e"
for attempt in $(seq 1 90); do
  if docker exec "$pg_name" pg_isready -U ai_admin -d ai_admin_test >/dev/null && curl --fail --silent "$TEST_KEYCLOAK_ISSUER/.well-known/openid-configuration" >/dev/null; then
    break
  fi
  sleep 2
done
curl --fail --silent "$TEST_KEYCLOAK_ISSUER/.well-known/openid-configuration" >/dev/null
GOTOOLCHAIN=go1.26.6 go test -race -count=1 -v ./... > "$run_dir/release-go-test-race.log" 2>&1
GOTOOLCHAIN=go1.26.6 go build ./... > "$run_dir/release-go-build.log" 2>&1
printf 'Go race tests and build passed\n'
