package automation

import (
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"
)

const maxScheduleLookaheadYears = 8

// Vixie cron 의 표준 별칭. @reboot 은 영구 저장된 next_run_at 으로 다음 시각을 미리 정하는
// 이 스케줄러에서 뜻이 없으므로 일부러 빼 두었다.
var cronAliases = map[string]string{
	"@hourly":   "0 * * * *",
	"@daily":    "0 0 * * *",
	"@midnight": "0 0 * * *",
	"@weekly":   "0 0 * * 0",
	"@monthly":  "0 0 1 * *",
	"@yearly":   "0 0 1 1 *",
	"@annually": "0 0 1 1 *",
}

var monthNames = map[string]int{
	"JAN": 1, "FEB": 2, "MAR": 3, "APR": 4, "MAY": 5, "JUN": 6,
	"JUL": 7, "AUG": 8, "SEP": 9, "OCT": 10, "NOV": 11, "DEC": 12,
}

var weekdayNames = map[string]int{
	"SUN": 0, "MON": 1, "TUE": 2, "WED": 3, "THU": 4, "FRI": 5, "SAT": 6,
}

type cronField struct {
	allowed []bool
	// unrestricted 는 필드가 별표로 시작해 값을 제한하지 않는다는 뜻이다(`*`, `*/2`).
	unrestricted bool
}

type Schedule struct {
	Expression string
	Timezone   string
	location   *time.Location
	minute     cronField
	hour       cronField
	day        cronField
	month      cronField
	weekday    cronField
}

func ParseSchedule(expression, timezone string) (*Schedule, error) {
	expression = strings.TrimSpace(expression)
	lowered := strings.ToLower(expression)
	if alias, ok := cronAliases[lowered]; ok {
		expression = alias
	} else if strings.HasPrefix(lowered, "@") {
		// 별칭을 쓰려 한 입력에 "다섯 필드" 를 말해 주면 사용자는 다섯 필드 식을 쓴 적이 없으므로
		// 오타인지 미지원 별칭인지 구분할 수 없다. 지원 목록을 그대로 돌려준다.
		return nil, fmt.Errorf("%w: unknown schedule alias %q, supported aliases are %s", ErrInvalid, expression, strings.Join(cronAliasNames(), ", "))
	}
	parts := strings.Fields(expression)
	if len(parts) != 5 {
		return nil, fmt.Errorf("%w: schedule cron must contain five fields", ErrInvalid)
	}
	timezone = strings.TrimSpace(timezone)
	if timezone == "" {
		timezone = "UTC"
	}
	location, err := time.LoadLocation(timezone)
	if err != nil {
		return nil, fmt.Errorf("%w: unknown schedule timezone %q", ErrInvalid, timezone)
	}
	minute, err := parseCronField(parts[0], 0, 59, nil, false)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid schedule minute: %v", ErrInvalid, err)
	}
	hour, err := parseCronField(parts[1], 0, 23, nil, false)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid schedule hour: %v", ErrInvalid, err)
	}
	day, err := parseCronField(parts[2], 1, 31, nil, false)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid schedule day: %v", ErrInvalid, err)
	}
	month, err := parseCronField(parts[3], 1, 12, monthNames, false)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid schedule month: %v", ErrInvalid, err)
	}
	weekday, err := parseCronField(parts[4], 0, 7, weekdayNames, true)
	if err != nil {
		return nil, fmt.Errorf("%w: invalid schedule weekday: %v", ErrInvalid, err)
	}
	return &Schedule{Expression: strings.Join(parts, " "), Timezone: timezone, location: location, minute: minute, hour: hour, day: day, month: month, weekday: weekday}, nil
}

