moyro v0.2.34

Bulk sidebar category updates are now all-or-nothing. `PUT
/api/v4/users/{uid}/teams/{tid}/channels/categories` called
`sidebar.Update` once per array item, each committing its own transaction,
so a missing or foreign category id (404), an invalid sorting value (400)
or a storage fault (500) on a later item left the earlier items committed
behind the error response and no `sidebar_categories_updated` event told
other tabs about the half-applied state. A new `UpdateMany` runs the whole
array on one transaction and re-reads the result in input order after the
commit, while the single-item `Update`, the sentinel errors, the 404/400/500
mapping, the error ids, the response shape and the event name are
unchanged; `getSidebarCategory` now answers 404 only for a missing row and
500 for a storage fault. Four Postgres integration tests pin the rollback,
the input-order result and the 404/500 split.
