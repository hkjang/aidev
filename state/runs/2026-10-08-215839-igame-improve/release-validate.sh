#!/usr/bin/env bash
set -Eeuo pipefail
run_dir=/mnt/c/Users/USER/projects/aidev/state/runs/2026-10-08-215839-igame-improve
run_check() {
  local check_name="$1"
  shift
  if timeout 1800 "$@" > "$run_dir/release-$check_name.log" 2>&1; then
    printf '%s PASS\n' "$check_name"
  else
    result=$?
    printf '%s FAIL (exit %s)\n' "$check_name" "$result"
    exit "$result"
  fi
}
run_check contract make check-contract
run_check lint make lint
run_check test make test
run_check race make test-race
run_check build make build
run_check test-db make test-db 'DSN=postgres://igame@127.0.0.1:15432/igame?sslmode=disable&search_path=public,igame_test_extensions'
run_check catalog-race env 'IGAME_TEST_DSN=postgres://igame@127.0.0.1:15432/igame?sslmode=disable&search_path=public,igame_test_extensions' go test ./internal/api -run '^TestCatalogList' -race -count=5 -v
run_check sdk-audit npm --prefix sdk/gamehub-js audit --audit-level=low
run_check web-audit npm --prefix web audit --audit-level=low
run_check govuln-install env GOBIN="$run_dir/release-tools" go install golang.org/x/vuln/cmd/govulncheck@v1.6.0
run_check govulncheck "$run_dir/release-tools/govulncheck" ./...
run_check diff git diff --check
