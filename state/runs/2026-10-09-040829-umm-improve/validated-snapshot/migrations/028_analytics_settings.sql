-- Visitor tracking, as an administrator attaches it from the settings screen.
--
-- A setting rather than an environment variable, because in a closed network
-- the collector's address differs per installation and changes while umm is
-- running; a setting that needs a redeploy to change ends up left off.
--
-- Seeded off. A new installation serves exactly the pages it served before
-- this row existed, and nothing about a visit is sent anywhere until an
-- administrator turns the switch — that is a decision, not a default that
-- arrives with an upgrade.
--
-- Momento is the first choice and the proxy is on: it is the in-house
-- collector, the one provider for which nothing leaves the network, and
-- through umm's own /momento path no outside origin ever enters the
-- content security policy.
INSERT INTO app_settings(key, value, updated_at)
VALUES ('analytics', jsonb_build_object(
  'enabled', false,
  'provider', 'momento',
  'momento_url', '',
  'momento_site_id', '',
  'momento_proxy', true,
  'measurement_id', '',
  'matomo_url', '',
  'matomo_site_id', '',
  'custom_snippet', '',
  'allowed_hosts', '',
  'include_admin', false,
  'placement', 'head'
), now())
ON CONFLICT (key) DO NOTHING;

INSERT INTO schema_migrations (version) VALUES ('028_analytics_settings')
  ON CONFLICT DO NOTHING;
