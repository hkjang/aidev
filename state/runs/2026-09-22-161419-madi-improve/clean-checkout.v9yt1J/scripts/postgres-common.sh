#!/usr/bin/env bash
# Shared helpers for the operator-run PostgreSQL backup scripts.
set -euo pipefail
umask 077

madi_fail() { printf 'madi: %s\n' "$1" >&2; exit 1; }
madi_require() { command -v "$1" >/dev/null 2>&1 || madi_fail "Required command is missing: $1"; }
madi_passwordless_dsn() {
  # Passwords belong in .pgpass / PGPASSFILE, not shell history or process args.
  [[ "$1" =~ ^postgres(ql)?://[^/@:]+@[^/]+/[^?]+(\?.*)?$ ]] || \
    madi_fail 'Use an explicit password-free URI: postgresql://user@host:port/database?sslmode=verify-full'
  [[ "$1" != *$'\n'* && "$1" != *$'\r'* ]] || madi_fail 'The DSN must be a single line.'
  [[ ! "$1" =~ [\?\&](password|sslpassword)= ]] || madi_fail 'Store passwords in .pgpass / PGPASSFILE.'
}
madi_plain_path() {
  [[ "$1" != *$'\n'* && "$1" != *$'\r'* ]] || madi_fail 'Paths must not contain line breaks.'
}
madi_version() {
  local madi_script_root
  madi_script_root="$(cd -- "$(dirname -- "${BASH_SOURCE[0]}")/.." && pwd)"
  [[ -f "$madi_script_root/VERSION" ]] || madi_fail 'Keep the VERSION file with these scripts.'
  tr -d '\r\n' < "$madi_script_root/VERSION"
}
