package server

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"sort"
	"strings"
	"unicode/utf8"
)

type migrationLinkIndex struct {
	files     map[string][]byte
	items     map[string]migrationSessionItem
	sources   map[string]migrationSessionItem
	basenames map[string]string
	parents   map[string]string
}

func newMigrationLinkIndex(items []migrationSessionItem) migrationLinkIndex {
	x := migrationLinkIndex{files: map[string][]byte{}, items: map[string]migrationSessionItem{}, sources: map[string]migrationSessionItem{}, basenames: map[string]string{}, parents: map[string]string{}}
	for _, i := range items {
		x.sources[i.SourceID] = i
		if i.Kind == "folder" {
			continue
		}
		x.files[i.FilePath] = nil
		x.items[i.FilePath] = i
		for _, key := range []string{path.Base(i.FilePath), strings.TrimSuffix(path.Base(i.FilePath), path.Ext(i.FilePath))} {
			if old, exists := x.basenames[key]; !exists {
				x.basenames[key] = i.FilePath
			} else if old != i.FilePath {
				x.basenames[key] = ""
			}
		}
	}
	return x
}

func (x migrationLinkIndex) validateParents(items []migrationSessionItem) error {
	parentSources := map[string]string{}
	for _, item := range items {
		if !oneOf(item.Kind, "document", "folder") {
			continue
		}
		parent := item.ParentSourceID
		if parent == "" && !boolean(item.Metadata, "attachment_root") {
			dir := path.Dir(item.FilePath)
			if item.Kind == "folder" {
				dir = path.Dir(dir)
			}
			if dir != "." && dir != "" {
				parent = migrationFolderSource(dir)
				for _, ext := range []string{".md", ".markdown"} {
					if i, ok := x.items[dir+ext]; ok {
						parent = i.SourceID
						break
					}
				}
			}
		}
		parentSources[item.SourceID] = parent
		if parent != "" {
			p, ok := x.sources[parent]
			if !ok || !oneOf(p.Kind, "document", "folder") {
				return errors.New("상위 원본 문서를 찾을 수 없습니다")
			}
			x.parents[item.SourceID] = p.TargetID
		}
	}
	for id := range parentSources {
		seen := map[string]bool{}
		for cur, depth := id, 0; cur != ""; cur, depth = parentSources[cur], depth+1 {
			if seen[cur] || depth >= 20 {
				return errors.New("문서 트리가 순환하거나 20단계를 초과합니다")
			}
			seen[cur] = true
		}
	}
	return nil
}

func (x migrationLinkIndex) resolve(file, target string) (migrationSessionItem, bool) {
	if i, ok := x.sources[target]; ok {
		return i, true
	}
	u, e := url.Parse(target)
	if e != nil || u.IsAbs() || u.Host != "" || strings.ContainsAny(u.Path, "\\\x00") {
		return migrationSessionItem{}, false
	}
	for _, candidate := range []string{path.Clean(path.Join(path.Dir(file), u.Path)), strings.TrimPrefix(path.Clean(u.Path), "/")} {
		for _, ext := range []string{"", ".md", ".markdown", ".html", ".htm"} {
			if i, ok := x.items[candidate+ext]; ok && safeVaultPath(candidate+ext) {
				return i, true
			}
		}
	}
	i, ok := x.items[x.basenames[u.Path]]
	return i, ok
}

