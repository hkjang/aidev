package migrations

import "embed"

// FS contains all database migrations required to start the standalone image.
//
//go:embed *.sql
var FS embed.FS
