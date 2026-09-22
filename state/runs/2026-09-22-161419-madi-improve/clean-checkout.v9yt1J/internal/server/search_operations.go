package server

import (
	"context"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"

	"github.com/jackc/pgx/v5"
	"golang.org/x/text/unicode/norm"
)

//go:embed search_operations.sql
var searchOperationsSchema string

type searchDictionaryEntry struct {
	ID        string   `json:"id"`
	Canonical string   `json:"canonical"`
	Aliases   []string `json:"aliases"`
}
type searchInterpretation struct {
	Normalized         string     `json:"normalized"`
	Parts              [][]string `json:"parts"`
	FoldedParts        [][]string `json:"-"`
	GramQuery          string     `json:"-"`
	DictionaryRevision int        `json:"dictionary_revision"`
	MatchedTerms       []string   `json:"matched_terms"`
	Strategy           string     `json:"strategy"`
	Warnings           []string   `json:"warnings"`
}

func searchFold(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) {
			return -1
		}
		return r
	}, strings.ToLower(norm.NFKC.String(value)))
}

func normalizeSearchDictionary(entries []searchDictionaryEntry) ([]searchDictionaryEntry, error) {
	if len(entries) > 300 {
		return nil, errors.New("조직 용어는 최대 300개입니다")
	}
	if entries == nil {
		entries = []searchDictionaryEntry{}
	}
	seen, ids := map[string]string{}, map[string]bool{}
	for i := range entries {
		e := &entries[i]
		e.Canonical = strings.TrimSpace(norm.NFKC.String(e.Canonical))
		if e.ID == "" {
			e.ID = newID()
		}
		if !validID(e.ID) || ids[e.ID] {
			return nil, errors.New("용어 식별자가 올바르지 않거나 중복되었습니다")
		}
		ids[e.ID] = true
		if e.Aliases == nil {
			e.Aliases = []string{}
		}
		if len(e.Aliases) > 12 {
			return nil, errors.New("용어별 별칭은 최대 12개입니다")
		}
		all := append([]string{e.Canonical}, e.Aliases...)
		for j, value := range all {
			value = strings.TrimSpace(norm.NFKC.String(value))
			fold := searchFold(value)
			if !utf8.ValidString(value) || len(value) == 0 || len(value) > 200 || fold == "" || strings.ContainsAny(value, "\x00\n\r") {
				return nil, errors.New("용어와 별칭은 한 줄의 1~200바이트로 입력하세요")
			}
			if old, ok := seen[fold]; ok && old != e.ID {
				return nil, errors.New("공백·대소문자를 정규화했을 때 같은 용어가 여러 묶음에 있습니다")
			}
			seen[fold] = e.ID
			if j > 0 {
				e.Aliases[j-1] = value
			}
		}
	}
	return entries, nil
}

func searchDictionary(ctx context.Context, q collaborationQuery, wid string) (int, []searchDictionaryEntry, error) {
	var revision int
	var raw []byte
	e := q.QueryRow(ctx, `SELECT revision,entries FROM workspace_search_dictionary WHERE workspace_id=$1`, wid).Scan(&revision, &raw)
	if errors.Is(e, pgx.ErrNoRows) {
		return 0, []searchDictionaryEntry{}, nil
	}
	if e != nil {
		return 0, nil, e
	}
	var entries []searchDictionaryEntry
	if e = json.Unmarshal(raw, &entries); e != nil {
		return 0, nil, e
	}
	return revision, entries, nil
}

