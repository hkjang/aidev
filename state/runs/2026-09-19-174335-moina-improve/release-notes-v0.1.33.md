v0.1.33 releases the Moin edit storage-error fix merged after v0.1.32.
In updatePost the UPDATE posts statement treated a database error and an
affected-row count of zero as one branch, so a lost connection or a
constraint violation reached the user as 409 not_editable ("본인의 공개
Moin만 수정할 수 있습니다") and the operator had no signal that storage was
the problem.

The two cases are now separated the way deletePost already does it: a
database error answers 500 storage_error with the existing "Moin을 변경할
수 없습니다" message, and zero affected rows - someone else's Moin, an own
Remoin, an unpublished Moin - keeps the unchanged 409 not_editable with the
same message. api/openapi.yaml lists 400 and 409 on PATCH /posts/{postID}
next to the 403 the middleware produces (forbidden, invalid_csrf) and names
the 500 in the route description, as no route in that document lists 500.
No frontend, documentation or MCP code reads not_editable, so nothing else
changes. A new PostgreSQL integration test drives the real New() server
through the session cookie, CSRF and pgx: a test-only BEFORE UPDATE trigger
rejects a sentinel body so the UPDATE itself fails inside PostgreSQL, and the
four cases (another user's Moin 409, own Remoin 409, failed UPDATE 500, own
Moin 200) failed on the previous code for exactly the 500 case before the
fix and pass after it.

This commit itself carries only the version bump: VERSION, the OpenAPI info
block, the package manifests and their lockfiles, the SPA fallback version,
the Dockerfile build args, the OG card and the screenshot manifest, plus the
version strings the published docs, the Markdown guides and READMEs quote.
The PDF prints of the two guides are left as they are because the repository
carries no tool to regenerate them. The README release paragraph now names
the Moin edit storage-error separation.

Verified with make fmt, make check, go build, go test -race against a
throwaway PostgreSQL (32 integration tests) and go vet across the backend,
staticcheck clean, and the frontend at 250 passing tests, ESLint within its
40 warning budget, and a v0.1.33 production build.
