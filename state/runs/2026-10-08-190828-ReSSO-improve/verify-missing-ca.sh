#!/usr/bin/env bash
set -euo pipefail
probe_dir="/mnt/c/Users/USER/projects/aidev/state/runs/2026-10-08-190828-ReSSO-improve"
missing_ca="$probe_dir/missing-ca-probe"
phase="${1:?phase required}"
test ! -e "$missing_ca"
probe_status=0
RESSO_TEST_CERT_DIR="$missing_ca" bash scripts/test-services.sh >"$probe_dir/ca-probe-$phase.stdout" 2>"$probe_dir/ca-probe-$phase.stderr" || probe_status=$?
printf '%s\n' "$probe_status" >"$probe_dir/ca-probe-$phase.status"
if [ "$probe_status" -eq 0 ]; then
  printf 'FAIL: missing CA preparation exited 0; expected nonzero (stdout bytes: %s)\n' "$(wc -c <"$probe_dir/ca-probe-$phase.stdout")"
  exit 1
fi
test ! -s "$probe_dir/ca-probe-$phase.stdout"
grep -F "$missing_ca/ca.crt" "$probe_dir/ca-probe-$phase.stderr" >/dev/null
grep -F 'not a readable regular file' "$probe_dir/ca-probe-$phase.stderr" >/dev/null
grep -F 'Mounts' "$probe_dir/ca-probe-$phase.stderr" >/dev/null
grep -F 'docker inspect resso-test-ldaps' "$probe_dir/ca-probe-$phase.stderr" >/dev/null
grep -F 'RESSO_TEST_CERT_DIR' "$probe_dir/ca-probe-$phase.stderr" >/dev/null
test ! -e "$missing_ca"
printf 'PASS: missing CA preparation exited %s, stdout empty, path/cause/recovery diagnostics present\n' "$probe_status"