// A deterministic lexical interpretation, not a Korean morphological analyzer.
// Dictionary matching tolerates spaces; an ASCII alias does not match inside
// another ASCII word ("ai" must not turn "mail" into artificial intelligence).
func interpretSearch(term string, revision int, entries []searchDictionaryEntry) searchInterpretation {
	result := searchInterpretation{Normalized: strings.ToLower(norm.NFKC.String(strings.TrimSpace(term))), DictionaryRevision: revision, Parts: [][]string{}, FoldedParts: [][]string{}, MatchedTerms: []string{}, Warnings: []string{}, Strategy: "nfkc_spacing_dictionary_bigrams"}
	if term == "" {
		return result
	}
	advanced := strings.ContainsAny(term, "\"")
	for _, word := range strings.Fields(term) {
		advanced = advanced || strings.EqualFold(word, "OR") || strings.HasPrefix(word, "-")
	}
	if advanced {
		result.Strategy = "websearch_syntax"
		result.Warnings = append(result.Warnings, "따옴표·OR·제외 문법은 기존 검색 문법으로 처리하며 공백·사전 확장을 적용하지 않습니다.")
		return result
	}
	type alias struct {
		fold  string
		entry searchDictionaryEntry
	}
	aliases := []alias{}
	for _, entry := range entries {
		for _, a := range append([]string{entry.Canonical}, entry.Aliases...) {
			aliases = append(aliases, alias{searchFold(a), entry})
		}
	}
	sort.SliceStable(aliases, func(i, j int) bool { return len(aliases[i].fold) > len(aliases[j].fold) })
	var folded strings.Builder
	starts, ends := []int{}, []int{}
	for pos, r := range result.Normalized {
		if unicode.IsSpace(r) {
			continue
		}
		folded.WriteRune(r)
		for range utf8.RuneLen(r) {
			starts = append(starts, pos)
			ends = append(ends, pos+utf8.RuneLen(r))
		}
	}
	compact := folded.String()
	lastOriginal := 0
	matched := map[string]bool{}
	addPlain := func(value string) {
		for _, word := range strings.Fields(value) {
			result.Parts = append(result.Parts, []string{word})
		}
	}
	for pos := 0; pos < len(compact); {
		var found *alias
		for i := range aliases {
			a := &aliases[i]
			if a.fold == "" || !strings.HasPrefix(compact[pos:], a.fold) {
				continue
			}
			end := pos + len(a.fold)
			first, _ := utf8.DecodeRuneInString(a.fold)
			last, _ := utf8.DecodeLastRuneInString(a.fold)
			asciiWord := func(r rune) bool { return r < 128 && (unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_') }
			if pos > 0 && asciiWord(first) {
				previous, _ := utf8.DecodeLastRuneInString(compact[:pos])
				if asciiWord(previous) && starts[pos] == ends[pos-1] {
					continue
				}
			}
			if end < len(compact) && asciiWord(last) {
				next, _ := utf8.DecodeRuneInString(compact[end:])
				if asciiWord(next) && starts[end] == ends[end-1] {
					continue
				}
			}
			found = a
			break
		}
		if found == nil {
			_, size := utf8.DecodeRuneInString(compact[pos:])
			pos += size
			continue
		}
		addPlain(result.Normalized[lastOriginal:starts[pos]])
		part := []string{}
		seen := map[string]bool{}
		for _, a := range append([]string{found.entry.Canonical}, found.entry.Aliases...) {
			fold := searchFold(a)
			if !seen[fold] {
				seen[fold] = true
				part = append(part, a)
			}
		}
		result.Parts = append(result.Parts, part)
		if !matched[found.entry.ID] {
			result.MatchedTerms = append(result.MatchedTerms, found.entry.Canonical)
			matched[found.entry.ID] = true
		}
		pos += len(found.fold)
		lastOriginal = ends[pos-1]
	}
	addPlain(result.Normalized[lastOriginal:])
	if len(result.Parts) > 16 {
		result.Parts = [][]string{}
		result.Strategy = "websearch_syntax"
		result.Warnings = append(result.Warnings, "16개를 초과하는 검색 부분은 누락하지 않고 기존 검색 문법으로 처리합니다. 공백·사전 확장은 적용하지 않습니다.")
		return result
	}
	groups := []string{}
	for _, part := range result.Parts {
		foldedPart := []string{}
		alternatives := []string{}
		canFilter := true
		for _, value := range part {
			fold := searchFold(value)
			foldedPart = append(foldedPart, fold)
			grams, _ := searchGrams(fold)
			if len(grams) == 0 {
				canFilter = false
				continue
			}
			quoted := make([]string, len(grams))
			for i, g := range grams {
				quoted[i] = "'" + g + "'"
			}
			alternatives = append(alternatives, "("+strings.Join(quoted, " & ")+")")
		}
		result.FoldedParts = append(result.FoldedParts, foldedPart)
		if canFilter && len(alternatives) > 0 {
			groups = append(groups, "("+strings.Join(alternatives, " | ")+")")
		}
	}
	result.GramQuery = strings.Join(groups, " & ")
	if len(result.GramQuery) > 65536 {
		result.GramQuery = ""
		result.Warnings = append(result.Warnings, "확장된 용어가 많아 bigram 사전 필터 없이 현재 범위의 정확한 정규화 문자열을 확인합니다.")
	}
	return result
}

// Hash-free, hex-encoded Unicode bigrams are safe tsquery lexemes and cannot
// lose error-code punctuation. A capped projection skips the GIN prefilter;
// the final exact folded predicate still prevents false-positive results.
func searchGrams(folded string) ([]string, bool) {
	const limit = 32768
	set := map[string]bool{}
	runes := []rune(folded)
	for i := 0; i+1 < len(runes); i++ {
		key := "x" + hex.EncodeToString([]byte(string(runes[i:i+2])))
		set[key] = true
		if len(set) > limit {
			return []string{}, false
		}
	}
	keys := make([]string, 0, len(set))
	for key := range set {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys, true
}

func (s *Server) writeFoldedProjectionTx(ctx context.Context, tx pgx.Tx, id string, version int, markdown string, fragments []searchFragment) error {
	var title string
	var metadata []byte
	if e := tx.QueryRow(ctx, `SELECT title,jsonb_build_array(`+documentSummaryTags+`,`+documentSummaryAliases+`) FROM documents d WHERE id=$1`, id).Scan(&title, &metadata); e != nil {
		return e
	}
	var groups [][]string
	if e := json.Unmarshal(metadata, &groups); e != nil {
		return e
	}
	fields := []string{title, markdown}
	for _, group := range groups {
		fields = append(fields, group...)
	}
	folded := searchFold(strings.Join(fields, "\n"))
	grams, complete := searchGrams(folded)
	if _, e := tx.Exec(ctx, `INSERT INTO search_folded_documents(document_id,document_version,normalized,gram_vector,grams_complete) VALUES($1,$2,$3,to_tsvector('simple',$4),$5) ON CONFLICT(document_id) DO UPDATE SET document_version=excluded.document_version,normalized=excluded.normalized,gram_vector=excluded.gram_vector,grams_complete=excluded.grams_complete,indexed_at=now()`, id, version, folded, strings.Join(grams, " "), complete); e != nil {
		return e
	}
	batch := &pgx.Batch{}
	for i, f := range fragments {
		fold := searchFold(f.Content)
		grams, complete := searchGrams(fold)
		batch.Queue(`INSERT INTO search_folded_fragments(document_id,ordinal,document_version,normalized,gram_vector,grams_complete) VALUES($1,$2,$3,$4,to_tsvector('simple',$5),$6)`, id, i, version, fold, strings.Join(grams, " "), complete)
	}
	if len(fragments) > 0 {
		if e := tx.SendBatch(ctx, batch).Close(); e != nil {
			return fmt.Errorf("검색 조각 정규화: %w", e)
		}
	}
	return nil
}
