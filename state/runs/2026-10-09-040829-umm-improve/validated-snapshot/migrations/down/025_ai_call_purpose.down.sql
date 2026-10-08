DROP INDEX IF EXISTS ai_calls_user_recent_idx;
ALTER TABLE ai_calls DROP CONSTRAINT IF EXISTS ai_calls_purpose_check;
ALTER TABLE ai_calls DROP COLUMN IF EXISTS purpose;
