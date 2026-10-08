-- Where umm sends a talk it has compiled.
--
-- Its own section rather than a field on ai_gateway, because it is not one.
-- ai_gateway is a model endpoint that writes text; Ptium lays out and renders
-- text umm already has, and umm never sends it a prompt. Sharing a credential
-- between the two would also mean an operator who rotates one silently changes
-- the other.
--
-- Seeded empty and therefore off: connecting another service to someone's
-- thoughts is a decision an administrator makes deliberately, not a default
-- that arrives with an upgrade.
INSERT INTO app_settings(key, value, updated_at)
VALUES ('ptium', jsonb_build_object(
  'base_url', '',
  'api_key', '',
  -- Ptium's own default template is used when this is empty, so an operator who
  -- has not chosen one still gets a deck rather than an error.
  'template_id', '',
  'language', 'ko',
  -- Compiling a deck is slower than an ordinary request and much faster than
  -- generation, which umm never waits on.
  'timeout_seconds', 30
), now())
ON CONFLICT (key) DO NOTHING;

INSERT INTO schema_migrations (version) VALUES ('019_ptium_settings')
  ON CONFLICT DO NOTHING;
