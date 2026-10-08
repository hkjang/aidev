CREATE TABLE diagnostic_logs (
    id TEXT PRIMARY KEY,
    occurred_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    level TEXT NOT NULL CHECK (level IN ('info', 'warning', 'error')),
    component TEXT NOT NULL,
    event_code TEXT NOT NULL,
    message TEXT NOT NULL,
    request_id TEXT NOT NULL DEFAULT '',
    instance_id TEXT NOT NULL,
    agent_id TEXT NOT NULL DEFAULT '',
    source_ip TEXT NOT NULL DEFAULT '',
    details_json TEXT NOT NULL DEFAULT '{}'
);

CREATE INDEX diagnostic_logs_occurred_at_idx
    ON diagnostic_logs(occurred_at DESC);
CREATE INDEX diagnostic_logs_instance_id_idx
    ON diagnostic_logs(instance_id, occurred_at DESC);
CREATE INDEX diagnostic_logs_request_id_idx
    ON diagnostic_logs(request_id);
CREATE INDEX diagnostic_logs_agent_id_idx
    ON diagnostic_logs(agent_id, occurred_at DESC);
