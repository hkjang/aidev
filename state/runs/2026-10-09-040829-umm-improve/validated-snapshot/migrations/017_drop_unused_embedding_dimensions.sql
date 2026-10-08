-- A setting nothing has ever read.
--
-- 007 seeded ai_gateway.embedding_dimensions and no code has looked at it since.
-- Verified against a running instance: setting it to 256 while the model returns
-- 1024-dimension vectors changes nothing at all — the vectors stay 1024 and every
-- measurement is unaffected.
--
-- A knob that appears in the settings API and does nothing is worse than a
-- missing one. Someone tuning a deployment reads it as a lever, sets it, and
-- concludes the thing they were trying to change does not matter.
--
-- Implementing it instead was the other option and it is worse: the OpenAI API
-- takes a dimensions parameter and ollama ignores it, so the setting would work
-- on some backends and silently not on others, which is the same problem wearing
-- a different hat.
UPDATE app_settings
SET value = value - 'embedding_dimensions'
WHERE key = 'ai_gateway' AND value ? 'embedding_dimensions';
