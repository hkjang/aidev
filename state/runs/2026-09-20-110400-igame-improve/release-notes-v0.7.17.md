Offline image: `igame:v0.7.17`

## What's Changed
* auto-improve: fix: let a portal-wide achievement be unlocked from any game session by @hkjang in https://github.com/hkjang/igame/pull/22

`unlockAchievement` joined achievements to game sessions on `game_id` alone, which never matches an achievement whose `game_id` is NULL, so a portal-wide `client_unlockable` achievement created from the admin screen always answered `403 achievement_not_unlockable`. A game-bound achievement is still unlocked only from a session of its own game; a portal-wide one now unlocks from a session of any game. The `client_unlockable`, own-session, session-token-hash and session-status conditions are unchanged. Three `IGAME_TEST_DSN` tests drive the real router for both cases; `docs/api.md` states the unlock rule.

No migration, no content change: RealmGuard content 0.3.1 and Defense Series content 0.4.0 ride along untouched.

**Full Changelog**: https://github.com/hkjang/igame/compare/v0.7.16...v0.7.17
