-- Every threshold behind related thoughts, clustering, search labelling and
-- auto-link used to be a constant in the binary. They were chosen by
-- measurement and they are good defaults, but they were measured on one dataset
-- in two languages — an operator running a narrow corpus may genuinely need a
-- different line, and the only way to find out was to rebuild umm.
--
-- The values seeded here are exactly the shipped constants, so a deployment that
-- changes nothing behaves exactly as it did before.
--
-- Bands are standard deviations above the mean of the scores being judged, not
-- raw cosine values. That is what lets one number mean the same thing whichever
-- embedding backend produced the scores.
INSERT INTO app_settings(key, value, updated_at)
VALUES ('intelligence', jsonb_build_object(
  'related_band', 0.6,
  'cluster_band', 1.1,
  'strong_band', 0.9,
  'autolink_enabled', true,
  'autolink_band', 1.1,
  'autolink_max_per_run', 12,
  'autolink_min_notes', 6,
  'semantic_accuracy_bar', 0.65,
  'semantic_purity_bar', 0.6,
  'quality_cache_minutes', 10,
  'duplicate_similarity', 0.92
), now())
ON CONFLICT (key) DO NOTHING;
