#!/usr/bin/env bash
set -euo pipefail
MADI_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$MADI_ROOT"
: "${MADI_BROWSER_POSTGRES_DSN:?An isolated browser-test PostgreSQL DSN is required}"
node -e 'const net=require("node:net"); const s=net.createServer();s.once("error",()=>{console.error("Port 8080 is occupied; refusing to test an existing service");process.exit(1)});s.listen(8080,"0.0.0.0",()=>s.close());'
MADI_TEST_OUTPUT="$MADI_ROOT/test-results"
mkdir -p "$MADI_TEST_OUTPUT"
MADI_TEMP_DIR="$(mktemp -d -t madi-browser.XXXXXXXX)"
MADI_TEST_PROCESS=""
cleanup() {
  if [[ -n "$MADI_TEST_PROCESS" ]]; then
    kill "$MADI_TEST_PROCESS" 2>/dev/null || true
    wait "$MADI_TEST_PROCESS" 2>/dev/null || true
  fi
  # The temporary build is intentionally retained for runner cleanup.
}
trap cleanup EXIT
go build -o "$MADI_TEMP_DIR/madi" ./cmd/madi
POSTGRES_DSN="$MADI_BROWSER_POSTGRES_DSN" \
BOOTSTRAP_ADMIN=admin@example.test \
BOOTSTRAP_ADMIN_PASSWORD='Browser-Test-Password-2026!' \
ENCRYPTION_KEY="$(openssl rand -base64 32)" \
  "$MADI_TEMP_DIR/madi" > "$MADI_TEST_OUTPUT/server.log" 2>&1 &
MADI_TEST_PROCESS=$!
for MADI_ATTEMPT in $(seq 1 60); do
  if curl --fail --silent http://127.0.0.1:8080/readyz >/dev/null; then break; fi
  if ! kill -0 "$MADI_TEST_PROCESS" 2>/dev/null; then
    printf 'Browser-test service exited; see test-results/server.log\n' >&2
    exit 1
  fi
  sleep 1
done
curl --fail --silent http://127.0.0.1:8080/readyz >/dev/null
# The native CI process has no permission to the container's default /var/lib
# storage directory. Configure the dedicated test path through the real admin API
# without adding a fifth application environment variable.
MADI_BROWSER_STORAGE="$MADI_TEMP_DIR/attachments" node --input-type=module <<'NODE'
import assert from 'node:assert/strict';
const base = 'http://127.0.0.1:8080/api/v1';
const login = await fetch(base + '/auth/login', {
  method: 'POST', headers: {'Content-Type': 'application/json', 'X-Madi-Request': '1'},
  body: JSON.stringify({email: 'admin@example.test', password: 'Browser-Test-Password-2026!'})
});
assert.ok(login.ok, `Bootstrap login failed: ${login.status}`);
const cookie = login.headers.getSetCookie().map(value => value.split(';')[0]).join('; ');
assert.ok(cookie, 'Bootstrap login did not issue a session');
const saved = await fetch(base + '/admin/settings', {
  method: 'PUT', headers: {'Content-Type': 'application/json', 'X-Madi-Request': '1', Cookie: cookie},
  body: JSON.stringify({storage_path: process.env.MADI_BROWSER_STORAGE})
});
assert.ok(saved.ok, `Test storage configuration failed: ${saved.status}`);
NODE
MADI_BROWSER_SUITES=(
  document-async document-foundations browser navigation document-read-mode
  database-advanced operations automation-browser notification-browser
  inbound-capture-browser storage-browser migration-browser spaces graph
  templates tasks tasks-html inbox discussion knowledge canvas plugins
  enterprise-browser transfer-browser pwa
)
MADI_BASE_URL=http://127.0.0.1:8080 \
MADI_TEST_EMAIL=admin@example.test \
MADI_TEST_PASSWORD='Browser-Test-Password-2026!' \
MADI_REGRESSION_SCREENSHOT_ROOT="$MADI_TEST_OUTPUT/regression-shared/screenshots" \
  node tests/regression-shared.mjs "${MADI_BROWSER_SUITES[@]}"
