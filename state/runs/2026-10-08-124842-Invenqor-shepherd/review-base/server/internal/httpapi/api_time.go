package httpapi

import "github.com/hkjang/invenqor/server/internal/apitime"

// apiTime normalises a timestamp that a handler scanned into an `any` before it
// goes into a response map. Response *structs* declare apitime.Time instead, so
// they are normalised by construction; this is for the handlers that assemble a
// map directly. See the apitime package for why this conversion exists.
func apiTime(value any) any { return apitime.Normalise(value) }
