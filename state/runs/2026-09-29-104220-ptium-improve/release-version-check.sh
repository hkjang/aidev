set -euo pipefail
version=1.69.52
notes=docs/release-notes-v1.69.52.md
fail() { echo "✗ $*" >&2; exit 1; }

# The version is written in five places. A release whose manifest, schema or
# install note names another version is a release that lies about itself in the
# one place an operator reads it.
[[ "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+([.-][0-9A-Za-z.-]+)?$ ]] || fail "'$version' is not a release version."
grep -q "service.version: $version" api/openapi.yaml || fail "api/openapi.yaml does not say service.version: $version"
grep -q "image: ptium:$version" deploy/kubernetes.yaml || fail "deploy/kubernetes.yaml does not run ptium:$version"
check_the_note_names_one_version() {
    # The install note is the whole description of the install that reaches an
    # offline site, and it names the release a dozen times: the archive, its
    # checksum, the two loader scripts, the compose file, the manifest and the
    # image it tells the operator to inspect. Watching the archive line alone
    # let every other line go on naming the release before it — the accident
    # this file exists to stop, in the one file an operator reads. Bare version
    # numbers in the prose ("records written before 1.52.1") describe older
    # releases on purpose, so only ptium-<version> and ptium:<version> count.
    local note="docs/offline-deployment.md" named stale
    named="$(grep -noE 'ptium[-:][0-9][0-9A-Za-z.+-]*' "$note" || true)"
    [ -n "$named" ] || fail "$note no longer names the version anywhere."
    stale="$(printf '%s\n' "$named" | grep -vE "^[0-9]+:ptium[-:]${version//./\\.}([^0-9A-Za-z]|\$)" || true)"
    [ -z "$stale" ] || {
        printf '%s\n' "$stale" >&2
        fail "$note still installs another version on the lines above."
    }
}
check_the_note_names_one_version
[ -f "$notes" ] || fail "$notes is missing: a release says what changed."

# And the bundle has to say which version it is. The env sample and the compose
# file are copied out of the repository, and until they were stamped they named
# whatever version was current when somebody last edited them.
check_bundle_names_itself() {
    grep -q "^PTIUM_VERSION=$version\$" "dist/ptium-$version.env.example" ||
        fail "dist/ptium-$version.env.example does not set PTIUM_VERSION=$version"
    grep -q "ptium-\${PTIUM_VERSION:-$version}:latest" "dist/docker-compose.ptium-$version.yml" ||
        fail "dist/docker-compose.ptium-$version.yml does not default to $version"
}


if [ "${1:-}" = "bundle" ]; then check_bundle_names_itself; fi
