UPDATE app_settings
SET value = value || '{"log_retention_days":90}'::jsonb
WHERE key = 'ai_gateway' AND NOT (value ? 'log_retention_days');
