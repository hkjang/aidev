CREATE TABLE server_metadata (
    key TEXT PRIMARY KEY,
    value TEXT NOT NULL,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE users (
    id TEXT PRIMARY KEY,
    username TEXT NOT NULL,
    normalized_username TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL DEFAULT '',
    email TEXT NOT NULL DEFAULT '',
    password_hash TEXT,
    active INTEGER NOT NULL DEFAULT 1 CHECK (active IN (0, 1)),
    super_admin INTEGER NOT NULL DEFAULT 0 CHECK (super_admin IN (0, 1)),
    emergency_access INTEGER NOT NULL DEFAULT 0 CHECK (emergency_access IN (0, 1)),
    failed_login_count INTEGER NOT NULL DEFAULT 0,
    locked_until TEXT,
    password_changed_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TEXT
);

CREATE TABLE external_identities (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id),
    provider TEXT NOT NULL,
    subject TEXT NOT NULL,
    claims_json TEXT NOT NULL DEFAULT '{}',
    last_login_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (provider, subject)
);

CREATE TABLE roles (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    system_role INTEGER NOT NULL DEFAULT 0 CHECK (system_role IN (0, 1)),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE permissions (
    name TEXT PRIMARY KEY,
    description TEXT NOT NULL DEFAULT ''
);

CREATE TABLE role_permissions (
    role_id TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    permission_name TEXT NOT NULL REFERENCES permissions(name) ON DELETE CASCADE,
    PRIMARY KEY (role_id, permission_name)
);

CREATE TABLE user_roles (
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id TEXT NOT NULL REFERENCES roles(id) ON DELETE CASCADE,
    source TEXT NOT NULL DEFAULT 'local',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (user_id, role_id, source)
);

CREATE TABLE sessions (
    id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash TEXT NOT NULL UNIQUE,
    csrf_hash TEXT NOT NULL,
    source_ip TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    idle_expires_at TEXT NOT NULL,
    absolute_expires_at TEXT NOT NULL,
    revoked_at TEXT
);

CREATE TABLE api_keys (
    id TEXT PRIMARY KEY,
    user_id TEXT REFERENCES users(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    key_hash TEXT NOT NULL UNIQUE,
    scopes_json TEXT NOT NULL DEFAULT '[]',
    expires_at TEXT,
    last_used_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at TEXT
);

CREATE TABLE agents (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL UNIQUE,
    hostname TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT 'discovered',
    version TEXT NOT NULL DEFAULT '',
    os_name TEXT NOT NULL DEFAULT '',
    architecture TEXT NOT NULL DEFAULT '',
    auth_method TEXT NOT NULL DEFAULT '',
    last_seen_at TEXT,
    last_inventory_at TEXT,
    policy_version TEXT NOT NULL DEFAULT '',
    blocked_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE agent_credentials (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    credential_type TEXT NOT NULL,
    secret_hash TEXT,
    certificate_fingerprint TEXT,
    certificate_expires_at TEXT,
    not_before TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    expires_at TEXT,
    grace_until TEXT,
    revoked_at TEXT,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE agent_groups (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE agent_group_members (
    group_id TEXT NOT NULL REFERENCES agent_groups(id) ON DELETE CASCADE,
    agent_id TEXT NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (group_id, agent_id)
);

CREATE TABLE agent_events (
    id TEXT PRIMARY KEY,
    agent_id TEXT NOT NULL REFERENCES agents(id),
    event_id TEXT NOT NULL,
    schema_version INTEGER NOT NULL,
    kind TEXT NOT NULL,
    snapshot_hash TEXT NOT NULL DEFAULT '',
    raw_event TEXT NOT NULL,
    received_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    processed_at TEXT,
    processing_status TEXT NOT NULL DEFAULT 'pending',
    processing_error TEXT,
    UNIQUE (agent_id, event_id)
);

CREATE TABLE agent_event_errors (
    id TEXT PRIMARY KEY,
    agent_event_id TEXT NOT NULL REFERENCES agent_events(id) ON DELETE CASCADE,
    error_code TEXT NOT NULL,
    message TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE collector_errors (
    id TEXT PRIMARY KEY,
    agent_event_id TEXT NOT NULL REFERENCES agent_events(id) ON DELETE CASCADE,
    collector TEXT NOT NULL,
    message TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE agent_policies (
    id TEXT PRIMARY KEY,
    version TEXT NOT NULL UNIQUE,
    name TEXT NOT NULL,
    policy_json TEXT NOT NULL DEFAULT '{}',
    active INTEGER NOT NULL DEFAULT 0 CHECK (active IN (0, 1)),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE agent_policy_assignments (
    policy_id TEXT NOT NULL REFERENCES agent_policies(id) ON DELETE CASCADE,
    group_id TEXT NOT NULL REFERENCES agent_groups(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (policy_id, group_id)
);

CREATE TABLE asset_types (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    display_name TEXT NOT NULL,
    schema_json TEXT NOT NULL DEFAULT '{}',
    system_type INTEGER NOT NULL DEFAULT 0 CHECK (system_type IN (0, 1)),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE assets (
    id TEXT PRIMARY KEY,
    asset_key TEXT NOT NULL,
    name TEXT NOT NULL,
    type TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'discovered',
    criticality TEXT NOT NULL DEFAULT 'normal',
    environment TEXT NOT NULL DEFAULT 'other',
    owner_department TEXT NOT NULL DEFAULT '',
    owner_user_id TEXT REFERENCES users(id),
    location TEXT NOT NULL DEFAULT '',
    business_service_id TEXT REFERENCES assets(id),
    confidence REAL NOT NULL DEFAULT 1.0,
    attributes_json TEXT NOT NULL DEFAULT '{}',
    custom_fields_json TEXT NOT NULL DEFAULT '{}',
    source TEXT NOT NULL DEFAULT 'agent',
    first_seen_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TEXT,
    UNIQUE (type, asset_key)
);

CREATE TABLE asset_sources (
    id TEXT PRIMARY KEY,
    asset_id TEXT NOT NULL REFERENCES assets(id),
    agent_id TEXT REFERENCES agents(id),
    category TEXT NOT NULL,
    source_asset_id TEXT NOT NULL,
    source_name TEXT NOT NULL,
    payload_json TEXT NOT NULL DEFAULT '{}',
    collected_at TEXT NOT NULL,
    first_seen_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    last_seen_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    deleted_at TEXT,
    UNIQUE (agent_id, category, source_asset_id)
);

CREATE TABLE asset_identifiers (
    id TEXT PRIMARY KEY,
    asset_id TEXT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    identifier_type TEXT NOT NULL,
    identifier_value TEXT NOT NULL,
    source TEXT NOT NULL,
    confidence REAL NOT NULL DEFAULT 1.0,
    trusted INTEGER NOT NULL DEFAULT 0 CHECK (trusted IN (0, 1)),
    ignored INTEGER NOT NULL DEFAULT 0 CHECK (ignored IN (0, 1)),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (asset_id, identifier_type, identifier_value)
);

CREATE TABLE asset_relations (
    id TEXT PRIMARY KEY,
    source_asset_id TEXT NOT NULL REFERENCES assets(id),
    relation_type TEXT NOT NULL,
    target_asset_id TEXT NOT NULL REFERENCES assets(id),
    valid_from TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    valid_to TEXT,
    source TEXT NOT NULL DEFAULT 'manual',
    confidence REAL NOT NULL DEFAULT 1.0,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (source_asset_id, relation_type, target_asset_id)
);

CREATE TABLE asset_snapshots (
    id TEXT PRIMARY KEY,
    asset_id TEXT NOT NULL REFERENCES assets(id),
    snapshot_json TEXT NOT NULL,
    snapshot_hash TEXT NOT NULL,
    captured_at TEXT NOT NULL,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE asset_changes (
    id TEXT PRIMARY KEY,
    asset_id TEXT NOT NULL REFERENCES assets(id),
    source_event_id TEXT REFERENCES agent_events(id),
    change_type TEXT NOT NULL,
    before_json TEXT,
    after_json TEXT,
    actor_type TEXT NOT NULL DEFAULT 'agent',
    actor_id TEXT,
    reason TEXT NOT NULL DEFAULT '',
    occurred_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    reverted_by TEXT REFERENCES asset_changes(id)
);

CREATE TABLE custom_field_definitions (
    id TEXT PRIMARY KEY,
    asset_type TEXT NOT NULL,
    field_key TEXT NOT NULL,
    display_name TEXT NOT NULL,
    value_type TEXT NOT NULL,
    required INTEGER NOT NULL DEFAULT 0 CHECK (required IN (0, 1)),
    validation_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (asset_type, field_key)
);

CREATE TABLE tags (
    id TEXT PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    color TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE asset_tags (
    asset_id TEXT NOT NULL REFERENCES assets(id) ON DELETE CASCADE,
    tag_id TEXT NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (asset_id, tag_id)
);

CREATE TABLE saved_queries (
    id TEXT PRIMARY KEY,
    owner_user_id TEXT NOT NULL REFERENCES users(id),
    name TEXT NOT NULL,
    query_dsl TEXT NOT NULL,
    shared_roles_json TEXT NOT NULL DEFAULT '[]',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE settings (
    key TEXT PRIMARY KEY,
    value_json TEXT NOT NULL,
    secret INTEGER NOT NULL DEFAULT 0 CHECK (secret IN (0, 1)),
    apply_mode TEXT NOT NULL DEFAULT 'immediate',
    pending_value_json TEXT,
    version INTEGER NOT NULL DEFAULT 1,
    updated_by TEXT REFERENCES users(id),
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE setting_versions (
    id TEXT PRIMARY KEY,
    setting_key TEXT NOT NULL,
    version INTEGER NOT NULL,
    before_json TEXT,
    after_json TEXT,
    changed_by TEXT REFERENCES users(id),
    reason TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (setting_key, version)
);

CREATE TABLE audit_logs (
    id TEXT PRIMARY KEY,
    occurred_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    actor_type TEXT NOT NULL,
    actor_id TEXT,
    actor_name TEXT NOT NULL DEFAULT '',
    action TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id TEXT,
    request_id TEXT NOT NULL DEFAULT '',
    source_ip TEXT NOT NULL DEFAULT '',
    user_agent TEXT NOT NULL DEFAULT '',
    result TEXT NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    before_json TEXT,
    after_json TEXT,
    metadata_json TEXT NOT NULL DEFAULT '{}'
);

CREATE TABLE db_migration_jobs (
    id TEXT PRIMARY KEY,
    status TEXT NOT NULL,
    target_fingerprint TEXT NOT NULL,
    mode TEXT NOT NULL DEFAULT 'copy',
    progress REAL NOT NULL DEFAULT 0,
    error_code TEXT,
    error_message TEXT,
    created_by TEXT REFERENCES users(id),
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TEXT
);

CREATE TABLE db_migration_checkpoints (
    job_id TEXT NOT NULL REFERENCES db_migration_jobs(id) ON DELETE CASCADE,
    table_name TEXT NOT NULL,
    last_key TEXT,
    copied_rows INTEGER NOT NULL DEFAULT 0,
    expected_rows INTEGER NOT NULL DEFAULT 0,
    checksum TEXT,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
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
    ('00000000-0000-0000-0000-000000000001', 'super_admin', 'All permissions', 1),
    ('00000000-0000-0000-0000-000000000002', 'asset_manager', 'Asset lifecycle management', 1),
    ('00000000-0000-0000-0000-000000000003', 'operator', 'Agent and collection operations', 1),
    ('00000000-0000-0000-0000-000000000004', 'security_admin', 'Security asset review', 1),
    ('00000000-0000-0000-0000-000000000005', 'auditor', 'Read-only audit and export', 1),
    ('00000000-0000-0000-0000-000000000006', 'viewer', 'Read-only asset access', 1),
    ('00000000-0000-0000-0000-000000000007', 'api_user', 'Scoped API access', 1);

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
