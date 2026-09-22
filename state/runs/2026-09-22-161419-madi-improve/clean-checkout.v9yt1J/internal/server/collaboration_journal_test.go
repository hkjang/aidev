package server

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/reearth/ygo/crdt"
)

func collaborationJournalFixture(t *testing.T) (*Server, *integrationTestClient, *integrationTestClient, string, string, *crdt.Doc, collaborationMessage) {
	t.Helper()
	s, admin, editor, wid, _ := collaborationTestSetup(t)
	d := testJSONObject(t, admin.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "장시간 공동 편집", "markdown": "base\n\n"}, 200))
	id := str(d, "id")
	session := collaborationTestSession(t, admin)
	hello, e := s.collaborationState(t.Context(), id, session, nil)
	if e != nil {
		t.Fatal(e)
	}
	doc := collaborationTestDoc("base")
	t.Cleanup(doc.Destroy)
	seed, e := s.collaborationState(t.Context(), id, session, &collaborationMessage{Type: "seed", Schema: collaborationSchemaID, Epoch: hello.Epoch, Version: hello.Version, State: doc.EncodeStateAsUpdate()})
	if e != nil {
		t.Fatal(e)
	}
	return s, admin, editor, id, session, doc, seed
}

func TestPostgresCollaborationJournalCheckpointReplayAndHistory(t *testing.T) {
	s, admin, _, id, session, doc, state := collaborationJournalFixture(t)
	ctx := t.Context()
	initialSnapshot := append([]byte(nil), state.State...)
	for i := 0; i < 10; i++ {
		vector := doc.StateVector()
		collaborationTestInsert(doc, " 새변경")
		var e error
		state, e = s.collaborationState(ctx, id, session, &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: state.Epoch, State: crdt.EncodeStateAsUpdateV1(doc, vector)})
		if e != nil {
			t.Fatal(e)
		}
	}
	var snapshotSeq, updates int64
	var snapshot []byte
	if e := s.DB.QueryRow(ctx, `SELECT snapshot_sequence,state,(SELECT count(*) FROM collaboration_updates WHERE document_id=$1) FROM document_collaboration WHERE document_id=$1`, id).Scan(&snapshotSeq, &snapshot, &updates); e != nil {
		t.Fatal(e)
	}
	if snapshotSeq != 1 || updates != 10 || string(snapshot) != string(initialSnapshot) {
		t.Fatal("full snapshot rewritten instead of journal", snapshotSeq, updates)
	}
	if e := s.migrateCollaboration(ctx); e != nil {
		t.Fatal(e)
	}
	fresh := &Server{DB: s.DB, EncryptionKey: s.EncryptionKey}
	replayed, e := fresh.collaborationState(ctx, id, session, nil)
	if e != nil || replayed.Markdown != state.Markdown || replayed.Version != 11 {
		t.Fatal("fresh replica replay failed", e, replayed.Version)
	}
	delta, e := fresh.collaborationStateSince(ctx, id, session, nil, state.Epoch, state.Sequence-1)
	if e != nil || delta.StateMode != "delta" || delta.FromSequence != state.Sequence-1 || len(delta.State) >= len(state.State) {
		t.Fatal("durable delta catchup failed", e, delta.StateMode, len(delta.State), len(state.State))
	}
	for i := 10; i < 64; i++ {
		vector := doc.StateVector()
		collaborationTestInsert(doc, " x")
		state, e = s.collaborationState(ctx, id, session, &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: state.Epoch, State: crdt.EncodeStateAsUpdateV1(doc, vector)})
		if e != nil {
			t.Fatal(e)
		}
	}
	if e = s.DB.QueryRow(ctx, `SELECT snapshot_sequence,(SELECT count(*) FROM collaboration_updates WHERE document_id=$1) FROM document_collaboration WHERE document_id=$1`, id).Scan(&snapshotSeq, &updates); e != nil {
		t.Fatal(e)
	}
	if snapshotSeq != 65 || updates != 0 || state.Version != 65 {
		t.Fatal("checkpoint/canonical CAS", snapshotSeq, updates, state.Version)
	}
	groups := admin.request("GET", "/api/v1/documents/"+id+"/collaboration/history", nil, 200)
	if !strings.Contains(string(groups), `"first_version":2`) || !strings.Contains(string(groups), `"last_version":65`) {
		t.Fatalf("history grouping missing: %s", groups)
	}
	var count int
	if e = s.DB.QueryRow(ctx, "SELECT count(*) FROM document_versions WHERE document_id=$1", id).Scan(&count); e != nil || count != 65 {
		t.Fatal("immutable per-version restore was lost", count, e)
	}
	admin.request("GET", "/api/v1/documents/"+id+"/collaboration/diagnostics", nil, 200)
}

