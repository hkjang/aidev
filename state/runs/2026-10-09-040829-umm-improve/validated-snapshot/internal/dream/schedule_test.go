package dream

import (
	"testing"
	"time"
)

func TestScheduledToday(t *testing.T) {
	monday := time.Date(2026, 8, 10, 2, 0, 0, 0, time.UTC)
	saturday := time.Date(2026, 8, 15, 2, 0, 0, 0, time.UTC)
	if !scheduledToday(Config{Frequency: "weekdays"}, monday) || scheduledToday(Config{Frequency: "weekdays"}, saturday) {
		t.Fatal("weekday schedule mismatch")
	}
	if !scheduledToday(Config{Frequency: "custom", CustomDays: []int{1, 3}}, monday) || scheduledToday(Config{Frequency: "custom", CustomDays: []int{2, 3}}, monday) {
		t.Fatal("custom weekday schedule mismatch")
	}
	interval := Config{Frequency: "interval", IntervalDays: 3}
	runs := 0
	for day := 0; day < 3; day++ {
		if scheduledToday(interval, monday.AddDate(0, 0, day)) {
			runs++
		}
	}
	if runs != 1 {
		t.Fatal("interval schedule must select one day per interval")
	}
}
