-- Every notification mail attempt, so an administrator can answer "did it go
-- out" for successes as well as failures. The body is never stored.
CREATE TABLE mail_deliveries (
    id TEXT PRIMARY KEY,
    event TEXT NOT NULL,
    recipient TEXT NOT NULL,
    subject TEXT NOT NULL,
    actor_id TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL CHECK (status IN ('queued', 'sent', 'failed')),
    attempts INTEGER NOT NULL DEFAULT 0,
    error_message TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TEXT NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX mail_deliveries_created_at_idx
    ON mail_deliveries(created_at DESC);