func TestPostgresCollaborationJournalGapFailsClosed(t *testing.T) {
	s, _, _, id, session, doc, state := collaborationJournalFixture(t)
	for i := 0; i < 2; i++ {
		collaborationTestInsert(doc, " changed")
		var e error
		state, e = s.collaborationState(t.Context(), id, session, &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: state.Epoch, State: doc.EncodeStateAsUpdate()})
		if e != nil {
			t.Fatal(e)
		}
	}
	if _, e := s.DB.Exec(t.Context(), "DELETE FROM collaboration_updates WHERE document_id=$1 AND sequence=2", id); e != nil {
		t.Fatal(e)
	}
	if _, e := s.collaborationState(t.Context(), id, session, nil); e == nil || !strings.Contains(e.Error(), "불완전") {
		t.Fatal("journal gap accepted", e)
	}
	var version int
	if e := s.DB.QueryRow(t.Context(), "SELECT version FROM documents WHERE id=$1", id).Scan(&version); e != nil || version != 3 {
		t.Fatal("gap changed canonical document", version, e)
	}
}

func TestPostgresCollaborationCompactionOfflineEpochAndConfirmation(t *testing.T) {
	s, admin, editor, id, session, doc, state := collaborationJournalFixture(t)
	input := map[string]any{"expected_version": state.Version, "expected_epoch": state.Epoch, "confirm": true}
	editor.request("POST", "/api/v1/documents/"+id+"/collaboration/compact", input, 403)
	input["confirm"] = false
	admin.request("POST", "/api/v1/documents/"+id+"/collaboration/compact", input, 400)
	input["confirm"] = true
	input["expected_version"] = state.Version + 1
	admin.request("POST", "/api/v1/documents/"+id+"/collaboration/compact", input, 409)
	input["expected_version"] = state.Version
	result := testJSONObject(t, admin.request("POST", "/api/v1/documents/"+id+"/collaboration/compact", input, 200))
	if str(result, "epoch") == state.Epoch || number(result, "version", 0) != state.Version || !boolean(result, "markdown_unchanged") {
		t.Fatal("compaction changed original content")
	}
	collaborationTestInsert(doc, " old offline change")
	stale, e := s.collaborationState(t.Context(), id, session, &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: state.Epoch, State: doc.EncodeStateAsUpdate()})
	if e != nil || stale.Type != "reset" || stale.ResetReason != "manual_compaction" || stale.Markdown != "base\n\n" || stale.Version != state.Version {
		t.Fatal("offline editor overwrote compacted epoch", e, stale)
	}
	fresh := collaborationTestDoc("base")
	defer fresh.Destroy()
	seed, e := s.collaborationState(t.Context(), id, session, &collaborationMessage{Type: "seed", Schema: collaborationSchemaID, Epoch: stale.Epoch, Version: stale.Version, State: fresh.EncodeStateAsUpdate()})
	if e != nil || seed.Markdown != "base\n\n" || seed.Version != state.Version {
		t.Fatal("new epoch seed rewrote canonical", e)
	}
}

