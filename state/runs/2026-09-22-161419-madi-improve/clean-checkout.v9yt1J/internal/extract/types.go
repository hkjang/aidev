// Package extract reads a bounded, non-executable projection of attachments.
// Source bytes are never rewritten and source locations are kept with each span.
package extract

import (
	"errors"
	"strings"
	"unicode/utf8"
)

type Position struct {
	Page      int       `json:"page,omitempty"`
	Slide     int       `json:"slide,omitempty"`
	Sheet     string    `json:"sheet,omitempty"`
	Cell      string    `json:"cell,omitempty"`
	Paragraph int       `json:"paragraph,omitempty"`
	Table     int       `json:"table,omitempty"`
	Row       int       `json:"row,omitempty"`
	Column    int       `json:"column,omitempty"`
	Bounds    []float64 `json:"bounds,omitempty"` // x0,y0,x1,y1, in source-page points.
	PageSize  []float64 `json:"page_size,omitempty"`
	OCR       bool      `json:"ocr,omitempty"`
}

type Fragment struct {
	Text     string   `json:"text"`
	Position Position `json:"position"`
}

type Result struct {
	Fragments  []Fragment `json:"fragments"`
	Warnings   []string   `json:"warnings"`
	Pages      int        `json:"pages"`
	EmptyPages []int      `json:"empty_pages"`
}

type Limits struct {
	InputBytes    int64
	ExpandedBytes int64
	TextBytes     int
	Fragments     int
	Pages         int
}

func DefaultLimits() Limits {
	return Limits{InputBytes: 50 << 20, ExpandedBytes: 100 << 20, TextBytes: 8 << 20, Fragments: 20000, Pages: 500}
}

var ErrLimit = errors.New("첨부 본문 추출 한도를 초과했습니다")

type collector struct {
	result Result
	limits Limits
	bytes  int
}

func newCollector(limits Limits) *collector {
	return &collector{result: Result{Fragments: []Fragment{}, Warnings: []string{}, EmptyPages: []int{}}, limits: limits}
}

func (c *collector) add(text string, position Position) error {
	text = strings.TrimSpace(strings.ReplaceAll(text, "\x00", ""))
	if text == "" {
		return nil
	}
	if !utf8.ValidString(text) {
		return errors.New("첨부 본문이 올바른 UTF-8 문자열이 아닙니다")
	}
	// A fragment is independently addressable. Bound each excerpt, splitting
	// only at UTF-8 boundaries while retaining its original source position.
	for text != "" {
		n := len(text)
		if n > 16<<10 {
			n = 16 << 10
			for n > 0 && !utf8.RuneStart(text[n]) {
				n--
			}
		}
		if c.bytes+n > c.limits.TextBytes || len(c.result.Fragments) >= c.limits.Fragments {
			return ErrLimit
		}
		c.result.Fragments = append(c.result.Fragments, Fragment{Text: text[:n], Position: position})
		c.bytes += n
		text = text[n:]
	}
	return nil
}
