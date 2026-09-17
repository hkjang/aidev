## Offline image

- Image: `qurio:v1.4.2`
- Archive: `qurio-v1.4.2.tar.gz`
- SHA-256: written by the `Offline release` workflow when it builds and attaches the archive on tag push

Only the compressed Docker image archive is attached to this release.

## Highlights

- `/mcp` accepts Keycloak access tokens as an OAuth 2.1 resource server (RFC 9728): `mcp.oauth.enabled` / `mcp.oauth.resource` / `mcp.oauth.audience` / `mcp.oauth.scopes` settings, `/.well-known/oauth-protected-resource` metadata, `WWW-Authenticate` challenges on MCP routes only, and a ceiling-bound `oauth` principal that never gains more than the token's scopes even after re-hydration.
- Release identity migration `0036_release_1_4_2.sql` marks `1.4.2` as the current release (`build_metadata.mcpOAuth = rfc9728-resource-server-v1`).

**Full Changelog**: https://github.com/hkjang/qurio/compare/v1.4.1...v1.4.2