func TestPostgresCollaborationJournalCacheRollbackAndRevalidation(t *testing.T) {
	s, _, _, id, session, doc, state := collaborationJournalFixture(t)
	rt, _, _, unsubscribe := s.subscribeCollaboration(id)
	defer unsubscribe()
	ctx := t.Context()
	for i := 0; i < 3; i++ {
		collaborationTestInsert(doc, " cache")
		var e error
		state, e = s.collaborationState(ctx, id, session, &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: state.Epoch, State: doc.EncodeStateAsUpdate()})
		if e != nil {
			t.Fatal(e)
		}
	}
	if rt.cacheHits.Load() < 2 || rt.cacheBytes.Load() == 0 {
		t.Fatal("active room did not reuse validated native state")
	}
	before := state
	if _, e := s.DB.Exec(ctx, `CREATE FUNCTION reject_journal_commit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN RAISE EXCEPTION 'injected durable journal failure'; END $$; CREATE TRIGGER reject_journal_commit BEFORE INSERT ON collaboration_updates FOR EACH ROW EXECUTE FUNCTION reject_journal_commit()`); e != nil {
		t.Fatal(e)
	}
	collaborationTestInsert(doc, " rejected private mutation")
	if _, e := s.collaborationState(ctx, id, session, &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: state.Epoch, State: doc.EncodeStateAsUpdate()}); e == nil {
		t.Fatal("failed journal commit accepted")
	}
	if rt.cacheBytes.Load() != 0 {
		t.Fatal("uncommitted cache survived rollback")
	}
	if _, e := s.DB.Exec(ctx, "DROP TRIGGER reject_journal_commit ON collaboration_updates"); e != nil {
		t.Fatal(e)
	}
	actual, e := s.collaborationState(ctx, id, session, nil)
	if e != nil || actual.Sequence != before.Sequence || actual.Markdown != before.Markdown || actual.Version != before.Version {
		t.Fatal("failed cache transaction changed canonical state", e)
	}
	clean := collaborationTestClone(t, actual.State)
	collaborationTestInsert(clean, " accepted")
	actual, e = s.collaborationState(ctx, id, session, &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: actual.Epoch, State: clean.EncodeStateAsUpdate()})
	if e != nil || strings.Contains(actual.Markdown, "rejected") {
		t.Fatal("rollback cache poisoned later writer", e)
	}
	// A restore can reuse an epoch/sequence but not the content hash. Simulate a
	// different durable branch and make sure the old in-memory branch is discarded.
	branch := collaborationTestDoc("restored")
	defer branch.Destroy()
	binary := branch.EncodeStateAsUpdate()
	if _, e = s.DB.Exec(ctx, `UPDATE documents SET markdown='restored' WHERE id=$1;`, id); e != nil {
		t.Fatal(e)
	}
	if _, e = s.DB.Exec(ctx, `UPDATE document_collaboration SET state=$2,projected_markdown='restored',snapshot_sequence=sequence,head_hash=$3 WHERE document_id=$1`, id, binary, digest(string(binary))); e != nil {
		t.Fatal(e)
	}
	if _, e = s.DB.Exec(ctx, "DELETE FROM collaboration_updates WHERE document_id=$1", id); e != nil {
		t.Fatal(e)
	}
	collaborationTestInsert(branch, " new branch")
	actual, e = s.collaborationState(ctx, id, session, &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: actual.Epoch, State: branch.EncodeStateAsUpdate()})
	if e != nil || actual.Markdown != "restored new branch" {
		t.Fatal("restored branch reused stale cache", e, actual.Markdown)
	}
	unsubscribe()
	select {
	case <-rt.done:
	case <-time.After(8 * time.Second):
		t.Fatal("last subscriber leaked listener")
	}
	if rt.cacheBytes.Load() != 0 {
		t.Fatal("last subscriber leaked cached native docs")
	}
}

func collaborationWait(t *testing.T, description string, fn func() bool) {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if fn() {
			return
		}
		select {
		case <-t.Context().Done():
			t.Fatal(t.Context().Err())
		case <-time.After(15 * time.Millisecond):
		}
	}
	t.Fatal(description)
}

