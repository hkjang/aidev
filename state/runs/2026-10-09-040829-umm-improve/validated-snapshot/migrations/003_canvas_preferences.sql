ALTER TABLE user_preferences
ADD COLUMN IF NOT EXISTS edge_style text NOT NULL DEFAULT 'bezier';
