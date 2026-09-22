package server

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"unicode/utf8"
)

func (s *Server) versionSnapshot(r *http.Request, id string, version int) (map[string]any, error) {
	if !s.canDocument(r.Context(), current(r), id, false) {
		return nil, errors.New("문서 이력 접근 권한이 없습니다")
	}
	return s.one(r.Context(), `SELECT jsonb_build_object('document_id',v.document_id,'version',v.version,'title',v.title,'markdown',v.markdown,'tags',v.tags,'block_metadata',v.block_metadata,'created_at',v.created_at,'user_name',u.name,'current_version',d.version) FROM document_versions v JOIN documents d ON d.id=v.document_id JOIN users u ON u.id=v.user_id WHERE v.document_id=$2 AND v.version=$3 AND madi_document_allowed($1,d.id,false) AND ($4='' OR d.workspace_id::text=$4)`, current(r).ID, id, version, current(r).WorkspaceID)
}
func (s *Server) documentVersionDetail(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	version, err := strconv.Atoi(r.PathValue("version"))
	if err != nil || version < 1 || version > 2147483647 {
		apiError(w, 400, "문서 이력 버전을 확인하세요")
		return
	}
	if !s.canDocument(r.Context(), current(r), id, false) {
		apiError(w, 403, "문서 이력 접근 권한이 없습니다")
		return
	}
	value, err := s.versionSnapshot(r, id, version)
	if err != nil {
		apiError(w, 404, "접근 가능한 문서 버전을 찾을 수 없습니다")
		return
	}
	jsonResponse(w, 200, value)
}
func (s *Server) documentVersionDiff(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	from, err := strconv.Atoi(r.URL.Query().Get("from"))
	to, err2 := strconv.Atoi(r.URL.Query().Get("to"))
	if err != nil || err2 != nil || from < 1 || to < 1 || from > 2147483647 || to > 2147483647 {
		apiError(w, 400, "비교할 두 문서 버전을 선택하세요")
		return
	}
	if !s.canDocument(r.Context(), current(r), id, false) {
		apiError(w, 403, "문서 이력 접근 권한이 없습니다")
		return
	}
	before, err := s.versionSnapshot(r, id, from)
	if err != nil {
		apiError(w, 404, "이전 버전을 찾을 수 없습니다")
		return
	}
	after, err := s.versionSnapshot(r, id, to)
	if err != nil {
		apiError(w, 404, "다음 버전을 찾을 수 없습니다")
		return
	}
	diff := boundedDocumentDiff(str(before, "markdown"), str(after, "markdown"))
	// Recheck after potentially expensive diff work; never publish stale access.
	if !s.canDocument(r.Context(), current(r), id, false) {
		apiError(w, 403, "문서 이력 접근 권한이 변경되었습니다")
		return
	}
	jsonResponse(w, 200, map[string]any{"from": from, "to": to, "current_version": after["current_version"], "title": map[string]any{"before": before["title"], "after": after["title"]}, "tags": map[string]any{"before": before["tags"], "after": after["tags"]}, "diff": diff})
}

type documentDiffRow struct {
	Kind    string `json:"kind"`
	Text    string `json:"text"`
	OldLine int    `json:"old_line,omitempty"`
	NewLine int    `json:"new_line,omitempty"`
	Count   int    `json:"count,omitempty"`
}
type documentDiff struct {
	Rows        []documentDiffRow `json:"rows"`
	Added       int               `json:"added"`
	Removed     int               `json:"removed"`
	Truncated   bool              `json:"truncated"`
	Coarse      bool              `json:"coarse"`
	Notice      string            `json:"notice"`
	BeforeBytes int               `json:"before_bytes"`
	AfterBytes  int               `json:"after_bytes"`
}

