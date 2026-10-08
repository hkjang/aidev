ALTER TABLE agents
    ADD COLUMN last_event_created_at TIMESTAMPTZ;

ALTER TABLE agents
    ADD COLUMN last_event_id TEXT NOT NULL DEFAULT '';

ALTER TABLE agents
    ADD COLUMN last_event_received_at TIMESTAMPTZ;

ALTER TABLE asset_sources
    ADD COLUMN last_event_created_at TIMESTAMPTZ;

ALTER TABLE asset_sources
    ADD COLUMN last_event_id TEXT NOT NULL DEFAULT '';

ALTER TABLE asset_sources
    ADD COLUMN last_event_received_at TIMESTAMPTZ;

UPDATE asset_sources
SET last_event_created_at = collected_at
WHERE last_event_created_at IS NULL;

UPDATE asset_sources
SET last_event_received_at = last_event_created_at
WHERE last_event_received_at IS NULL;

ALTER TABLE external_identities
    ADD COLUMN issuer TEXT NOT NULL DEFAULT '';

ALTER TABLE external_identities
    DROP CONSTRAINT external_identities_provider_subject_key;

ALTER TABLE external_identities
    ADD CONSTRAINT external_identities_provider_issuer_subject_key
    UNIQUE (provider, issuer, subject);
