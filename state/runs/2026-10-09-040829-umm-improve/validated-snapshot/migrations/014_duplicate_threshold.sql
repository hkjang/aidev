-- Duplicate detection arrived after the intelligence settings row was seeded, so
-- deployments created before it have no value for the threshold. The code falls
-- back to the same default, but an absent key means the administrator screen
-- shows an empty field where a number belongs.
--
-- 0.92 is measured rather than chosen. It is also the one absolute cosine
-- threshold umm keeps: every other bar is relative because backends disagree
-- about what "close" means, but near-identical text lands at the top of any sane
-- embedding space and the two models measured agree on where — bge-m3 puts
-- near-duplicates at 0.943 and above, paraphrase-multilingual at 0.954 and
-- above, while the next class down tops out at 0.681 and 0.581.
UPDATE app_settings
SET value = jsonb_set(value, '{duplicate_similarity}', '0.92'::jsonb)
WHERE key = 'intelligence' AND NOT (value ? 'duplicate_similarity');
