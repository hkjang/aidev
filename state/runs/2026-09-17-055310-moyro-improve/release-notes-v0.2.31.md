moyro v0.2.31

Sidebar categories now keep only the channels their owner can actually
see. `sidebar.Create` and `Update` stored whatever channel ids the client
sent, so ids of channels the user never joined, of another team, or
archived were written and merely hidden on read, and an unknown id leaked
a foreign-key error that let a caller probe which ids exist; the write now
joins the payload against live membership in a single `unnest` statement
that silently drops the rest, as Mattermost does, and category reordering
folds into one UPDATE that ignores another user's ids and pins a repeated
id to its first position.

