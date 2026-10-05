package main

import (
	"fmt"
	"time"

	"kanpic/internal/automation"
)

func main() {
	base := time.Date(2026, 10, 6, 7, 0, 0, 0, time.UTC)
	exprs := []string{
		"+5 0 * * *",
		"-0 0 * * *",
		"+0 +0 * * *",
		"5 0 * * +1",
		"0 0 +1 +1 *",
		"0 0 1 1 +7",
		"+1-+3 0 * * *",
		"0 0 * * +0",
		"5 0 * * *",
		"*/15 * * * *",
		"0 0 * * MON-FRI",
		"@daily",
		"0 0 * * 7",
		"0 0 31 2 *",
	}
	for _, expr := range exprs {
		s, err := automation.ParseSchedule(expr, "UTC")
		if err != nil {
			fmt.Printf("%-18s REJECT err=%v\n", expr, err)
			continue
		}
		next, nerr := s.Next(base)
		fmt.Printf("%-18s ACCEPT stored=%s next=%v nexterr=%v\n", expr, s.Expression, next, nerr)
	}
}
