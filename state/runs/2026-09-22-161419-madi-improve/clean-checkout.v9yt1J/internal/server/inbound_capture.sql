CREATE TABLE IF NOT EXISTS inbound_capture_settings(id integer PRIMARY KEY CHECK(id=1),hooks_enabled boolean NOT NULL DEFAULT false,imap_enabled boolean NOT NULL DEFAULT false,allowed_hosts jsonb NOT NULL DEFAULT '[]');
INSERT INTO inbound_capture_settings(id) VALUES(1) ON CONFLICT DO NOTHING;
CREATE TABLE IF NOT EXISTS inbound_capture_channels(
 id uuid PRIMARY KEY,user_id uuid NOT NULL REFERENCES users(id),workspace_id uuid NOT NULL REFERENCES workspaces(id),
 name text NOT NULL,kind text NOT NULL CHECK(kind IN ('hmac','imap')),config jsonb NOT NULL DEFAULT '{}',secret_ciphertext text NOT NULL,
 enabled boolean NOT NULL DEFAULT false,revision integer NOT NULL DEFAULT 1,checkpoint jsonb NOT NULL DEFAULT '{}',next_run timestamptz,
 created_at timestamptz NOT NULL DEFAULT now(),updated_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS inbound_capture_messages(
 id uuid PRIMARY KEY,channel_id uuid NOT NULL REFERENCES inbound_capture_channels(id),message_key text NOT NULL,message_id text NOT NULL DEFAULT '',
 payload_hash text NOT NULL,payload_ciphertext text NOT NULL DEFAULT '',payload_size integer NOT NULL DEFAULT 0,
 document_id uuid REFERENCES documents(id) ON DELETE SET NULL,job_id uuid REFERENCES automation_jobs(id),channel_revision integer NOT NULL,
 status text NOT NULL DEFAULT 'pending' CHECK(status IN ('pending','completed','rejected','cancelled')),title text NOT NULL DEFAULT '',message text NOT NULL DEFAULT '',
 created_at timestamptz NOT NULL DEFAULT now(),completed_at timestamptz,UNIQUE(channel_id,message_key)
);
CREATE UNIQUE INDEX IF NOT EXISTS inbound_capture_message_id_idx ON inbound_capture_messages(channel_id,message_id) WHERE message_id<>'';
CREATE INDEX IF NOT EXISTS inbound_capture_messages_channel_idx ON inbound_capture_messages(channel_id,created_at DESC);
