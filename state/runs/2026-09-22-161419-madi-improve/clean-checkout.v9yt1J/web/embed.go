package web

import "embed"

// Assets contains the complete offline frontend.
//
//go:embed all:dist
var Assets embed.FS
