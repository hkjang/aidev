CREATE TABLE server_metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE users (
    id UUID PRIMARY KEY,
    username TEXT NOT NULL,
    normalized_username TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL DEFAULT '',
    password_hash TEXT,
    active BOOLEAN NOT NULL DEFAULT TRUE,
    super_admin BOOLEAN NOT NULL DEFAULT FALSE,
    emergency_access BOOLEAN NOT NULL DEFAULT FALSE,
    failed_login_count INTEGER NOT NULL DEFAULT 0,
    locked_until TIMESTAMPTZ,
    password_changed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ
);

CREATE TABLE external_identities (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id),
    provider TEXT NOT NULL,
    subject TEXT NOT NULL,
    claims_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    last_login_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (provider, subject)
);

CREATE TABLE roles (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    system_role BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE permissions (
    name TEXT PRIMARY KEY,
    description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE role_permissions (
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_name TEXT NOT NULL REFERENCES permissions(name) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_name)
);

CREATE TABLE user_roles (
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    source TEXT NOT NULL DEFAULT 'local',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role_id, source)
);

CREATE TABLE sessions (
    id UUID PRIMARY KEY,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    csrf_hash TEXT NOT NULL,
    source_ip TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    idle_expires_at TIMESTAMPTZ NOT NULL,
    absolute_expires_at TIMESTAMPTZ NOT NULL,
    revoked_at TIMESTAMPTZ
);

CREATE TABLE api_keys (
    id UUID PRIMARY KEY,
    user_id UUID REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    key_hash TEXT NOT NULL UNIQUE,
    scopes_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMPTZ
);

