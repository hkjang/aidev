ALTER TABLE agents
    ADD COLUMN last_event_created_at TEXT;

ALTER TABLE agents
    ADD COLUMN last_event_id TEXT NOT NULL DEFAULT '';

ALTER TABLE agents
    ADD COLUMN last_event_received_at TEXT;

ALTER TABLE asset_sources
    ADD COLUMN last_event_created_at TEXT;

ALTER TABLE asset_sources
    ADD COLUMN last_event_id TEXT NOT NULL DEFAULT '';

ALTER TABLE asset_sources
    ADD COLUMN last_event_received_at TEXT;

UPDATE asset_sources
SET last_event_created_at = collected_at
WHERE last_event_created_at IS NULL;

UPDATE asset_sources
SET last_event_received_at = last_event_created_at
WHERE last_event_received_at IS NULL;

CREATE TABLE external_identities_v7 (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    provider TEXT NOT NULL,
    issuer TEXT NOT NULL DEFAULT '',
    subject TEXT NOT NULL,
    claims_json TEXT NOT NULL DEFAULT '{}',
    last_login_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (provider, issuer, subject)
);

INSERT INTO external_identities_v7(
    id, user_id, provider, issuer, subject, claims_json,
    last_login_at, created_at
)
SELECT
    id, user_id, provider, '', subject, claims_json,
    last_login_at, created_at
FROM external_identities;

DROP TABLE external_identities;

ALTER TABLE external_identities_v7 RENAME TO external_identities;

CREATE INDEX idx_external_identities_user
    ON external_identities(user_id);
