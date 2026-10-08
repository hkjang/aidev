-- AI Assist sends selected note content to the configured external Gateway and
-- consumes the paid AI quota. Give operators a dedicated least-privilege scope
-- instead of implicitly granting that capability to every notes:read key.
UPDATE app_settings
SET value = jsonb_set(
  value,
  '{api_key_scopes}',
  COALESCE(value->'api_key_scopes', '[]'::jsonb) || '["ai:assist"]'::jsonb
)
WHERE key = 'security'
  AND NOT (COALESCE(value->'api_key_scopes', '[]'::jsonb) ? 'ai:assist');