func migrationAttachmentID(assetID, documentID string) string {
	h := sha256.Sum256([]byte("madi/migration/attachment/v1/" + assetID + "/" + documentID))
	b := h[:16]
	b[6] = (b[6] & 15) | 80
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func migrationContentLinks(item migrationSessionItem, md string, index migrationLinkIndex) (string, []string, map[string]any) {
	assets := map[string]bool{}
	references := map[string]bool{}
	resolved, missing := 0, 0
	unresolved := []string{}
	noteMissing := func(v string) {
		missing++
		if len(unresolved) < 100 {
			unresolved = append(unresolved, v)
		}
	}
	resolve := func(target string) (string, bool) {
		u, e := url.Parse(target)
		if e != nil {
			return target, false
		}
		if u.IsAbs() || u.Host != "" || strings.HasPrefix(target, "#") {
			return target, false
		}
		lookup := u.Path
		for _, prefix := range []string{"/app/documents/", "/app/databases/", "/api/v1/attachments/"} {
			lookup = strings.TrimPrefix(lookup, prefix)
		}
		i, ok := index.resolve(item.FilePath, lookup)
		if !ok {
			noteMissing(target)
			return target, false
		}
		result := ""
		switch i.Kind {
		case "document", "folder":
			references[i.SourceID] = true
			result = "/app/documents/" + i.TargetID
		case "attachment":
			assets[i.SourceID] = true
			result = "/api/v1/attachments/" + migrationAttachmentID(i.TargetID, item.TargetID)
		case "csv":
			references[i.SourceID] = true
			result = "/app/databases/" + i.TargetID
		}
		if result == "" {
			noteMissing(target)
			return target, false
		}
		resolved++
		if u.Fragment != "" {
			result += "#" + u.EscapedFragment()
		}
		return result, true
	}
	out := rewriteVaultContent(md, func(part string) string {
		part = rewriteVaultMarkdownLinks(part, func(target string) string { updated, _ := resolve(target); return updated })
		part = vaultWikiPattern.ReplaceAllStringFunc(part, func(match string) string {
			pieces := vaultWikiPattern.FindStringSubmatch(match)
			target, fragment, label := splitVaultWiki(pieces[2])
			updated, ok := resolve(target + fragment)
			if !ok {
				return match
			}
			display := strings.TrimPrefix(label, "|")
			if display == "" {
				display = target
			}
			display = strings.NewReplacer("[", "\\[", "]", "\\]").Replace(display)
			return pieces[1] + "[" + display + "](" + updated + ")"
		})
		return part
	})
	attachments := []string{}
	for a := range assets {
		attachments = append(attachments, a)
	}
	sort.Strings(attachments)
	sources := []string{}
	for id := range references {
		sources = append(sources, id)
	}
	sort.Strings(sources)
	return out, attachments, map[string]any{"resolved": resolved, "unresolved": missing, "unresolved_examples": unresolved, "examples_truncated": missing > 100, "reference_sources": sources}
}

func (s *Server) prepareMigrationSessionItem(ctx context.Context, p *Principal, v migrationSession, item migrationSessionItem, index migrationLinkIndex, items []migrationSessionItem) error {
	raw, e := s.migrationItemRaw(ctx, item)
	if e != nil {
		return e
	}
	prepared := migrationPrepared{Title: strings.TrimSuffix(path.Base(item.FilePath), path.Ext(item.FilePath)), Tags: []string{}, Aliases: []string{}, Icon: "file"}
	compat := map[string]any{"status": "exact", "source_sha256": item.SourceHash, "source_bytes": len(raw), "warnings": []string{}}
	columns := []migrationCSVColumn{}
	switch item.Kind {
	case "attachment":
		prepared.Title = path.Base(item.FilePath)
		prepared.ContentType = http.DetectContentType(raw)
		compat["status"] = "binary_preserved"
		compat["warnings"] = []string{"첨부 원본 바이트를 보존합니다. 바이너리 내부 서식·민감정보의 완전한 해석을 보장하지 않습니다."}
	case "csv":
		data, err := parseResumeCSV(raw)
		if err != nil {
			return err
		}
		prepared.CSV = &data
		columns = inferMigrationCSV(data)
		compat["status"] = "type_review_required"
		compat["rows"] = len(data.Rows)
		compat["columns"] = len(data.Header)
	case "folder":
		prepared.Title = path.Base(path.Dir(item.FilePath))
		if boolean(item.Metadata, "attachment_root") {
			prepared.Title = "가져온 첨부파일"
		}
	case "document":
		if !utf8.Valid(raw) {
			return errors.New("문서 원본은 UTF-8이어야 합니다")
		}
		prepared.Markdown = string(raw)
		if oneOf(strings.ToLower(path.Ext(item.FilePath)), ".html", ".htm") {
			title, md, losses, err := migrationHTMLQuality(raw, item.FilePath, index.files)
			if err != nil {
				return err
			}
			prepared.Markdown = md
			if title != "" {
				prepared.Title = title
			}
			compat["status"] = "converted"
			compat["conversion_losses"] = losses
			compat["warnings"] = []string{"HTML은 안전한 Markdown으로 변환합니다. CSS·스크립트·폼·원격 이미지·병합 셀과 실행형 요소는 보존되지 않습니다. 원문 비교로 서식을 확인하세요."}
		} else if strings.EqualFold(path.Ext(item.FilePath), ".json") {
			var d struct {
				Format   string   `json:"format"`
				Version  int      `json:"version"`
				Title    string   `json:"title"`
				Markdown string   `json:"markdown"`
				Tags     []string `json:"tags"`
				Aliases  []string `json:"aliases"`
			}
			if json.Unmarshal(raw, &d) != nil || d.Format != "madi-migration-document" || d.Version != 1 {
				return errors.New("재개 JSON 항목은 madi-migration-document 버전 1 형식이어야 합니다")
			}
			prepared.Title = d.Title
			prepared.Markdown = d.Markdown
			prepared.Tags = d.Tags
			prepared.Aliases = d.Aliases
			compat["status"] = "converted"
		}
		fm, err := parseFrontMatter(prepared.Markdown)
		if err != nil {
			return err
		}
		if title := str(fm, "title"); strings.TrimSpace(title) != "" {
			prepared.Title = title
		}
		if tags, ok := fm["tags"]; ok {
			prepared.Tags = listStrings(tags)
		}
		if aliases, ok := fm["aliases"]; ok {
			prepared.Aliases = listStrings(aliases)
		}
		var links map[string]any
		prepared.Markdown, prepared.Attachments, links = migrationContentLinks(item, prepared.Markdown, index)
		compat["links"] = links
		if len(listStrings(links["reference_sources"])) > 10000 {
			return jobPermanent("한 문서의 문서·DB 참조가 10,000개를 초과합니다. 문서를 나누세요")
		}
		if prepared.Markdown != string(raw) && compat["status"] == "exact" {
			compat["status"] = "links_rewritten"
		}
	default:
		return errors.New("지원하지 않는 이관 항목입니다")
	}
	if oneOf(item.Kind, "document", "folder") {
		prepared.ParentID = index.parents[item.SourceID]
	}
	if cipher := str(item.Metadata, "source_metadata_cipher"); cipher != "" {
		raw, err := s.decrypt(cipher)
		if err != nil {
			return err
		}
		var meta map[string]any
		if err = json.Unmarshal([]byte(raw), &meta); err != nil {
			return err
		}
		if value, ok := meta["title"]; ok {
			title, ok := value.(string)
			if !ok {
				return errors.New("원본 제목은 문자열이어야 합니다")
			}
			prepared.Title = title
		}
		if value, ok := meta["tags"]; ok {
			prepared.Tags = listStrings(value)
		}
		if value, ok := meta["aliases"]; ok {
			prepared.Aliases = listStrings(value)
		}
		if value, ok := meta["icon"]; ok {
			icon, ok := value.(string)
			if !ok || len(icon) > 50 {
				return errors.New("원본 아이콘 형식을 확인하세요")
			}
			prepared.Icon = icon
		}
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var status string
	if e = tx.QueryRow(ctx, "SELECT status FROM migration_sessions WHERE id=$1 FOR UPDATE", v.ID).Scan(&status); e != nil {
		return e
	}
	if status != "preparing" {
		return jobPermanent("이관 준비가 취소되었습니다")
	}
	if !oneOf(item.Kind, "attachment", "csv") {
		protected, err := s.protectCanonicalDocumentTx(ctx, tx, p, item.TargetID, v.WorkspaceID, strings.TrimSpace(prepared.Title), prepared.Markdown, prepared.Tags, prepared.Aliases, map[string]any{})
		if err != nil {
			return err
		}
		prepared.Title = protected.Title
		prepared.Markdown = protected.Markdown
		prepared.Tags, prepared.Aliases = protected.Tags, protected.Aliases
		if protected.Changed {
			compat["status"] = "policy_masked"
		}
	}
	if item.Kind == "attachment" {
		protected, err := s.ProtectAttachmentTx(ctx, tx, p, "", v.WorkspaceID, prepared.Title, prepared.ContentType, raw)
		if err != nil {
			return err
		}
		prepared.Title = protected.Name
		prepared.AttachmentData = protected.Data
		compat["canonical_sha256"] = digest(string(protected.Data))
		compat["canonical_bytes"] = len(protected.Data)
		if protected.Changed {
			compat["status"] = "policy_masked"
		}
	}
	if item.Kind == "csv" {
		changed, err := s.protectMigrationCSVTx(ctx, tx, p, v.WorkspaceID, &prepared)
		if err != nil {
			return err
		}
		columns = inferMigrationCSV(*prepared.CSV)
		if changed {
			compat["status"] = "policy_masked"
		}
		compat["canonical_sha256"] = digest(string(jsonValue(prepared.CSV)))
		compat["canonical_bytes"] = len(jsonValue(prepared.CSV))
	}
	if len(prepared.Attachments) > 10000 {
		return jobPermanent("한 문서의 첨부 참조가 10,000개를 초과합니다. 문서를 나누어 준비하세요")
	}
	compat["attachment_sources"] = prepared.Attachments
	if prepared.Attachments == nil {
		compat["attachment_sources"] = []string{}
	}
	if oneOf(item.Kind, "document", "folder") && item.Disposition == "unchanged" {
		var same bool
		if e = tx.QueryRow(ctx, `SELECT title=$2 AND markdown=$3 AND tags=$4::jsonb AND aliases=$5::jsonb AND coalesce(parent_id::text,'')=$6 FROM documents WHERE id=$1`, item.TargetID, prepared.Title, prepared.Markdown, jsonValue(prepared.Tags), jsonValue(prepared.Aliases), prepared.ParentID).Scan(&same); e != nil {
			return e
		}
		if !same {
			item.Disposition = "changed"
		}
	}
	encoded, e := migrationMetadataCipher(s, prepared)
	if e != nil {
		return e
	}
	if item.Kind != "attachment" && item.Kind != "csv" {
		h := sha256.Sum256([]byte(prepared.Markdown))
		compat["canonical_sha256"] = hex.EncodeToString(h[:])
		compat["canonical_bytes"] = len(prepared.Markdown)
	}
	// Raw samples are returned only through the separately authorized item-detail
	// endpoint. The checkpoint index never stores source cell values in plaintext.
	for i := range columns {
		columns[i].Samples = nil
	}
	preparedResult, e := tx.Exec(ctx, `UPDATE migration_session_items SET prepared_data=$2,compatibility=$3,csv_types=$4,status='prepared',checkpoint=2,error='',updated_at=now(),disposition=$5 WHERE id=$1 AND status<>'prepared'`, item.ID, encoded, jsonValue(compat), jsonValue(columns), item.Disposition)
	if e == nil {
		// The session lock serializes checkpoint publication. Increment only for
		// the first durable transition, avoiding an O(item_count²) progress scan.
		_, e = tx.Exec(ctx, "UPDATE migration_sessions SET prepared_count=prepared_count+$2,updated_at=now() WHERE id=$1", v.ID, preparedResult.RowsAffected())
	}
	if e == nil {
		e = tx.Commit(ctx)
	}
	return e
}
