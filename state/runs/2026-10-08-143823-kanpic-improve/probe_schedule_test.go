package automation

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestScheduleNextSupportsStepsRangesNamesAndTimezone(t *testing.T) {
	schedule, err := ParseSchedule("*/15 9-17 * * MON-FRI", "Asia/Seoul")
	if err != nil {
		t.Fatal(err)
	}
	after := time.Date(2026, time.August, 3, 8, 59, 30, 0, time.FixedZone("KST", 9*60*60))
	next, err := schedule.Next(after)
	if err != nil {
		t.Fatal(err)
	}
	want := time.Date(2026, time.August, 3, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) || schedule.Expression != "*/15 9-17 * * MON-FRI" || schedule.Timezone != "Asia/Seoul" {
		t.Fatalf("next=%s schedule=%#v", next, schedule)
	}
	next, err = schedule.Next(time.Date(2026, time.August, 3, 8, 59, 0, 0, time.UTC))
	if err != nil {
		t.Fatal(err)
	}
	want = time.Date(2026, time.August, 4, 0, 0, 0, 0, time.UTC)
	if !next.Equal(want) {
		t.Fatalf("next workday=%s, want %s", next, want)
	}
}

func TestScheduleAliasesLeapDayAndCronDayOrSemantics(t *testing.T) {
	daily, err := ParseSchedule("@daily", "UTC")
	if err != nil || daily.Expression != "0 0 * * *" {
		t.Fatalf("daily=%#v, %v", daily, err)
	}
	leap, err := ParseSchedule("0 9 29 FEB *", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	next, err := leap.Next(time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || !next.Equal(time.Date(2028, 2, 29, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("leap next=%s, %v", next, err)
	}
	orSchedule, err := ParseSchedule("0 9 1 * MON", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	next, err = orSchedule.Next(time.Date(2026, 8, 2, 12, 0, 0, 0, time.UTC))
	if err != nil || !next.Equal(time.Date(2026, 8, 3, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("cron OR next=%s, %v", next, err)
	}
	monthName, err := ParseSchedule("0 9 * JAN MON", "UTC")
	if err != nil {
		t.Fatal(err)
	}
	next, err = monthName.Next(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	if err != nil || !next.Equal(time.Date(2026, 1, 5, 9, 0, 0, 0, time.UTC)) {
		t.Fatalf("named month next=%s, %v", next, err)
	}
}

// 일·요일 두 필드를 AND 로 합칠지 OR 로 합칠지는 "두 필드가 모두 제한되어 있는가" 로 갈린다.
// 별표로 시작하는 필드(`*`, `*/2`)는 제한이 아니므로 AND 여야 한다 — 아래 표의 앞 두 사례가
// 그것을 못 박고, 나머지는 OR·한쪽 제한 사례가 함께 그대로 유지되는지 보는 대조군이다.
func TestScheduleCombinesDayAndWeekdayByRestriction(t *testing.T) {
	after := time.Date(2026, 10, 30, 12, 0, 0, 0, time.UTC)
	for _, test := range []struct {
		name       string
		expression string
		want       []string
	}{
		{
			name:       "별표 step 인 일 필드는 요일과 AND",
			expression: "0 0 */2 * MON",
			want: []string{
				"2026-11-09 Mon 00:00", "2026-11-23 Mon 00:00", "2026-12-07 Mon 00:00",
				"2026-12-21 Mon 00:00", "2027-01-11 Mon 00:00",
			},
		},
		{
			name:       "별표 step 인 요일 필드는 일과 AND",
			expression: "0 0 1 * */2",
			want: []string{
				"2026-11-01 Sun 00:00", "2026-12-01 Tue 00:00", "2027-04-01 Thu 00:00",
				"2027-05-01 Sat 00:00", "2027-06-01 Tue 00:00",
			},
		},
		{
			name:       "두 필드가 모두 제한이면 OR 로 남는다",
			expression: "0 0 1-31 * MON",
			want: []string{
				"2026-10-31 Sat 00:00", "2026-11-01 Sun 00:00", "2026-11-02 Mon 00:00",
				"2026-11-03 Tue 00:00", "2026-11-04 Wed 00:00",
			},
		},
		{
			name:       "일 필드가 별표면 요일만 본다",
			expression: "0 0 * * */3",
			want: []string{
				"2026-10-31 Sat 00:00", "2026-11-01 Sun 00:00", "2026-11-04 Wed 00:00",
				"2026-11-07 Sat 00:00", "2026-11-08 Sun 00:00",
			},
		},
		{
			name:       "요일 이름만 제한하면 그 요일에만 돈다",
			expression: "* * * * MON",
			want: []string{
				"2026-11-02 Mon 00:00", "2026-11-02 Mon 00:01", "2026-11-02 Mon 00:02",
				"2026-11-02 Mon 00:03", "2026-11-02 Mon 00:04",
			},
		},
		{
			name:       "요일 필드가 별표면 일만 본다",
			expression: "0 9 1 * *",
			want: []string{
				"2026-11-01 Sun 09:00", "2026-12-01 Tue 09:00", "2027-01-01 Fri 09:00",
				"2027-02-01 Mon 09:00", "2027-03-01 Mon 09:00",
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			schedule, err := ParseSchedule(test.expression, "UTC")
			if err != nil {
				t.Fatal(err)
			}
			got := make([]string, 0, len(test.want))
			cursor := after
			for range test.want {
				next, err := schedule.Next(cursor)
				if err != nil {
					t.Fatalf("Next(%s): %v", cursor, err)
				}
				got = append(got, next.Format("2006-01-02 Mon 15:04"))
				cursor = next
			}
			for index := range test.want {
				if got[index] != test.want[index] {
					t.Fatalf("%q 다음 실행=%v, want %v", test.expression, got, test.want)
				}
			}
		})
	}
}

// 별칭은 "같은 뜻의 다섯 필드 식과 실제 실행 시각이 같은가" 로만 확인할 수 있다 —
// 맵에 문자열이 있다는 사실은 Next 가 같은 시각을 돌려준다는 증명이 아니므로
// 두 ParseSchedule 결과를 연속 4회 돌려 반환 시각을 그대로 비교한다.
func TestScheduleAliasesMatchEquivalentCronExpressions(t *testing.T) {
	after := time.Date(2026, 10, 30, 12, 34, 56, 0, time.UTC)
	for _, test := range []struct{ alias, equivalent string }{
		{"@hourly", "0 * * * *"},
		{"@daily", "0 0 * * *"},
		{"@midnight", "0 0 * * *"},
		{"@weekly", "0 0 * * 0"},
		{"@monthly", "0 0 1 * *"},
		{"@yearly", "0 0 1 1 *"},
		{"@annually", "0 0 1 1 *"},
		{"@MIDNIGHT", "0 0 * * *"},
		{"  @annually  ", "0 0 1 1 *"},
	} {
		t.Run(test.alias, func(t *testing.T) {
			aliasSchedule, err := ParseSchedule(test.alias, "UTC")
			if err != nil {
				t.Fatalf("ParseSchedule(%q): %v", test.alias, err)
			}
			plainSchedule, err := ParseSchedule(test.equivalent, "UTC")
			if err != nil {
				t.Fatal(err)
			}
			aliasRuns, plainRuns := make([]string, 0, 4), make([]string, 0, 4)
			aliasCursor, plainCursor := after, after
			for range 4 {
				aliasNext, err := aliasSchedule.Next(aliasCursor)
				if err != nil {
					t.Fatalf("alias Next(%s): %v", aliasCursor, err)
				}
				plainNext, err := plainSchedule.Next(plainCursor)
				if err != nil {
					t.Fatalf("plain Next(%s): %v", plainCursor, err)
				}
				aliasRuns = append(aliasRuns, aliasNext.Format("2006-01-02 Mon 15:04"))
				plainRuns = append(plainRuns, plainNext.Format("2006-01-02 Mon 15:04"))
				aliasCursor, plainCursor = aliasNext, plainNext
			}
			for index := range plainRuns {
				if aliasRuns[index] != plainRuns[index] {
					t.Fatalf("%q 다음 실행=%v, %q=%v", test.alias, aliasRuns, test.equivalent, plainRuns)
				}
			}
		})
	}
}

func TestScheduleSkipsNonexistentDSTWallTime(t *testing.T) {
	schedule, err := ParseSchedule("30 2 * * *", "America/New_York")
	if err != nil {
		t.Fatal(err)
	}
	next, err := schedule.Next(time.Date(2026, 3, 8, 5, 0, 0, 0, time.UTC))
	want := time.Date(2026, 3, 9, 6, 30, 0, 0, time.UTC)
	if err != nil || !next.Equal(want) {
		t.Fatalf("DST next=%s, want %s, %v", next, want, err)
	}
}

func TestScheduleRejectsInvalidExpressions(t *testing.T) {
	for _, test := range []struct{ expression, timezone string }{
		{"* * * *", "UTC"},
		{"60 * * * *", "UTC"},
		{"*/0 * * * *", "UTC"},
		{"0 9 * * FUNDAY", "UTC"},
		{"0 9 * * *", "Mars/Base"},
		{"@reboot", "UTC"},
		{"@nope", "UTC"},
	} {
		if _, err := ParseSchedule(test.expression, test.timezone); !errors.Is(err, ErrInvalid) {
			t.Fatalf("ParseSchedule(%q,%q) error=%v", test.expression, test.timezone, err)
		}
	}
}

// cron 문법에는 부호가 없다. strconv.Atoi 는 `+5`·`-0` 을 받아들이므로 오타가 그대로
// Schedule.Expression 에 저장돼 다른 cron 구현과 사람의 눈에 읽히지 않는 식이 영구 기록으로
// 남는다. 저장 시점에 사람이 읽을 수 있는 ErrInvalid 로 돌려줘야 한다.
func TestScheduleRejectsSignedCronValues(t *testing.T) {
	for _, expression := range []string{
		"+5 0 * * *",
		"0 0 +1 +1 *",
		"0 0 1 1 +7",
		"0 0 * * +0",
		"-0 0 * * *",
		"+1-+3 0 * * *",
	} {
		t.Run(expression, func(t *testing.T) {
			schedule, err := ParseSchedule(expression, "UTC")
			if err == nil {
				next, nextErr := schedule.Next(time.Date(2026, 10, 6, 0, 0, 0, 0, time.UTC))
				t.Fatalf("ParseSchedule(%q) 가 받아들였다: expression=%q 다음 실행=%s (%v)", expression, schedule.Expression, next, nextErr)
			}
			if !errors.Is(err, ErrInvalid) {
				t.Fatalf("ParseSchedule(%q) error=%v, want ErrInvalid", expression, err)
			}
			if !strings.Contains(err.Error(), "must be between") {
				t.Fatalf("ParseSchedule(%q) error=%q — 400 응답에 실리는 문구가 값 범위 계열이어야 한다", expression, err.Error())
			}
		})
	}
}

// 모르는 별칭을 "다섯 필드가 아니다" 로 거절하면 다섯 필드 식을 쓴 적이 없는 사용자는
// 오타인지 미지원인지 알 수 없다. @reboot 은 영구 저장된 next_run_at 기반 스케줄러에
// 뜻이 없어 영영 지원하지 않으므로, 두 경우 모두 지원 목록을 문구에 담아 거절한다.
func TestScheduleRejectsUnknownAliasesWithSupportedList(t *testing.T) {
	for _, expression := range []string{"@reboot", "@nope", "@DAILY2"} {
		_, err := ParseSchedule(expression, "UTC")
		if !errors.Is(err, ErrInvalid) {
			t.Fatalf("ParseSchedule(%q) error=%v, want ErrInvalid", expression, err)
		}
		message := err.Error()
		if strings.Contains(message, "five fields") {
			t.Fatalf("ParseSchedule(%q) error=%q — 별칭 입력에 다섯 필드 문구를 쓰면 원인을 알 수 없다", expression, message)
		}
		for _, alias := range []string{"@annually", "@daily", "@hourly", "@midnight", "@monthly", "@weekly", "@yearly"} {
			if !strings.Contains(message, alias) {
				t.Fatalf("ParseSchedule(%q) error=%q — 지원 별칭 %s 가 빠졌다", expression, message, alias)
			}
		}
	}
}

func TestScoutStepProbe(t *testing.T) {
 for _, expression := range []string{"*/+2 0 * * *", "1-5/+2 0 * * *", "1/+2 0 * * *", "0 */+2 * * *", "0 0 */+2 * *", "0 0 1 */+2 *", "0 0 * * MON-FRI/+2", "*/02 0 * * *", "*/-2 0 * * *", "0 0 * * */8"} {
  schedule, err := ParseSchedule(expression, "UTC")
  if err != nil { t.Logf("%q ERROR %v", expression, err); continue }
  next, nextErr := schedule.Next(time.Date(2026, 10, 8, 0, 0, 0, 0, time.UTC))
  t.Logf("%q ACCEPT expression=%q next=%s err=%v", expression, schedule.Expression, next, nextErr)
 }
}

func TestScoutMidnightProbe(t *testing.T) {
 loc, err := time.LoadLocation("America/Santiago"); if err != nil { t.Fatal(err) }
 date := time.Date(2026, 9, 5, 0, 0, 0, 0, loc)
 for i := 0; i < 4; i++ { t.Logf("cursor %d: %s", i, date); date = time.Date(date.Year(), date.Month(), date.Day()+1, 0, 0, 0, 0, loc) }
 schedule, err := ParseSchedule("0 9 * * *", "America/Santiago"); if err != nil { t.Fatal(err) }
 done:=make(chan time.Time, 1)
 go func(){ next,_:=schedule.Next(time.Date(2026,9,5,13,0,0,0,time.UTC)); done<-next }()
 select {case next:= <-done: t.Logf("Next returned %s",next); case <-time.After(200*time.Millisecond): t.Log("Next did not return in 200ms; cursor loop above is evidence of non-progress")}
}