func boundedDocumentDiff(before, after string) documentDiff {
	result := documentDiff{Rows: []documentDiffRow{}, BeforeBytes: len(before), AfterBytes: len(after)}
	if strings.Count(before, "\n") > 50000 || strings.Count(after, "\n") > 50000 {
		result.Truncated = true
		result.Notice = "50,000줄을 초과한 문서는 전체 원문을 내려받아 비교하세요. 원문은 변경하지 않았습니다."
		return result
	}
	split := func(s string) []string {
		if s == "" {
			return []string{}
		}
		out := strings.SplitAfter(s, "\n")
		if len(out) > 0 && out[len(out)-1] == "" {
			out = out[:len(out)-1]
		}
		return out
	}
	a, b := split(before), split(after)
	prefix := 0
	for prefix < len(a) && prefix < len(b) && a[prefix] == b[prefix] {
		prefix++
	}
	suffix := 0
	for suffix < len(a)-prefix && suffix < len(b)-prefix && a[len(a)-1-suffix] == b[len(b)-1-suffix] {
		suffix++
	}
	left, right := a[prefix:len(a)-suffix], b[prefix:len(b)-suffix]
	changes, ok := documentMyersDiff(left, right)
	if !ok {
		result.Coarse = true
		for _, line := range left {
			changes = append(changes, documentDiffRow{Kind: "remove", Text: line})
		}
		for _, line := range right {
			changes = append(changes, documentDiffRow{Kind: "add", Text: line})
		}
	}
	rows := make([]documentDiffRow, 0, prefix+suffix+len(changes))
	for _, line := range a[:prefix] {
		rows = append(rows, documentDiffRow{Kind: "equal", Text: line})
	}
	rows = append(rows, changes...)
	for _, line := range a[len(a)-suffix:] {
		rows = append(rows, documentDiffRow{Kind: "equal", Text: line})
	}
	oldLine, newLine := 0, 0
	for i := range rows {
		switch rows[i].Kind {
		case "equal":
			oldLine++
			newLine++
			rows[i].OldLine = oldLine
			rows[i].NewLine = newLine
		case "remove":
			oldLine++
			result.Removed++
			rows[i].OldLine = oldLine
		case "add":
			newLine++
			result.Added++
			rows[i].NewLine = newLine
		}
	}
	// Keep three context lines on either side, and bound both output count and
	// serialized text. Very long lines are sliced only on UTF-8 boundaries.
	compacted := []documentDiffRow{}
	for i := 0; i < len(rows); {
		j := i
		for j < len(rows) && rows[j].Kind == "equal" {
			j++
		}
		if j-i > 6 {
			compacted = append(compacted, rows[i:i+3]...)
			compacted = append(compacted, documentDiffRow{Kind: "skip", Count: j - i - 6})
			compacted = append(compacted, rows[j-3:j]...)
			i = j
		} else {
			compacted = append(compacted, rows[i])
			i++
		}
	}
	bytes := 0
	for _, row := range compacted {
		if len(result.Rows) >= 2000 || bytes+len(row.Text) > 512<<10 {
			result.Truncated = true
			break
		}
		if len(row.Text) > 8192 {
			end := 8192
			for end > 0 && !utf8.RuneStart(row.Text[end]) {
				end--
			}
			row.Text = row.Text[:end] + "…"
			result.Truncated = true
		}
		bytes += len(row.Text)
		result.Rows = append(result.Rows, row)
	}
	if result.Coarse {
		result.Notice = "비교 연산 한도에 도달해 변경 구간 전체를 표시합니다. 원문은 변경하지 않았습니다."
	}
	if result.Truncated {
		result.Notice += " 화면은 최대 2,000줄·512KB·한 줄 8KB로 제한합니다. 전체 원문을 내려받아 비교할 수 있습니다."
	}
	return result
}

// Myers shortest edit script, bounded by a shared step/trace-entry budget. The
// fallback is still a valid replacement script, but intentionally not minimal.
func documentMyersDiff(a, b []string) ([]documentDiffRow, bool) {
	if len(a) == 0 {
		out := make([]documentDiffRow, 0, len(b))
		for _, s := range b {
			out = append(out, documentDiffRow{Kind: "add", Text: s})
		}
		return out, true
	}
	if len(b) == 0 {
		out := make([]documentDiffRow, 0, len(a))
		for _, s := range a {
			out = append(out, documentDiffRow{Kind: "remove", Text: s})
		}
		return out, true
	}
	v := map[int]int{1: 0}
	trace := []map[int]int{}
	budget := 400000
	for d := 0; d <= len(a)+len(b); d++ {
		previous := map[int]int{}
		for k, x := range v {
			previous[k] = x
			budget--
		}
		trace = append(trace, previous)
		for k := -d; k <= d; k += 2 {
			budget--
			if budget < 0 {
				return nil, false
			}
			x := 0
			if k == -d || (k != d && v[k-1] < v[k+1]) {
				x = v[k+1]
			} else {
				x = v[k-1] + 1
			}
			y := x - k
			for x < len(a) && y < len(b) && a[x] == b[y] {
				x++
				y++
				budget--
				if budget < 0 {
					return nil, false
				}
			}
			v[k] = x
			if x >= len(a) && y >= len(b) {
				return backtrackDocumentDiff(a, b, trace), true
			}
		}
	}
	return nil, false
}
func backtrackDocumentDiff(a, b []string, trace []map[int]int) []documentDiffRow {
	x, y := len(a), len(b)
	out := []documentDiffRow{}
	for d := len(trace) - 1; d >= 0; d-- {
		v := trace[d]
		k := x - y
		previousK := 0
		if k == -d || (k != d && v[k-1] < v[k+1]) {
			previousK = k + 1
		} else {
			previousK = k - 1
		}
		previousX := v[previousK]
		previousY := previousX - previousK
		for x > previousX && y > previousY {
			out = append(out, documentDiffRow{Kind: "equal", Text: a[x-1]})
			x--
			y--
		}
		if d > 0 {
			if x == previousX {
				out = append(out, documentDiffRow{Kind: "add", Text: b[y-1]})
				y--
			} else {
				out = append(out, documentDiffRow{Kind: "remove", Text: a[x-1]})
				x--
			}
		}
	}
	for i, j := 0, len(out)-1; i < j; i, j = i+1, j-1 {
		out[i], out[j] = out[j], out[i]
	}
	return out
}
