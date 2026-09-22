package server

import "time"

const searchObservationKey contextKey = "madi-search-observation"

// Explicit diagnostic/evaluation requests inspect only their own parameterized
// read query. This collector is private context, never client-supplied state.
type searchObservation struct {
	SQL       string
	Arguments []any
	Elapsed   time.Duration
}
