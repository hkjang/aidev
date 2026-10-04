moyro v0.2.45

Bulk user delete now refuses a body it could not read instead of reporting the
refused batch as a success. `DELETE /api/v4/users` decoded its
`{"user_ids":[...]}` body with `_ = decodeCappedBody`, so a body over
`collectionBodyMaxBytes` — or one truncated mid-array — left the destination
zeroed and the handler answered 200 `{"status":"OK","count":0}` without writing a
single `audit.ActionUserBulkDelete` row. An administrative integration reads that
envelope as "the bulk delete was processed" and stops retrying, so the entire
batch was silently dropped behind a response that claimed success, and the audit
trail the endpoint promises was absent.

`request_body.go` already settled the rule for this shape of endpoint: dropping
part of a write is worse than refusing the write, so the write batches reject.
`bulkDeleteUsers` was the only write batch among the compat stubs that did not
follow it, and it failed in the worst direction, because the cap discarded the
whole batch rather than part of it and still answered 200.

The new `decodeOptionalCollectionBody` refuses an undecodable body exactly as
`decodeCollectionBody` does — 413 over the cap, 400 malformed — under the new
error id `api.user.bulk_delete.invalid_body`, and the refusal is written before
the audit loop runs. An absent body is still read as an empty batch, so a bodyless
`DELETE /users` keeps its unchanged 200 `count: 0`. `decodeCappedBody`,
`decodeCollectionBody`, `drainCappedBody` and `tooManyBatchItems` are untouched,
so the sixteen other stubs that deliberately tolerate a malformed body continue
to do so.

Operators integrating against this route should note the narrow contract change.
A caller that sent an oversized or malformed body and relied on the false 200 now
receives 413 or 400 and should retry with a body under the cap. A caller already
sending a well-formed body, or no body at all, sees a byte-identical response. The
webapp and the e2e suite do not consume this route and neither OpenAPI document
describes the refused-body case, so integrations outside the repository are the
only ones affected.

There is no schema change and no migration. There is one new error id, no new
route and no web change. One helper is added and one decode call changes.

The regression runs the production `router.go` group verbatim — the real
`requireRole` middleware, the real auth and audit services, an isolated
PostgreSQL schema — and pins the two new refusals alongside the four behaviors
that must not move: the byte-identical success envelope with one audit row per
id, the bodyless 200, the 400 `too_many` at 201 ids, and the non-administrator
403. It skips without a PostgreSQL DSN, so an `ok` from a run with no database
does not mean it executed.

Two changes in this tag carry no behavior. The settings serialization test
announced the second activation before recording it, which let the assertions
read the counter before the increment landed; the ordering is corrected and the
serialization it covers was never at fault. And the new helper's doc comment
described the bodyless tolerance as a published contract, which overstated it —
nothing documents that tolerance. The comment now records that no caller was
found to depend on it and that it is preserved conservatively rather than turned
into a 400 here.
