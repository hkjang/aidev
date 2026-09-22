package server

import (
	"sync"
	"time"
)

var runbookPreparationLimits = struct {
	sync.Mutex
	entries map[string]attempt
}{entries: map[string]attempt{}}

// The per-process preflight rate is deliberately separate from login limiting.
// It is bounded in memory; execution capacity itself is also checked in SQL.
func allowRunbookPreparation(uid string, now time.Time) bool {
	l := &runbookPreparationLimits
	l.Lock()
	defer l.Unlock()
	if len(l.entries) >= 10000 {
		for key, v := range l.entries {
			if !now.Before(v.until) {
				delete(l.entries, key)
			}
		}
	}
	v, exists := l.entries[uid]
	if !exists && len(l.entries) >= 10000 {
		return false
	}
	if !now.Before(v.until) {
		v = attempt{until: now.Add(time.Minute)}
	}
	if v.count >= 10 {
		return false
	}
	v.count++
	l.entries[uid] = v
	return true
}
