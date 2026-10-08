CREATE TABLE login_attempts (
    id TEXT PRIMARY KEY,
    normalized_username TEXT NOT NULL,
    source_ip TEXT NOT NULL,
    succeeded INTEGER NOT NULL CHECK (succeeded IN (0, 1)),
    occurred_at TEXT NOT NULL
);

CREATE TABLE oidc_flows (
    id TEXT PRIMARY KEY,
    state_hash TEXT NOT NULL UNIQUE,
    nonce_hash TEXT NOT NULL,
    pkce_verifier TEXT NOT NULL,
    redirect_uri TEXT NOT NULL,
    return_to TEXT NOT NULL DEFAULT '/',
    source_ip TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    expires_at TEXT NOT NULL,
    consumed_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE user_totp (
    user_id TEXT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    encrypted_secret TEXT NOT NULL,
    enabled INTEGER NOT NULL DEFAULT 0 CHECK (enabled IN (0, 1)),
    recovery_codes_json TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    verified_at TEXT,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_login_attempts_ip_time
    ON login_attempts(source_ip, occurred_at);
CREATE INDEX idx_login_attempts_user_time
    ON login_attempts(normalized_username, occurred_at);
CREATE INDEX idx_oidc_flows_expiry
    ON oidc_flows(expires_at, consumed_at);