func TestPostgresCollaborationNotificationsReplicaCatchupAndCleanup(t *testing.T) {
	s, _, _, id, session, doc, state := collaborationJournalFixture(t)
	other := &Server{DB: s.DB, EncryptionKey: s.EncryptionKey}
	rt1, wake1, _, stop1 := s.subscribeCollaboration(id)
	defer stop1()
	rt2, wake2, _, stop2 := other.subscribeCollaboration(id)
	defer stop2()
	collaborationWait(t, "listeners did not connect", func() bool { return rt1.connected.Load() && rt2.connected.Load() })
	for {
		select {
		case <-wake1:
			continue
		default:
		}
		break
	}
	for {
		select {
		case <-wake2:
			continue
		default:
		}
		break
	}
	collaborationTestInsert(doc, " notification")
	updated, e := s.collaborationState(t.Context(), id, session, &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: state.Epoch, State: doc.EncodeStateAsUpdate()})
	if e != nil {
		t.Fatal(e)
	}
	for _, wake := range []<-chan struct{}{wake1, wake2} {
		select {
		case <-wake:
		case <-time.After(3 * time.Second):
			t.Fatal("committed document did not wake replica")
		}
	}
	catchup, e := other.collaborationStateSince(t.Context(), id, session, nil, state.Epoch, state.Sequence)
	if e != nil || catchup.Sequence != updated.Sequence || catchup.StateMode != "delta" {
		t.Fatal("second replica durable catchup failed", e)
	}
	pid := rt2.listenerPID.Load()
	if _, e = s.DB.Exec(t.Context(), "SELECT pg_terminate_backend($1)", pid); e != nil {
		t.Fatal(e)
	}
	collaborationWait(t, "lost listener did not reconnect", func() bool {
		return rt2.connected.Load() && rt2.listenerPID.Load() != 0 && rt2.listenerPID.Load() != pid
	})
	other.CloseCollaboration()
	select {
	case <-rt2.done:
	case <-time.After(8 * time.Second):
		t.Fatal("shutdown leaked listener")
	}
	if rt2.connected.Load() {
		t.Fatal("closed listener reported connected")
	}
}

func TestPostgresCollaborationAutomaticCompactionPreservesCanonical(t *testing.T) {
	s, _, _, id, session, doc, state := collaborationJournalFixture(t)
	oldEpoch := state.Epoch
	history := doc.GetMap("historical-root")
	for i := 0; i < 2; i++ {
		vector := doc.StateVector()
		// Represent retained CRDT history outside the visible XML tree. Strict
		// protection modes reject it before reaching this point (separate tests).
		doc.Transact(func(tx *crdt.Transaction) {
			history.Set(tx, string(rune('a'+i)), strings.Repeat("x", 7<<20))
		})
		var e error
		state, e = s.collaborationState(t.Context(), id, session, &collaborationMessage{Type: "update", ID: int64(i + 1), Schema: collaborationSchemaID, Epoch: oldEpoch, State: crdt.EncodeStateAsUpdateV1(doc, vector)})
		if e != nil {
			t.Fatal(e)
		}
	}
	if state.Type != "reset" || !state.Committed || state.Epoch == oldEpoch || state.ResetReason != "history_compacted" || state.Markdown != "base\n\n" || state.Version != 1 {
		t.Fatalf("automatic compaction lost canonical content/commit acknowledgement: %+v", state)
	}
	var snapshotBytes, tail, versions int
	if e := s.DB.QueryRow(t.Context(), `SELECT octet_length(state),(SELECT count(*) FROM collaboration_updates WHERE document_id=$1),(SELECT count(*) FROM document_versions WHERE document_id=$1) FROM document_collaboration WHERE document_id=$1`, id).Scan(&snapshotBytes, &tail, &versions); e != nil {
		t.Fatal(e)
	}
	if snapshotBytes != 0 || tail != 0 || versions != 1 {
		t.Fatal("old binary history survived compaction", snapshotBytes, tail, versions)
	}
}

