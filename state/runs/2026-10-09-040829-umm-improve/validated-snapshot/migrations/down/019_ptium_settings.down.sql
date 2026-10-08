-- Reverses migrations/019_ptium_settings.sql.
--
-- The row goes, credential and all. Decks already made stay in Ptium and the
-- links to them stay in presentation_links: forgetting where Ptium is does not
-- make it untrue that a space produced a deck.
DELETE FROM app_settings WHERE key = 'ptium';

DELETE FROM schema_migrations WHERE version = '019_ptium_settings';
