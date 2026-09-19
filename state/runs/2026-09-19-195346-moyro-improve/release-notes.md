moyro v0.2.33

Sidebar category writes are now validated and answer by cause.
`sidebar.Update` stored whatever `display_name` and `sorting` the client
sent — a blank name on a custom category, a sorting value outside
alpha/recent/manual, and a new name for the three default categories that
Mattermost never renames — and a missing, foreign-user or foreign-team
category id came back as 400 "no rows in result set" from `Update` and
`Delete` while `Get` said 404, with storage faults leaving as 400 too.
`Update` now checks sorting first (an empty value keeps the stored one),
locks the row to learn its type, refuses a blank name only for custom
categories and preserves the default names inside the UPDATE, so a rejected
payload leaves the row untouched; the service exposes `ErrNotFound` and
`ErrInvalid`, and the four write handlers share one mapping to 404, 400 or
500 with the existing error ids, response shape and `/api/v4` paths.
