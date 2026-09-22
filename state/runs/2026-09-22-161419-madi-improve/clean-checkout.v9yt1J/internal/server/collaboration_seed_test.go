package server

import (
	"strings"
	"testing"
)

func TestCollaborationSeedEquivalenceDoesNotHideContent(t *testing.T) {
	table := `<table><tr><th colspan="2" colwidth="120,180"><p>병합 제목</p></th></tr><tr><td><p style="text-align:center">가운데</p></td><td><p>오른쪽</p></td></tr></table>`
	propagated := strings.ReplaceAll(strings.ReplaceAll(table, `<td><p style=`, `<td colwidth="120"><p style=`), `<td><p>오른쪽`, `<td colwidth="180"><p>오른쪽`)
	if !collaborationSeedEquivalent(table, propagated) {
		t.Fatal("equivalent propagated table widths rejected")
	}
	if collaborationSeedEquivalent(table, strings.Replace(propagated, `colwidth="180"`, `colwidth="999"`, 1)) {
		t.Fatal("different table width accepted")
	}
	for _, pair := range [][2]string{{"original\n\n", "original"}, {"*strong*\n", "_strong_"}, {"[link][ref]\n\n[ref]: https://example.test", "[link](https://example.test)"}} {
		if !collaborationSeedEquivalent(pair[0], pair[1]) {
			t.Fatal("equivalent source rejected", pair)
		}
	}
	for _, pair := range [][2]string{{"original", "changed"}, {"    code", "code"}, {"<!-- private -->", "<!-- different -->"}, {"<aside>secret</aside>", "<aside>other</aside>"}, {"- [ ] task", "- [x] task"}} {
		if collaborationSeedEquivalent(pair[0], pair[1]) {
			t.Fatal("different source hidden", pair)
		}
	}
}

func TestPostgresCollaborationSeedKeepsExactSourceAndVersion(t *testing.T) {
	s, admin, _, wid, _ := collaborationTestSetup(t)
	ctx := t.Context()
	session := collaborationTestSession(t, admin)
	original := "---\r\naliases: [원문]\r\n---\r\noriginal\n\n"
	d := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "변경 없는 초기화", "markdown": original}, 200))
	id := str(d, "id")
	initial, e := s.collaborationState(ctx, id, session, nil)
	if e != nil {
		t.Fatal(e)
	}
	seed := collaborationTestDoc("original")
	defer seed.Destroy()
	state, e := s.collaborationState(ctx, id, session, &collaborationMessage{Type: "seed", Schema: collaborationSchemaID, Epoch: initial.Epoch, Version: 1, State: seed.EncodeStateAsUpdate()})
	if e != nil || state.Markdown != original || state.Version != 1 {
		t.Fatal("seed rewrote original", state, e)
	}
	repeated, e := s.collaborationState(ctx, id, session, &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: initial.Epoch, State: seed.EncodeStateAsUpdate()})
	if e != nil || repeated.Markdown != original || repeated.Version != 1 {
		t.Fatal("no-op rewrote original", repeated, e)
	}
	reread, e := s.collaborationState(ctx, id, session, nil)
	if e != nil || reread.Epoch != state.Epoch || reread.Markdown != original {
		t.Fatal("canonical/source mismatch reset state", reread, e)
	}
	collaborationTestInsert(seed, " actual change")
	changed, e := s.collaborationState(ctx, id, session, &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: initial.Epoch, State: seed.EncodeStateAsUpdate()})
	if e != nil || changed.Version != 2 || !strings.Contains(changed.Markdown, "actual change") || !strings.HasPrefix(changed.Markdown, "---\r\naliases: [원문]\r\n---\r\n") {
		t.Fatal("actual edit failed", changed, e)
	}
}
