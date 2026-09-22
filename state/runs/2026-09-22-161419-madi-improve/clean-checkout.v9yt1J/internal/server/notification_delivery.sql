CREATE TABLE IF NOT EXISTS notification_settings(id integer PRIMARY KEY CHECK(id=1),enabled boolean NOT NULL DEFAULT false,allowed_hosts jsonb NOT NULL DEFAULT '[]');
INSERT INTO notification_settings(id) VALUES(1) ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS notification_channels(
 id uuid PRIMARY KEY,name text NOT NULL,kind text NOT NULL CHECK(kind IN ('smtp','slack','teams','mattermost','webhook')),
 workspace_id uuid REFERENCES workspaces(id) ON DELETE CASCADE,config jsonb NOT NULL DEFAULT '{}',secret_ciphertext text NOT NULL,
 enabled boolean NOT NULL DEFAULT false,revision integer NOT NULL DEFAULT 1,created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS notification_preferences(user_id uuid REFERENCES users(id) ON DELETE CASCADE,channel_id uuid REFERENCES notification_channels(id) ON DELETE CASCADE,enabled boolean NOT NULL DEFAULT false,PRIMARY KEY(user_id,channel_id));
CREATE TABLE IF NOT EXISTS notification_outbox(notification_id uuid PRIMARY KEY REFERENCES notifications(id) ON DELETE CASCADE,processed_at timestamptz,created_at timestamptz NOT NULL DEFAULT now());
CREATE INDEX IF NOT EXISTS notification_outbox_pending_idx ON notification_outbox(created_at) WHERE processed_at IS NULL;
CREATE OR REPLACE FUNCTION madi_notification_outbox() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN
 INSERT INTO notification_outbox(notification_id) VALUES(NEW.id) ON CONFLICT DO NOTHING; RETURN NEW;
END $$;
DROP TRIGGER IF EXISTS madi_notification_created ON notifications;
CREATE TRIGGER madi_notification_created AFTER INSERT ON notifications FOR EACH ROW EXECUTE FUNCTION madi_notification_outbox();
CREATE TABLE IF NOT EXISTS notification_deliveries(
 id uuid PRIMARY KEY,notification_id uuid NOT NULL REFERENCES notifications(id) ON DELETE CASCADE,channel_id uuid NOT NULL REFERENCES notification_channels(id),
 user_id uuid NOT NULL REFERENCES users(id),job_id uuid REFERENCES automation_jobs(id),channel_revision integer NOT NULL,
 status text NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','sending','sent','failed','skipped')),message text NOT NULL DEFAULT '',
 sent_at timestamptz,created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now(),UNIQUE(notification_id,channel_id)
);
CREATE INDEX IF NOT EXISTS notification_deliveries_user_idx ON notification_deliveries(user_id,created_at DESC);
