#!/usr/bin/env bash
# Writes synthetic fixtures only in an isolated random PostgreSQL schema.
set -euo pipefail
if [[ "${1:-}" != '--isolated-test-database' || $# != 1 ]]; then
  printf 'Usage: MADI_TEST_POSTGRES_DSN=... bash scripts/benchmark-readonly.sh --isolated-test-database\nUse a disposable PostgreSQL test database, never the service DSN.\n' >&2
  exit 2
fi
: "${MADI_TEST_POSTGRES_DSN:?A disposable PostgreSQL test DSN is required}"
MADI_ROOT="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$MADI_ROOT"
MADI_SCALE_BENCH=1 go test ./internal/server -run '^TestReadOnlyScaleBenchmark$' -count=1 -timeout=25m -v
