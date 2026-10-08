// Package apitime carries a database timestamp from a scan to a JSON response
// without letting the driver's formatting choices reach the browser.
//
// Timestamps arrive in whatever shape the active driver produced. PostgreSQL
// returns time.Time. The SQLite start-up fallback returns text, in two layouts:
// "2026-07-29 19:17:12" for a CURRENT_TIMESTAMP default, and Go's own
// time.Time.String() output for a value the server passed in.
//
// Passing those through produced a console that was quietly wrong in exactly
// the mode operators fall back to. The naive form carries no zone, so a browser
// reads stored UTC as local time and every audit entry, login time and key
// rotation shifted by the viewer's offset - nine hours in Korea. Go's String()
// form is not a layout any specification obliges a browser to parse, so it
// renders as an unparseable value in some and throws while formatting in
// others.
//
// Fields of this type are normalised by construction, so a new endpoint cannot
// reintroduce the problem by forgetting a conversion.
package apitime

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// layouts covers every shape the supported drivers produce, most specific first.
var layouts = []string{
	time.RFC3339Nano,
	"2006-01-02 15:04:05.999999999Z07:00",
	"2006-01-02 15:04:05Z07:00",
	"2006-01-02 15:04:05.999999999 -0700 MST",
	"2006-01-02 15:04:05 -0700 MST",
	"2006-01-02T15:04:05.999999999",
	"2006-01-02 15:04:05.999999999",
	"2006-01-02 15:04:05",
	"2006-01-02",
}

// Time is a nullable timestamp that scans from any supported driver and always
// marshals as RFC 3339 in UTC.
type Time struct {
	Time  time.Time
	Valid bool
	// raw keeps text that matched no known layout. Showing an operator an
	// unexpected value beats hiding that the column had content at all.
	raw string
}

// New wraps a Go time, treating the zero value as NULL.
func New(value time.Time) Time {
	if value.IsZero() {
		return Time{}
	}
	return Time{Time: value.UTC(), Valid: true}
}

func (value *Time) Scan(source any) error {
	*value = Time{}
	switch typed := source.(type) {
	case nil:
		return nil
	case time.Time:
		if typed.IsZero() {
			return nil
		}
		value.Time = typed.UTC()
		value.Valid = true
		return nil
	case []byte:
		return value.parse(string(typed))
	case string:
		return value.parse(typed)
	default:
		return fmt.Errorf("unsupported timestamp type %T", source)
	}
}

func (value Time) Value() (driver.Value, error) {
	if !value.Valid {
		return nil, nil
	}
	return value.Time.UTC(), nil
}

func (value Time) MarshalJSON() ([]byte, error) {
	if !value.Valid {
		if value.raw != "" {
			return json.Marshal(value.raw)
		}
		return []byte("null"), nil
	}
	return json.Marshal(value.Time.UTC().Format(time.RFC3339Nano))
}

func (value *Time) UnmarshalJSON(data []byte) error {
	var text *string
	if err := json.Unmarshal(data, &text); err != nil {
		return err
	}
	*value = Time{}
	if text == nil {
		return nil
	}
	return value.parse(*text)
}

func (value Time) String() string {
	if !value.Valid {
		return value.raw
	}
	return value.Time.UTC().Format(time.RFC3339Nano)
}

func (value *Time) parse(text string) error {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}
	// Go's String() output for a monotonic reading carries an " m=+0.00" suffix
	// that no layout matches.
	if index := strings.Index(trimmed, " m="); index >= 0 {
		trimmed = strings.TrimSpace(trimmed[:index])
	}
	for _, layout := range layouts {
		// A layout with no zone parses as UTC, which is what the column holds:
		// every writer here stores UTC, and CURRENT_TIMESTAMP is UTC in both
		// dialects.
		if parsed, err := time.Parse(layout, trimmed); err == nil {
			value.Time = parsed.UTC()
			value.Valid = true
			return nil
		}
	}
	// Not a recognised timestamp. Keep it rather than failing the whole scan.
	value.raw = text
	return nil
}

// Normalise converts a value already scanned into an `any` - the shape used by
// handlers that build a response map directly.
func Normalise(value any) any {
	var stamp Time
	if err := stamp.Scan(value); err != nil {
		return value
	}
	if !stamp.Valid {
		if stamp.raw != "" {
			return stamp.raw
		}
		return nil
	}
	return stamp.Time.Format(time.RFC3339Nano)
}