func TestPostgresCollaborationWebSocketsAcrossReplicasAndShutdown(t *testing.T) {
	s, admin, _, id, _, doc, state := collaborationJournalFixture(t)
	other, e := New(t.Context(), s.DB, s.EncryptionKey, "test", "admin@example.test", "Integration-Test-Password-2026!", fstest.MapFS{"index.html": &fstest.MapFile{Data: []byte("madi")}})
	if e != nil {
		t.Fatal(e)
	}
	defer other.CloseCollaboration()
	server := httptest.NewServer(other)
	defer server.Close()
	peer := newIntegrationTestClient(t, server.URL)
	peer.request("POST", "/api/v1/auth/login", map[string]any{"email": "admin@example.test", "password": "Integration-Test-Password-2026!"}, 200)
	a, _, e := collaborationTestDial(t, admin, id, admin.base)
	if e != nil {
		t.Fatal(e)
	}
	b, _, e := collaborationTestDial(t, peer, id, peer.base)
	if e != nil {
		t.Fatal(e)
	}
	defer a.CloseNow()
	defer b.CloseNow()
	_ = collaborationTestRead(t, a, "hello")
	_ = collaborationTestRead(t, b, "hello")
	vector := doc.StateVector()
	collaborationTestInsert(doc, " across-replicas")
	ctx, cancel := context.WithTimeout(t.Context(), 8*time.Second)
	defer cancel()
	if e = wsjson.Write(ctx, a, collaborationMessage{Type: "update", ID: 1, Schema: collaborationSchemaID, Epoch: state.Epoch, State: crdt.EncodeStateAsUpdateV1(doc, vector)}); e != nil {
		t.Fatal(e)
	}
	ack := collaborationTestRead(t, a, "ack")
	for {
		got := collaborationTestRead(t, b, "sync")
		if got.Sequence < ack.Sequence {
			continue
		}
		if got.Markdown != ack.Markdown || got.StateMode != "delta" {
			t.Fatal("replica did not receive durable delta", got.StateMode)
		}
		break
	}
	other.CloseCollaboration()
	for {
		var m collaborationMessage
		e = wsjson.Read(ctx, b, &m)
		if e != nil {
			break
		}
	}
	if ctx.Err() != nil {
		t.Fatal("shutdown did not close active socket")
	}
	if websocket.CloseStatus(e) == websocket.StatusMessageTooBig {
		t.Fatal(e)
	}
	s.CloseCollaboration()
}

func TestPostgresCollaborationRestoreRebaseAndLegacyCheckpoint(t *testing.T) {
	s, _, _, id, session, doc, state := collaborationJournalFixture(t)
	ctx := t.Context()
	collaborationTestInsert(doc, " historical")
	state, e := s.collaborationState(ctx, id, session, &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: state.Epoch, State: doc.EncodeStateAsUpdate()})
	if e != nil {
		t.Fatal(e)
	}
	// A real legacy row stored the entire state at sequence, without a separate
	// checkpoint column. Simulate only this fixture's old schema, then migrate.
	if _, e = s.DB.Exec(ctx, `UPDATE document_collaboration SET state=$2 WHERE document_id=$1`, id, state.State); e != nil {
		t.Fatal(e)
	}
	if _, e = s.DB.Exec(ctx, `DELETE FROM collaboration_updates;ALTER TABLE document_collaboration DROP COLUMN snapshot_sequence`); e != nil {
		t.Fatal(e)
	}
	if e = s.migrateCollaboration(ctx); e != nil {
		t.Fatal(e)
	}
	legacy, e := s.collaborationState(ctx, id, session, nil)
	if e != nil || legacy.Sequence != state.Sequence || legacy.Markdown != state.Markdown {
		t.Fatal("legacy migration lost original head", e)
	}
	var checkpoint int64
	if e = s.DB.QueryRow(ctx, `SELECT snapshot_sequence FROM document_collaboration WHERE document_id=$1`, id).Scan(&checkpoint); e != nil || checkpoint != state.Sequence {
		t.Fatal("legacy state treated as incomplete tail", checkpoint, e)
	}
	tx, e := s.DB.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = invalidateCollaborationRestoreTx(ctx, tx); e != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(e)
	}
	if e = tx.Rollback(ctx); e != nil {
		t.Fatal(e)
	}
	rolled, e := s.collaborationState(ctx, id, session, nil)
	if e != nil || rolled.Epoch != state.Epoch || rolled.Markdown != state.Markdown {
		t.Fatal("rollback changed restoration branch", e)
	}
	tx, e = s.DB.Begin(ctx)
	if e != nil {
		t.Fatal(e)
	}
	if e = invalidateCollaborationRestoreTx(ctx, tx); e != nil {
		_ = tx.Rollback(ctx)
		t.Fatal(e)
	}
	if e = tx.Commit(ctx); e != nil {
		t.Fatal(e)
	}
	collaborationTestInsert(doc, " stale-post-backup")
	rejected, e := s.collaborationState(ctx, id, session, &collaborationMessage{Type: "update", Schema: collaborationSchemaID, Epoch: state.Epoch, State: doc.EncodeStateAsUpdate()})
	if e != nil || rejected.Type != "reset" || rejected.Epoch == state.Epoch || rejected.ResetReason != "backup_restored" || rejected.Markdown != state.Markdown || rejected.Version != state.Version {
		t.Fatal("restored branch accepted previous client", e)
	}
}