CREATE TABLE agents (
    id UUID PRIMARY KEY,
    agent_id UUID NOT NULL UNIQUE,
    hostname TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'discovered',
    version TEXT NOT NULL DEFAULT '',
    os_name TEXT NOT NULL DEFAULT '',
    architecture TEXT NOT NULL DEFAULT '',
    auth_method TEXT NOT NULL DEFAULT '',
    last_seen_at TIMESTAMPTZ,
    last_inventory_at TIMESTAMPTZ,
    policy_version TEXT NOT NULL DEFAULT '',
    blocked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE agent_credentials (
    id UUID PRIMARY KEY,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    credential_type TEXT NOT NULL,
    secret_hash TEXT,
    certificate_fingerprint TEXT,
    certificate_expires_at TIMESTAMPTZ,
    not_before TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMPTZ,
    grace_until TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE agent_groups (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE agent_group_members (
    group_id UUID NOT NULL REFERENCES agent_groups(id) ON DELETE CASCADE,
    agent_id UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (group_id, agent_id)
);

CREATE TABLE agent_events (
    id UUID PRIMARY KEY,
    agent_id UUID NOT NULL REFERENCES agents(id),
    event_id UUID NOT NULL,
    schema_version INTEGER NOT NULL,
    kind TEXT NOT NULL,
    snapshot_hash TEXT NOT NULL DEFAULT '',
    raw_event JSONB NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at TIMESTAMPTZ,
    processing_status TEXT NOT NULL DEFAULT 'pending',
    processing_error TEXT,
    UNIQUE (agent_id, event_id)
);

CREATE TABLE agent_event_errors (
    id UUID PRIMARY KEY,
    agent_event_id UUID NOT NULL REFERENCES agent_events(id) ON DELETE CASCADE,
    error_code TEXT NOT NULL,
    message TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE collector_errors (
    id UUID PRIMARY KEY,
    agent_event_id UUID NOT NULL REFERENCES agent_events(id) ON DELETE CASCADE,
    collector TEXT NOT NULL,
    message TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE agent_policies (
    id UUID PRIMARY KEY,
    version TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    policy_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    active BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE agent_policy_assignments (
    policy_id UUID NOT NULL REFERENCES agent_policies(id) ON DELETE CASCADE,
    group_id UUID NOT NULL REFERENCES agent_groups(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (policy_id, group_id)
);

CREATE TABLE asset_types (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    schema_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    system_type BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE assets (
    id UUID PRIMARY KEY,
    asset_key TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'discovered',
    criticality TEXT NOT NULL DEFAULT 'normal',
    environment TEXT NOT NULL DEFAULT 'other',
    owner_department TEXT NOT NULL DEFAULT '',
    owner_user_id UUID REFERENCES users(id),
    location TEXT NOT NULL DEFAULT '',
    business_service_id UUID REFERENCES assets(id),
    confidence DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    attributes_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    custom_fields_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    source TEXT NOT NULL DEFAULT 'agent',
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    UNIQUE (type, asset_key)
);

CREATE TABLE asset_sources (
    id UUID PRIMARY KEY,
    asset_id UUID NOT NULL REFERENCES assets(id),
    agent_id UUID REFERENCES agents(id),
    category TEXT NOT NULL,
    source_asset_id TEXT NOT NULL,
    source_name TEXT NOT NULL,
    payload_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    collected_at TIMESTAMPTZ NOT NULL,
    first_seen_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMPTZ,
    UNIQUE (agent_id, category, source_asset_id)
);

CREATE TABLE asset_identifiers (
    id UUID PRIMARY KEY,
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    identifier_type TEXT NOT NULL,
    identifier_value TEXT NOT NULL,
    source TEXT NOT NULL,
    confidence DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    trusted BOOLEAN NOT NULL DEFAULT FALSE,
    ignored BOOLEAN NOT NULL DEFAULT FALSE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (asset_id, identifier_type, identifier_value)
);

CREATE TABLE asset_relations (
    id UUID PRIMARY KEY,
    source_asset_id UUID NOT NULL REFERENCES assets(id),
    relation_type TEXT NOT NULL,
    target_asset_id UUID NOT NULL REFERENCES assets(id),
    valid_from TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    valid_to TIMESTAMPTZ,
    source TEXT NOT NULL DEFAULT 'manual',
    confidence DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (source_asset_id, relation_type, target_asset_id)
);

CREATE TABLE asset_snapshots (
    id UUID PRIMARY KEY,
    asset_id UUID NOT NULL REFERENCES assets(id),
    snapshot_json JSONB NOT NULL,
    snapshot_hash TEXT NOT NULL,
    captured_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE asset_changes (
    id UUID PRIMARY KEY,
    asset_id UUID NOT NULL REFERENCES assets(id),
    source_event_id UUID REFERENCES agent_events(id),
    change_type TEXT NOT NULL,
    before_json JSONB,
    after_json JSONB,
    actor_type TEXT NOT NULL DEFAULT 'agent',
    actor_id UUID,
    reason TEXT NOT NULL DEFAULT '',
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reverted_by UUID REFERENCES asset_changes(id)
);

CREATE TABLE custom_field_definitions (
    id UUID PRIMARY KEY,
    asset_type TEXT NOT NULL,
    field_key TEXT NOT NULL,
    display_name TEXT NOT NULL,
    value_type TEXT NOT NULL,
    required BOOLEAN NOT NULL DEFAULT FALSE,
    validation_json JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (asset_type, field_key)
);

CREATE TABLE tags (
    id UUID PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    color TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE asset_tags (
    asset_id UUID NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    tag_id UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (asset_id, tag_id)
);

CREATE TABLE saved_queries (
    id UUID PRIMARY KEY,
    owner_user_id UUID NOT NULL REFERENCES users(id),
    name TEXT NOT NULL,
    query_dsl TEXT NOT NULL,
    shared_roles_json JSONB NOT NULL DEFAULT '[]'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value_json JSONB NOT NULL,
    secret BOOLEAN NOT NULL DEFAULT FALSE,
    apply_mode TEXT NOT NULL DEFAULT 'immediate',
    pending_value_json JSONB,
    version INTEGER NOT NULL DEFAULT 1,
    updated_by UUID REFERENCES users(id),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE setting_versions (
    id UUID PRIMARY KEY,
    setting_key TEXT NOT NULL,
    version INTEGER NOT NULL,
    before_json JSONB,
    after_json JSONB,
    changed_by UUID REFERENCES users(id),
    reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (setting_key, version)
);

CREATE TABLE audit_logs (
    id UUID PRIMARY KEY,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actor_type TEXT NOT NULL,
    actor_id UUID,
    actor_name TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    request_id TEXT NOT NULL DEFAULT '',
    source_ip TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    result TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    before_json JSONB,
    after_json JSONB,
    metadata_json JSONB NOT NULL DEFAULT '{}'::jsonb
);

CREATE TABLE db_migration_jobs (
    id UUID PRIMARY KEY,
    status TEXT NOT NULL,
    target_fingerprint TEXT NOT NULL,
    mode TEXT NOT NULL DEFAULT 'copy',
    progress DOUBLE PRECISION NOT NULL DEFAULT 0,
    error_code TEXT,
    error_message TEXT,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMPTZ
);

CREATE TABLE db_migration_checkpoints (
    job_id UUID NOT NULL REFERENCES db_migration_jobs(id) ON DELETE CASCADE,
    table_name TEXT NOT NULL,
    last_key TEXT,
    copied_rows BIGINT NOT NULL DEFAULT 0,
    expected_rows BIGINT NOT NULL DEFAULT 0,
    checksum TEXT,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (job_id, table_name)
);

CREATE INDEX idx_agents_last_seen ON agents(last_seen_at);
CREATE INDEX idx_agent_events_status ON agent_events(processing_status, received_at);
CREATE INDEX idx_assets_type_status ON assets(type, status, deleted_at);
CREATE INDEX idx_assets_last_seen ON assets(last_seen_at);
CREATE INDEX idx_asset_sources_asset ON asset_sources(asset_id, deleted_at);
CREATE INDEX idx_asset_sources_agent ON asset_sources(agent_id, category, deleted_at);
CREATE INDEX idx_asset_changes_asset_time ON asset_changes(asset_id, occurred_at);
CREATE INDEX idx_asset_relations_source ON asset_relations(source_asset_id, valid_to);
CREATE INDEX idx_asset_relations_target ON asset_relations(target_asset_id, valid_to);
CREATE INDEX idx_audit_logs_time ON audit_logs(occurred_at);
CREATE INDEX idx_audit_logs_resource ON audit_logs(resource_type, resource_id, occurred_at);

INSERT INTO permissions(name, description) VALUES
    ('assets.read', 'Read assets'),
    ('assets.write', 'Create and update assets'),
    ('assets.delete', 'Logically delete assets'),
    ('assets.merge', 'Merge and split representative assets'),
    ('assets.export', 'Export asset data'),
    ('relations.read', 'Read asset relations'),
    ('relations.write', 'Create and remove asset relations'),
    ('agents.read', 'Read agent status and events'),
    ('agents.manage', 'Manage agent credentials and state'),
    ('users.read', 'Read users and roles'),
    ('users.manage', 'Manage users and roles'),
    ('settings.read', 'Read runtime settings'),
    ('settings.write', 'Change runtime settings'),
    ('database.migrate', 'Run database migration and cutover'),
    ('audit.read', 'Read audit logs'),
    ('queries.execute', 'Execute safe Query DSL'),
    ('queries.manage', 'Manage saved queries');

INSERT INTO roles(id, name, description, system_role) VALUES
    ('00000000-0000-0000-0000-000000000001', 'super_admin', 'All permissions', TRUE),
    ('00000000-0000-0000-0000-000000000002', 'asset_manager', 'Asset lifecycle management', TRUE),
    ('00000000-0000-0000-0000-000000000003', 'operator', 'Agent and collection operations', TRUE),
    ('00000000-0000-0000-0000-000000000004', 'security_admin', 'Security asset review', TRUE),
    ('00000000-0000-0000-0000-000000000005', 'auditor', 'Read-only audit and export', TRUE),
    ('00000000-0000-0000-0000-000000000006', 'viewer', 'Read-only asset access', TRUE),
    ('00000000-0000-0000-0000-000000000007', 'api_user', 'Scoped API access', TRUE);

INSERT INTO role_permissions(role_id, permission_name)
SELECT '00000000-0000-0000-0000-000000000001', name FROM permissions;

INSERT INTO role_permissions(role_id, permission_name) VALUES
    ('00000000-0000-0000-0000-000000000002', 'assets.read'),
    ('00000000-0000-0000-0000-000000000002', 'assets.write'),
    ('00000000-0000-0000-0000-000000000002', 'assets.delete'),
    ('00000000-0000-0000-0000-000000000002', 'assets.merge'),
    ('00000000-0000-0000-0000-000000000002', 'assets.export'),
    ('00000000-0000-0000-0000-000000000002', 'relations.read'),
    ('00000000-0000-0000-0000-000000000002', 'relations.write'),
    ('00000000-0000-0000-0000-000000000003', 'agents.read'),
    ('00000000-0000-0000-0000-000000000003', 'agents.manage'),
    ('00000000-0000-0000-0000-000000000003', 'assets.read'),
    ('00000000-0000-0000-0000-000000000004', 'assets.read'),
    ('00000000-0000-0000-0000-000000000004', 'agents.read'),
    ('00000000-0000-0000-0000-000000000005', 'assets.read'),
    ('00000000-0000-0000-0000-000000000005', 'assets.export'),
    ('00000000-0000-0000-0000-000000000005', 'audit.read'),
    ('00000000-0000-0000-0000-000000000006', 'assets.read');

INSERT INTO server_metadata(key, value) VALUES
    ('schema_version', '1'),
    ('policy_version', '2026-07-28.1');
