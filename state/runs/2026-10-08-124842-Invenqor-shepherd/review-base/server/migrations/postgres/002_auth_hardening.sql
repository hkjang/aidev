CREATE TABLE login_attempts (
    id UUID PRIMARY KEY,
    normalized_username TEXT NOT NULL,
    source_ip INET NOT NULL,
    succeeded BOOLEAN NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL
);

CREATE TABLE oidc_flows (
    id UUID PRIMARY KEY,
    state_hash TEXT NOT NULL UNIQUE,
    nonce_hash TEXT NOT NULL,
    pkce_verifier TEXT NOT NULL,
    redirect_uri TEXT NOT NULL,
    return_to TEXT NOT NULL DEFAULT '/',
    source_ip INET,
    user_agent TEXT NOT NULL DEFAULT '',
    expires_at TIMESTAMPTZ NOT NULL,
    consumed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_totp (
    user_id UUID PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    encrypted_secret TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT FALSE,
    recovery_codes_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    verified_at TIMESTAMPTZ,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_login_attempts_ip_time
    ON login_attempts(source_ip, occurred_at);
CREATE INDEX idx_login_attempts_user_time
    ON login_attempts(normalized_username, occurred_at);
CREATE INDEX idx_oidc_flows_expiry
    ON oidc_flows(expires_at, consumed_at);
