Every one of the six lookups behind /userinfo — the realm in the path, the
token's revocation state, the account, the session, and the realm and client
roles the roles scope asks for — refuses through one helper,
writeUserInfoUnavailable, with the same 500 server_error. That answer is the
right one: the token is sound and there is nothing for the relying party to
fix, so refusing with 401 invalid_token would have it throw away a credential
that still works and send the person back through login. What the helper did
not do was record which lookup broke anywhere but a server log line, so an
operator saw one 500 in resso_http_requests_total{route,status} and had to read
the log to learn whether the realm, the account or a role query was the one
that stopped. The three other errors_total series exist so that nobody has to
do that; userinfo was the endpoint left out of the set.

It is now counted under resso_userinfo_errors_total, whose one stage label
holds the six literals already written in the callers — realm,
revocation_state, user, session, realm_roles, client_roles — and nothing that
arrived with the request. The wiring is one line in front of the helper's log
line, so the helper's signature and all six call sites are untouched.

The response is unchanged to the byte: the 500 server_error and its body, the
existing log line userinfo could not judge the request it was given, the 401
invalid_token for an expired, forged or foreign token, for a realm or session
that is not there and for an account switched off, and the 400 for a token
named in the header and the body both. The ordinary refusals are not counted at
all. A bearer token needs no credential to send, so a series that moved on a
401 would be one anybody outside could raise this alert with; because they are
left out, this series moving is always a fault on this side.

Unlike recordUnjudgedIntrospection the helper does not filter out
store.ErrNotFound, and a comment says why: every caller has already decided
that a thing which is simply not there is a fact about the token and answered
401 before reaching here, so the filter would be code that never runs — and the
two role lookups draw no such distinction, so it would quietly stop counting a
fault this endpoint does refuse.

The operations guide's alert list gains the entry for this series, naming what
each of the six stages means, saying that the 500 is visible in the request
counter but that six unrelated lookups stand behind it, giving the log line to
search for and why the 401s and the 400 are not counted; the README's metric
table gains its row. Stamps the version in the Makefile, console package
metadata and both compose files, with no schema or configuration changes.
