ALTER TABLE api_keys ADD COLUMN key_prefix TEXT NOT NULL DEFAULT '';
ALTER TABLE api_keys ADD COLUMN previous_key_hash TEXT;
ALTER TABLE api_keys ADD COLUMN previous_valid_until TEXT;
ALTER TABLE api_keys ADD COLUMN updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP;

CREATE INDEX idx_api_keys_previous_hash ON api_keys(previous_key_hash);
CREATE INDEX idx_api_keys_owner ON api_keys(user_id, revoked_at);

INSERT OR IGNORE INTO permissions(name, description) VALUES
    ('api_keys.manage', 'Create, scope, rotate, and revoke API keys'),
    ('mcp.access', 'Access the MCP Streamable HTTP endpoint');

INSERT OR IGNORE INTO role_permissions(role_id, permission_name)
VALUES
    ('00000000-0000-0000-0000-000000000001', 'api_keys.manage'),
    ('00000000-0000-0000-0000-000000000001', 'mcp.access');

UPDATE server_metadata
SET value = '3', updated_at = CURRENT_TIMESTAMP
WHERE key = 'schema_version';
