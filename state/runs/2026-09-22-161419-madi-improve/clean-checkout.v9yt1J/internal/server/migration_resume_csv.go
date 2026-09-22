package server

import (
	"encoding/csv"
	"errors"
	"io"
	"math"
	"math/big"
	"net/mail"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"
)

// The resumable path has an item-sized bound instead of expanding the legacy
// ZIP limits. CSV records are read incrementally; one CSV item is capped at
// 50MiB/100,000 records, with the original retained separately for comparison.
func parseResumeCSV(raw []byte) (migrationCSV, error) {
	out := migrationCSV{}
	if !utf8.Valid(raw) || len(raw) > 50<<20 {
		return out, errors.New("CSV는 UTF-8 형식의 50MiB 이하 항목이어야 합니다")
	}
	r := csv.NewReader(strings.NewReader(strings.TrimPrefix(string(raw), "\ufeff")))
	header, e := r.Read()
	if e != nil || len(header) < 1 || len(header) > 100 {
		return out, errors.New("CSV 헤더는 1~100개 열입니다")
	}
	seen := map[string]bool{}
	for i, v := range header {
		v = strings.TrimSpace(v)
		if v == "" || len(v) > 100 || seen[v] {
			return out, errors.New("CSV 열 이름은 중복 없는 1~100바이트 문자열입니다")
		}
		header[i] = v
		seen[v] = true
	}
	out.Header = header
	for {
		row, e := r.Read()
		if e == io.EOF {
			break
		}
		if e != nil {
			return out, errors.New("CSV 행의 열 수 또는 인용부호가 올바르지 않습니다")
		}
		if len(out.Rows) >= 100000 {
			return out, errors.New("CSV 항목은 최대 100,000행이며 더 큰 자료는 원본 ID를 유지한 여러 CSV 항목으로 나누세요")
		}
		for _, v := range row {
			if len(v) > 64<<10 {
				return out, errors.New("CSV 셀은 64KiB 이하여야 합니다")
			}
		}
		out.Rows = append(out.Rows, row)
	}
	return out, nil
}

func migrationNumber(v string) (float64, bool) {
	// Leading zeroes, whitespace and unsafe integer precision are meaningful
	// identifiers rather than numbers. Inference must never silently erase them.
	if v == "" || strings.TrimSpace(v) != v {
		return 0, false
	}
	unsigned := strings.TrimPrefix(strings.TrimPrefix(v, "-"), "+")
	if len(unsigned) > 1 && unsigned[0] == '0' && unsigned[1] != '.' {
		return 0, false
	}
	f, e := strconv.ParseFloat(v, 64)
	if e != nil || math.IsNaN(f) || math.IsInf(f, 0) || math.Abs(f) > 9007199254740991 {
		return 0, false
	}
	original, ok := new(big.Rat).SetString(v)
	if !ok {
		return 0, false
	}
	canonical, ok := new(big.Rat).SetString(strconv.FormatFloat(f, 'g', -1, 64))
	if !ok || original.Cmp(canonical) != 0 {
		return 0, false
	}
	return f, true
}

func migrationCheckbox(v string) (bool, bool) {
	switch strings.ToLower(v) {
	case "true", "yes", "참", "예":
		return true, true
	case "false", "no", "거짓", "아니요":
		return false, true
	}
	return false, false
}

func migrationDate(v string) bool {
	if _, e := time.Parse("2006-01-02", v); e == nil {
		return true
	}
	_, e := time.Parse(time.RFC3339, v)
	return e == nil
}

