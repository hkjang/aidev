-- A login started with prompt=none is answered by the provider without any
-- screen. The callback has to know the flow was silent so that a refusal
-- (login_required) lands on the login page with the do-not-retry marker
-- instead of being reported as a provider failure.
ALTER TABLE oidc_flows
    ADD COLUMN silent BOOLEAN NOT NULL DEFAULT FALSE;
