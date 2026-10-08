package auth

import (
	"database/sql/driver"
	"fmt"
	"strings"
	"time"
)

type flexibleTime struct {
	Time  time.Time
	Valid bool
}

func (value *flexibleTime) Scan(source any) error {
	if source == nil {
		value.Time = time.Time{}
		value.Valid = false
		return nil
	}
	switch typed := source.(type) {
	case time.Time:
		value.Time = typed.UTC()
		value.Valid = true
		return nil
	case string:
		return value.parse(typed)
	case []byte:
		return value.parse(string(typed))
	default:
		return fmt.Errorf("unsupported timestamp type %T", source)
	}
}

func (value *flexibleTime) parse(text string) error {
	if index := strings.Index(text, " m="); index >= 0 {
		text = text[:index]
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		"2006-01-02 15:04:05.999999999Z07:00",
		"2006-01-02 15:04:05Z07:00",
		"2006-01-02 15:04:05.999999999 -0700 MST",
		"2006-01-02 15:04:05 -0700 MST",
		"2006-01-02 15:04:05",
	} {
		if parsed, err := time.Parse(layout, text); err == nil {
			value.Time = parsed.UTC()
			value.Valid = true
			return nil
		}
	}
	return fmt.Errorf("invalid timestamp %q", text)
}

func (value flexibleTime) Value() (driver.Value, error) {
	if !value.Valid {
		return nil, nil
	}
	return value.Time.UTC(), nil
}