// cronAliasNames 는 오류 문구가 맵 순회 순서로 흔들리지 않게 지원 별칭을 정렬해 돌려준다.
func cronAliasNames() []string {
	names := make([]string, 0, len(cronAliases))
	for name := range cronAliases {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func (s *Schedule) Next(after time.Time) (time.Time, error) {
	localAfter := after.In(s.location)
	start := localAfter.Truncate(time.Minute).Add(time.Minute)
	limit := start.AddDate(maxScheduleLookaheadYears, 0, 0)
	for date := time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, s.location); date.Before(limit); date = time.Date(date.Year(), date.Month(), date.Day()+1, 0, 0, 0, 0, s.location) {
		if !s.month.allowed[int(date.Month())] || !s.matchesDay(date) {
			continue
		}
		for hour := 0; hour <= 23; hour++ {
			if !s.hour.allowed[hour] {
				continue
			}
			for minute := 0; minute <= 59; minute++ {
				if !s.minute.allowed[minute] {
					continue
				}
				candidate := time.Date(date.Year(), date.Month(), date.Day(), hour, minute, 0, 0, s.location)
				local := candidate.In(s.location)
				if local.Year() != date.Year() || local.Month() != date.Month() || local.Day() != date.Day() || local.Hour() != hour || local.Minute() != minute {
					continue
				}
				if !candidate.Before(start) && candidate.After(after) {
					return candidate.UTC(), nil
				}
			}
		}
	}
	return time.Time{}, fmt.Errorf("%w: schedule has no occurrence within %d years", ErrInvalid, maxScheduleLookaheadYears)
}

func (s *Schedule) matchesDay(date time.Time) bool {
	dayMatch := s.day.allowed[date.Day()]
	weekdayMatch := s.weekday.allowed[int(date.Weekday())]
	// crontab(5) 의 규칙: 일·요일 두 필드가 **모두 제한되어 있을 때만** 어느 한쪽이 맞으면 실행한다.
	// 한쪽이 별표로 시작하면 그 필드는 아무 날도 걸러내지 않으므로 AND 로 합쳐야 한다 —
	// 그래야 `0 0 */2 * MON` 이 "격일 중 월요일" 로 읽힌다.
	if s.day.unrestricted || s.weekday.unrestricted {
		return dayMatch && weekdayMatch
	}
	return dayMatch || weekdayMatch
}

func parseCronField(raw string, minimum, maximum int, names map[string]int, sundaySeven bool) (cronField, error) {
	// Vixie cron 은 필드 전체의 첫 글자만 보므로 `1,*/2` 처럼 다른 값이 앞서면 제한으로 남는다.
	field := cronField{allowed: make([]bool, maximum+1), unrestricted: strings.HasPrefix(raw, "*")}
	for _, segment := range strings.Split(strings.ToUpper(raw), ",") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			return cronField{}, fmt.Errorf("empty list item")
		}
		base, stepRaw, hasStep := strings.Cut(segment, "/")
		step := 1
		if hasStep {
			if strings.Contains(stepRaw, "/") {
				return cronField{}, fmt.Errorf("too many step separators")
			}
			value, err := strconv.Atoi(stepRaw)
			if err != nil || value < 1 || value > maximum-minimum+1 {
				return cronField{}, fmt.Errorf("step must be between 1 and %d", maximum-minimum+1)
			}
			step = value
		}
		start, end := minimum, maximum
		switch {
		case base == "*":
		case strings.Contains(base, "-"):
			left, right, ok := strings.Cut(base, "-")
			if !ok || strings.Contains(right, "-") {
				return cronField{}, fmt.Errorf("invalid range %q", base)
			}
			var err error
			start, err = cronValue(left, minimum, maximum, names)
			if err != nil {
				return cronField{}, err
			}
			end, err = cronValue(right, minimum, maximum, names)
			if err != nil {
				return cronField{}, err
			}
			if start > end {
				return cronField{}, fmt.Errorf("range start exceeds end")
			}
		default:
			value, err := cronValue(base, minimum, maximum, names)
			if err != nil {
				return cronField{}, err
			}
			start = value
			if !hasStep {
				end = value
			}
		}
		for value := start; value <= end; value += step {
			index := value
			if sundaySeven && value == 7 {
				index = 0
			}
			field.allowed[index] = true
		}
	}
	for _, allowed := range field.allowed {
		if allowed {
			return field, nil
		}
	}
	return cronField{}, fmt.Errorf("field selects no values")
}

func cronValue(raw string, minimum, maximum int, names map[string]int) (int, error) {
	if value, ok := names[raw]; ok {
		return value, nil
	}
	// strconv.Atoi 는 부호를 허용하므로 cron 문법에 없는 `+5`·`+0` 이 통과한다 — 값 자리는
	// 십진 숫자만이어야 하고, 아니면 같은 범위 오류로 거절한다.
	if !isDecimalDigits(raw) {
		return 0, fmt.Errorf("value %q must be between %d and %d", raw, minimum, maximum)
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		return 0, fmt.Errorf("value %q must be between %d and %d", raw, minimum, maximum)
	}
	return value, nil
}

func isDecimalDigits(raw string) bool {
	if raw == "" {
		return false
	}
	for index := 0; index < len(raw); index++ {
		if raw[index] < '0' || raw[index] > '9' {
			return false
		}
	}
	return true
}
