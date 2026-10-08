-- Handing a space to another in-house service without a person carrying the file.
--
-- A thought starts on umm's canvas, becomes a document in muni, slides in
-- ptium, a report in weekly. The formats already fit — umm writes Markdown and
-- the others read it — but at every step somebody downloaded a file and
-- uploaded it again. What was missing was the hand.
--
-- The services hold no credentials for one another. umm issues a claim: a
-- random token bound to one document and one person's right to read it, good
-- for five minutes and for one collection. The receiving service is opened in
-- the browser with the claim, fetches the document from umm with it, and the
-- claim is gone.
--
-- The document is rendered when the claim is issued and kept with it, so the
-- receiving service gets exactly the bytes the claim announced (the size is in
-- the claim's answer) even if the space changes in the minutes between. Only a
-- digest of the token is stored: a read of this table yields nothing anybody
-- could present. A row is deleted when it is collected — that is what single
-- use means — and expired rows are swept when the next claim is issued.
CREATE TABLE IF NOT EXISTS handoff_claims (
  claim_digest bytea PRIMARY KEY,
  space_id uuid NOT NULL REFERENCES spaces(id) ON DELETE CASCADE,
  issued_by uuid NOT NULL REFERENCES users(id) ON DELETE CASCADE,
  filename text NOT NULL,
  content_type text NOT NULL,
  body bytea NOT NULL,
  expires_at timestamptz NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS handoff_claims_expires_idx ON handoff_claims(expires_at);

-- Where a space may be sent. Seeded empty: the menu that sends a space to
-- another service does not appear until an administrator names one, so a new
-- installation behaves exactly as it did before.
INSERT INTO app_settings(key, value, updated_at)
VALUES ('handoff', jsonb_build_object('targets', '[]'::jsonb), now())
ON CONFLICT (key) DO NOTHING;

INSERT INTO schema_migrations (version) VALUES ('029_handoff_claims')
  ON CONFLICT DO NOTHING;
