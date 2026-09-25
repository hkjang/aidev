moyro v0.2.37

Reminder creation now reports storage faults as 500 instead of disguising
them, on both routes that create one. `POST
/api/v4/posts/{post_id}/remind_me` and its official-shape alias `POST
/api/v4/users/{user_id}/posts/{post_id}/reminder` folded every `posts.Get`
failure into 404 `api.reminder.create.not_found`, so a connection failure, an
unusable `posts` table or a cancelled context looked to a client exactly like
"that post was deleted" — a permanent answer no client retries, so a
transient fault silently cost the user their reminder. `posts.Service.Get`
already returns `pgx.ErrNoRows` for a genuinely absent row, so the 404 is now
kept only for that sentinel and for the documented `nil, nil` result, with
the error id, message and response shape unchanged because clients match on
them, and every other failure answers 500 `api.reminder.create.app_error`.
The alias also dropped the `channels.IsMember` error and answered 403 "not a
channel member" for it, while the native route already split that out as 500
`api.reminder.create.member_check`; the alias now uses the native route's id
and shape, so the same resource answers the same input the same way through
either path. There are no new error ids, no migration and no service-contract
change, so other `posts.Get` callers are unaffected, and there is no web
change. A Postgres integration test pins both faults against a real schema,
the real posts, channels and reminders services and a running `ws.Hub` with
the production audience resolver.
