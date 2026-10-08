package migrations

import "embed"

// Files contains independent PostgreSQL and SQLite migration streams.
//
//go:embed postgres/*.sql sqlite/*.sql
var Files embed.FS
