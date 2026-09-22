package server

import (
	"context"
	"errors"
	"testing"
)

func TestPostgresCollaborationStrictProtectionBlocksHiddenBinaryAndReadback(t *testing.T) {
	s, admin, _, wid, _ := collaborationTestSetup(t)
	doc := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "보호 공동 편집", "markdown": "정상 원문"}, 200))
	id := str(doc, "id")
	ctx := context.Background()
	session := collaborationTestSession(t, admin)
	initial, e := s.collaborationState(ctx, id, session, nil)
	if e != nil {
		t.Fatal(e)
	}
	seed := collaborationTestDoc("정상 원문")
	defer seed.Destroy()
	state, e := s.collaborationState(ctx, id, session, &collaborationMessage{Type: "seed", Schema: collaborationSchemaID, Epoch: initial.Epoch, Version: initial.Version, State: seed.EncodeStateAsUpdate()})
	if e != nil {
		t.Fatal(e)
	}
	quiet, _, e := collaborationTestDial(t, admin, id, admin.base)
	if e != nil {
		t.Fatal(e)
	}
	_ = collaborationTestRead(t, quiet, "hello")
	for _, mode := range []string{"block", "mask"} {
		if _, e = s.DB.Exec(ctx, `UPDATE protection_settings SET data=data||jsonb_build_object('enabled',true,'mode',$1::text),revision=revision+1 WHERE id=1`, mode); e != nil {
			t.Fatal(e)
		}
		// Even perfectly innocent visible Markdown cannot prove the absence of
		// PII in tombstones or an unprojected Yjs root, so all binary is denied.
		for _, in := range []*collaborationMessage{nil, {Type: "update", Schema: collaborationSchemaID, Epoch: state.Epoch, State: []byte{1, 2, 3}}} {
			out, err := s.collaborationState(ctx, id, session, in)
			var fault *collaborationFault
			if !errors.As(err, &fault) || fault.code != "protection_policy" || len(out.State) != 0 {
				t.Fatalf("strict mode %s returned binary or accepted update: %#v %v", mode, out, err)
			}
		}
		if mode == "block" {
			notice := collaborationTestRead(t, quiet, "error")
			if notice.Code != "protection_policy" || len(notice.State) != 0 {
				t.Fatal("quiet recipient did not receive safe policy notice")
			}
		}
		blocked, _, err := collaborationTestDial(t, admin, id, admin.base)
		if err != nil {
			t.Fatal(err)
		}
		notice := collaborationTestRead(t, blocked, "error")
		if notice.Code != "protection_policy" || len(notice.State) != 0 {
			t.Fatal("new connection leaked binary instead of source-mode notice")
		}
		var sequence int64
		var markdown string
		if e = s.DB.QueryRow(ctx, `SELECT c.sequence,d.markdown FROM document_collaboration c JOIN documents d ON d.id=c.document_id WHERE d.id=$1`, id).Scan(&sequence, &markdown); e != nil {
			t.Fatal(e)
		}
		if sequence != state.Sequence || markdown != "정상 원문" {
			t.Fatal("strict rejection committed mutation")
		}
	}
	if _, e = s.DB.Exec(ctx, `UPDATE protection_settings SET data=data||'{"enabled":false}'::jsonb,revision=revision+1 WHERE id=1`); e != nil {
		t.Fatal(e)
	}
	if _, e = s.collaborationState(ctx, id, session, nil); e != nil {
		t.Fatal("explicitly disabled protection should retain normal collaboration", e)
	}
}