func inferMigrationCSV(data migrationCSV) []migrationCSVColumn {
	cols := []migrationCSVColumn{}
	for index, name := range data.Header {
		c := migrationCSVColumn{Name: name, Type: "text", Options: []string{}, Samples: []string{}, Reason: "원문을 보존하는 텍스트"}
		allNumber, allBool, allDate, allURL, allEmail := true, true, true, true, true
		distinct := map[string]bool{}
		count := 0
		selectSafe := true
		for _, row := range data.Rows {
			v := row[index]
			if v == "" {
				c.Empty++
				continue
			}
			count++
			if len(c.Samples) < 5 {
				c.Samples = append(c.Samples, v)
			}
			if _, ok := migrationNumber(v); !ok {
				allNumber = false
			}
			if _, ok := migrationCheckbox(v); !ok {
				allBool = false
			}
			if !migrationDate(v) {
				allDate = false
			}
			u, e := url.Parse(v)
			if e != nil || !oneOf(u.Scheme, "http", "https") || u.Host == "" {
				allURL = false
			}
			if a, e := mail.ParseAddress(v); e != nil || a.Address != v {
				allEmail = false
			}
			if len(v) > 200 || strings.TrimSpace(v) == "" {
				selectSafe = false
			}
			if len(distinct) < 21 {
				distinct[v] = true
			}
		}
		switch {
		case count == 0:
			c.Reason = "값이 없어 텍스트로 제안합니다"
		case allBool:
			c.Type = "checkbox"
			c.Reason = "비어 있지 않은 모든 값이 명시적 참/거짓입니다"
		case allNumber:
			c.Type = "number"
			c.Reason = "선행 0·공백·정밀도 손실 없는 숫자입니다"
		case allDate:
			c.Type = "date"
			c.Reason = "모든 값이 ISO 날짜 또는 시간입니다"
		case allURL:
			c.Type = "url"
			c.Reason = "모든 값이 HTTP(S) 주소입니다"
		case allEmail:
			c.Type = "email"
			c.Reason = "모든 값이 이메일 주소 형식입니다"
		case selectSafe && len(distinct) > 0 && len(distinct) <= 20 && count >= len(distinct)*2:
			c.Type = "select"
			c.Reason = "20개 이하의 반복 값입니다. 선택 옵션을 검토하세요"
			for v := range distinct {
				c.Options = append(c.Options, v)
			}
			sort.Strings(c.Options)
		}
		cols = append(cols, c)
	}
	return cols
}

func convertMigrationCSVCell(raw string, column migrationCSVColumn) (any, error) {
	if raw == "" {
		return nil, nil
	}
	switch column.Type {
	case "text", "phone":
		return raw, nil
	case "number":
		if v, ok := migrationNumber(raw); ok {
			return v, nil
		}
	case "checkbox":
		if v, ok := migrationCheckbox(raw); ok {
			return v, nil
		}
	case "date":
		if migrationDate(raw) {
			return raw, nil
		}
	case "select":
		if oneOf(raw, column.Options...) {
			return raw, nil
		}
	case "url":
		u, e := url.Parse(raw)
		if e == nil && oneOf(u.Scheme, "http", "https") && u.Host != "" {
			return raw, nil
		}
	case "email":
		if a, e := mail.ParseAddress(raw); e == nil && a.Address == raw {
			return raw, nil
		}
	}
	return nil, errors.New("선택한 열 타입으로 원문을 손실 없이 변환할 수 없습니다")
}

func validateMigrationCSVColumns(data migrationCSV, columns []migrationCSVColumn) error {
	if len(columns) != len(data.Header) {
		return errors.New("CSV 열별 타입을 모두 확인하세요")
	}
	for i, c := range columns {
		if c.Name != data.Header[i] || !oneOf(c.Type, "text", "number", "checkbox", "date", "select", "url", "email", "phone") || len(c.Options) > 100 {
			return errors.New("CSV 열 이름·타입·옵션을 확인하세요")
		}
		seen := map[string]bool{}
		for _, v := range c.Options {
			if strings.TrimSpace(v) == "" || len(v) > 200 || seen[v] {
				return errors.New("선택 옵션은 중복 없는 1~200바이트 문자열입니다")
			}
			seen[v] = true
		}
		for _, row := range data.Rows {
			if _, e := convertMigrationCSVCell(row[i], c); e != nil {
				return errors.New(c.Name + ": " + e.Error())
			}
		}
	}
	return nil
}
