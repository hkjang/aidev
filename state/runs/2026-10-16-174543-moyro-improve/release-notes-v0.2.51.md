moyro v0.2.51

A batch reaction hydration whose reaction read fails now refuses with a server
error instead of answering 200 with that post missing from the map, which is how
this endpoint spells "no reactions here". This is a server behavior fix and the
only product change in the tag. There is no schema change, no migration, no new
route, no new error id and no web source change. One production Go file changes,
and the route gains its first fault-injection test.

`postsByIDsReactions` in `compat_wave_handlers_late.go` implements a deliberate
visibility filter: the reactions for a post in a channel the caller cannot see
are dropped from the response rather than 403'ing the whole batch. That filter
was three bare `continue`s, and the last of them covered something else
entirely. When `h.reactions.ListForPost` returned an error the post was left out
of `out` and the handler fell through to its normal success path, so a database
fault arrived at the client as a 200 whose body was byte-identical to the body a
post with no reactions produces. The emoji a user had just added vanished from
the thread, and nothing in the response said why. The 500 for this condition,
`api.reaction.list.app_error`, already existed and was already reachable —
`listReactions` at `handlers.go:2503` returns it for the identical call on the
single-post route — so the batch route was the one place where the same failure
was silently absorbed.

The fix returns that existing 500 from that one branch and ends the handler
there. No error id and no status code is introduced, the 200-id cap, the request
decoding, the bare-object response shape and the success path are untouched, and
a genuinely unreacted post still comes back as an absent key.

The other two `continue`s are not touched. `ChannelForPost` and `IsMember`
answering negatively is the visibility filter itself — an invisible or deleted
post is a legitimate miss, not a fault — and they are pinned by preserved
assertions instead. Telling a fault apart from a legitimate miss on those two
calls needs a policy decision about what a batch endpoint owes a caller whose
membership lookup failed, and that is a different change, deliberately not made
here.

What an operator should expect is a fault becoming visible, not a contract
change. A deployment whose monitoring treated this route as one that does not
fail will begin to see 500s from it during a database incident, where it
previously saw 200s carrying silently incomplete maps. A caller that retries on
500 now retries instead of caching an empty reaction set.

The route also gains its first fault-injection test, 185 lines in
`posts_ids_reactions_fault_postgres_test.go`, on the isolated-schema PostgreSQL
harness the package already uses. It drives a chi router built on the production
route pattern through `store.Migrate` and the real `reactions` and `channels`
services, and injects the fault inside the isolated schema by replacing
`reactions` with a view that divides by zero for one `post_id` while leaving
`posts` and `channel_members` healthy, so `ChannelForPost` and `IsMember`
succeed and `ListForPost` is the only call that fails. The test is therefore
aimed at the one branch that changed.

Verified on this release tree in the order `release.yml` runs it, against a real
PostgreSQL instance: `go test -race -p 1 -count=1 ./...` at 51 packages ok, 10
without test files and 0 failures, nothing served from the test cache, with
`httpapi` at 64.672s rather than the fraction of a second it takes when the
database is absent, which confirms the database-backed tests — the new reaction
fault regression among them — executed instead of skipping; a targeted verbose
run of that regression reports all nine of its subtests passing. As at the
previous tag the checksum-pinned Mattermost plugin fixtures were fetched on this
machine against their four pinned checksums and all four
`MOYRO_TEST_*_PLUGIN_ARCHIVE` variables were set, so `pluginhost` passed at
67.056s and a targeted verbose run took
`TestMattermostArchivesInstallRunAndSurviveRestart` to `--- PASS` rather than a
skip; the release job is not the first place those run. `go build ./...` and
`go vet ./...` clean, `plugin-sdk/go` tests and vet clean, and `govulncheck`
v1.7.0 reporting this code affected by 0 vulnerabilities, which is the gate the
previous tag's toolchain bump was made for and which holds here unchanged. In
`webapp`, a clean `npm ci`, then `npm audit --omit=dev --audit-level=high` and
`npm audit --audit-level=high` both at 0 vulnerabilities as of 2026-10-16, then
`npm test` at 49 files and 224 tests on Node v22.23.1, then `npm run typecheck`
and `npm run build`. `node scripts/verify-pages.mjs` passes at 5 HTML pages, 47
product screenshots, 16 navigable settings/admin routes and 7 brand PNGs, and
`bash scripts/check-source-sizes.sh` passes. `git diff --check` is clean and the
commit this tag points at touches no Go file.

The three gates the release job owns were run on this machine against this tag's
commit rather than deferred to the tag push. The `linux/amd64` image build
succeeded with `VERSION`, `COMMIT`, `BUILD_DATE` and `SOURCE_DATE_EPOCH` derived
from commit a7b87e6e30ff82b4099679832c1bb7a05601e853 exactly as the workflow
derives them, and the built image carries `linux/amd64` with the
`org.opencontainers.image.version` label at `v0.2.51` and the revision label at
that commit. `scripts/verify-product-ui.sh` passed, reporting all REST contract
checks passed and then 77 of 77 Playwright chromium tests passing in 4.2m across
the SSO/OIDC, product-pages, plugin-compatibility, accessibility and
visual-regression specs, with the plugin fixtures supplied from this machine.
`scripts/verify-release.sh` passed against the exported archive with
`EXPECTED_COMMIT` set to that commit, verifying offline startup, web UI,
version, bootstrap login and restart from `moyro-v0.2.51.tar.gz`, built as
`docker save` piped through `gzip -n -9` exactly as the workflow exports it.

The release job builds and uploads the offline archive itself and creates the
GitHub release from the pushed tag, so no asset is attached by hand here. The
archive it publishes is rebuilt from this commit in the job, and the local
export above was made only to run the verification gate.
