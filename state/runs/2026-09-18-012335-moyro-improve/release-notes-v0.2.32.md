MCP clients can now open `/mcp` with a Keycloak access token next to the
personal keys. A new `oauth` object on the MCP settings, off by default,
publishes RFC 9728 protected-resource metadata, attaches the
`resource_metadata` challenge to `/mcp` 401s only, verifies the bearer
against the live Keycloak JWKS (asymmetric signature, iss, exp, nbf, no ID
token, no `cnf`, non-empty `sub`), requires the resource identifier in `aud`
or an administrator-listed `aud`/`azp` with a refusal that names what was
seen and what to add, and admits only accounts the web sign-in already
linked; scopes come from the administrator list intersected with the token
and the existing MCP gate and RBAC checks apply unchanged. The admin MCP
card and the personal-key page gain the switches and copyable addresses,
and the guides and OpenAPI contract follow.

