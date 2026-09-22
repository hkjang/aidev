package server

import (
	"context"
	"strings"
	"testing"
	"time"

	git "github.com/go-git/go-git/v5"
	"github.com/go-git/go-git/v5/plumbing/transport"
	gitServer "github.com/go-git/go-git/v5/plumbing/transport/server"
)

func gitSyncBareFactory(t *testing.T) (gitSyncRemoteFactory, string) {
	t.Helper()
	dir := t.TempDir()
	if _, e := git.PlainInit(dir, true); e != nil {
		t.Fatal(e)
	}
	endpoint, e := transport.NewEndpoint(dir)
	if e != nil {
		t.Fatal(e)
	}
	return func(context.Context, map[string]any, []string, bool) (*gitSyncRemote, error) {
		return &gitSyncRemote{Endpoint: endpoint, Transport: gitServer.NewClient(gitServer.DefaultLoader), Close: func() {}}, nil
	}, dir
}
func gitSyncAdvanceFixture(t *testing.T, factory gitSyncRemoteFactory, files []gitSyncFile) {
	t.Helper()
	remote, _ := factory(t.Context(), nil, nil, false)
	repo, e := gitSyncFetch(t.Context(), remote, "main")
	if e != nil {
		t.Fatal(e)
	}
	for i := range files {
		files[i].Hash = gitSyncContentHash(files[i].Data)
	}
	candidate, e := gitSyncCandidate(t.Context(), repo, files, newID(), time.Now())
	if e != nil {
		t.Fatal(e)
	}
	if uncertain, e := gitSyncPush(t.Context(), remote, "main", repo, candidate, nil); e != nil || uncertain {
		t.Fatal(uncertain, e)
	}
}

func TestPostgresGitSyncPreviewConfirmCASAndPrivatePull(t *testing.T) {
	s, client, ctx, p, wid := jobTestFixture(t)
	if _, e := s.DB.Exec(ctx, "UPDATE settings SET data=data||$1::jsonb WHERE id=1", jsonValue(map[string]any{"storage_path": t.TempDir()})); e != nil {
		t.Fatal(e)
	}
	if e := s.migrateGitSync(ctx); e != nil {
		t.Fatal(e)
	}
	registered := false
	for _, route := range s.apiRoutes {
		if route == "POST /api/v1/git-sync/connections/{id}/preview" {
			registered = true
		}
	}
	if !registered {
		s.registerGitSync()
	}
	factory, _ := gitSyncBareFactory(t)
	handler := func(ctx context.Context, j Job) (map[string]any, error) { return s.runGitSyncJob(ctx, j, factory) }
	s.RegisterJobHandler("git-sync.preview", handler)
	s.RegisterJobHandler("git-sync.execute", handler)
	policy := testJSONObject(t, client.request("GET", "/api/v1/admin/git-sync/settings", nil, 200))
	client.request("PUT", "/api/v1/admin/git-sync/settings", map[string]any{"revision": policy["revision"], "settings": map[string]any{"enabled": true, "allowed_hosts": []string{"git.example.invalid"}}}, 200)
	connection := func(prefix string) string {
		return str(testJSONObject(t, client.request("POST", "/api/v1/git-sync/connections", map[string]any{"workspace_id": wid, "owner_id": p.ID, "name": "Git 실제 bare 작업 검증", "enabled": true, "config": map[string]any{"url": "https://git.example.invalid/fixture.git", "branch": "main", "prefix": prefix}}, 200)), "id")
	}
	pushID := connection("madi")
	doc := testJSONObject(t, client.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "선택 원본", "markdown": "# 원본\n비공개 fixture 본문", "visibility": "private"}, 200))
	docID := str(doc, "id")
	client.request("POST", "/api/v1/documents", map[string]any{"workspace_id": wid, "title": "전송 제외", "markdown": "NEVER_EXPORT_UNSELECTED", "visibility": "private"}, 200)
	preview := func(id, direction string, ids, paths []string) map[string]any {
		t.Helper()
		queued := testJSONObject(t, client.request("POST", "/api/v1/git-sync/connections/"+id+"/preview", map[string]any{"direction": direction, "document_ids": ids, "paths": paths, "read_consent": true}, 202))
		drainJobs(t, s)
		run := testJSONObject(t, client.request("GET", "/api/v1/git-sync/runs/"+str(queued, "id"), nil, 200))
		if str(run, "status") != "preview" {
			t.Fatalf("preview failed: %s", jsonValue(run))
		}
		return run
	}
	confirm := func(run map[string]any, want int) {
		t.Helper()
		client.request("POST", "/api/v1/git-sync/runs/"+str(run, "id")+"/confirm", map[string]any{"revision": run["revision"], "config_fingerprint": run["config_fingerprint"], "remote_commit": run["remote_commit"], "confirm_external_visibility": true}, want)
	}
	run := preview(pushID, "push", []string{docID}, nil)
	var cipher []byte
	if e := s.DB.QueryRow(ctx, "SELECT snapshot_cipher FROM git_sync_runs WHERE id=$1", run["id"]).Scan(&cipher); e != nil || !strings.HasPrefix(string(cipher), "enc:") || strings.Contains(string(cipher), "fixture 본문") {
		t.Fatal("preview not encrypted", e)
	}
	confirm(run, 202)
	client.request("PUT", "/api/v1/documents/"+docID, map[string]any{"version": 1, "markdown": "# 수정 후 원본"}, 200)
	drainJobs(t, s)
	stale := testJSONObject(t, client.request("GET", "/api/v1/git-sync/runs/"+str(run, "id"), nil, 200))
	if str(stale, "status") != "failed" {
		t.Fatal("stale source executed", stale)
	}
	run = preview(pushID, "push", []string{docID}, nil)
	confirm(run, 202)
	drainJobs(t, s)
	done := testJSONObject(t, client.request("GET", "/api/v1/git-sync/runs/"+str(run, "id"), nil, 200))
	if str(done, "status") != "succeeded" {
		t.Fatalf("push failed: %s", jsonValue(done))
	}
	// Simulate a process losing the local completion acknowledgement after the
	// remote accepted its exact candidate. Recovery is a read-only comparison.
	if _, e := s.DB.Exec(ctx, "UPDATE git_sync_runs SET status='running' WHERE id=$1", done["id"]); e != nil {
		t.Fatal(e)
	}
	recoveredJob, e := scanJob(s.DB.QueryRow(ctx, "SELECT "+jobSelect+" FROM automation_jobs WHERE id=$1", done["job_id"]))
	if e != nil {
		t.Fatal(e)
	}
	if result, e := s.runGitSyncJob(ctx, recoveredJob, factory); e != nil || str(result, "status") != "succeeded" {
		t.Fatal("same candidate recovery failed", result, e)
	}
	remote, _ := factory(ctx, nil, nil, false)
	repo, e := gitSyncFetch(ctx, remote, "main")
	if e != nil {
		t.Fatal(e)
	}
	files, e := gitSyncRemoteFiles(ctx, repo, "madi", 1<<20)
	if e != nil || len(files) != 1 {
		t.Fatal("wrong export selection", len(files), e)
	}
	var exported gitSyncFile
	for _, file := range files {
		exported = file
		if strings.Contains(string(file.Data), "NEVER_EXPORT_UNSELECTED") || !strings.Contains(string(file.Data), "madi_source_version: 2") {
			t.Fatal("wrong exported source", string(file.Data))
		}
	}
	gitSyncAdvanceFixture(t, factory, []gitSyncFile{{Path: exported.Path, Data: []byte("# 원격 경쟁 수정")}})
	run = preview(pushID, "push", []string{docID}, nil)
	if !boolean(run["report"].(map[string]any), "has_conflicts") {
		t.Fatal("remote conflict not reported")
	}
	confirm(run, 409)
	pullID := connection("incoming")
	gitSyncAdvanceFixture(t, factory, []gitSyncFile{{Path: "incoming/외부.md", Data: []byte("---\ntitle: 외부 초안\nvisibility: workspace\nstatus: published\nid: " + newID() + "\n---\n# 외부 원문\n![그림](picture.png)\n")}, {Path: "incoming/picture.png", Data: []byte("not-a-real-image-fixture")}})
	browse := preview(pullID, "pull", nil, nil)
	if !boolean(browse["report"].(map[string]any), "selection_required") {
		t.Fatal("pull must require explicit paths")
	}
	confirm(browse, 409)
	run = preview(pullID, "pull", nil, []string{"incoming/외부.md", "incoming/picture.png"})
	confirm(run, 202)
	drainJobs(t, s)
	done = testJSONObject(t, client.request("GET", "/api/v1/git-sync/runs/"+str(run, "id"), nil, 200))
	if str(done, "status") != "succeeded" {
		t.Fatalf("pull failed: %s", jsonValue(done))
	}
	var md, visibility, status, owner string
	var version int
	e = s.DB.QueryRow(ctx, "SELECT markdown,visibility,status,owner_id::text,version FROM documents WHERE title='외부 초안'").Scan(&md, &visibility, &status, &owner, &version)
	if e != nil || visibility != "private" || status != "draft" || owner != p.ID || !strings.Contains(md, "/api/v1/attachments/") || strings.Contains(md, "status: published") {
		t.Fatal("private canonical import failed", e, visibility, status, owner, md)
	}
	run = preview(pullID, "pull", nil, []string{"incoming/외부.md", "incoming/picture.png"})
	if boolean(run["report"].(map[string]any), "has_conflicts") {
		t.Fatalf("unchanged pull unexpectedly conflicts: %s", jsonValue(run))
	}
	confirm(run, 202)
	drainJobs(t, s)
	var afterVersion int
	if e := s.DB.QueryRow(ctx, "SELECT version FROM documents WHERE title='외부 초안'").Scan(&afterVersion); e != nil || afterVersion != version {
		t.Fatal("unchanged pull produced an extra mutation", version, afterVersion, e)
	}
	var receivedID string
	if e := s.DB.QueryRow(ctx, "SELECT id::text FROM documents WHERE title='외부 초안'").Scan(&receivedID); e != nil {
		t.Fatal(e)
	}
	run = preview(pullID, "push", []string{receivedID}, nil)
	filesReport := run["report"].(map[string]any)["files"].([]any)
	stablePaths := map[string]bool{}
	for _, raw := range filesReport {
		stablePaths[str(raw.(map[string]any), "path")] = true
	}
	if len(stablePaths) != 2 || !stablePaths["incoming/외부.md"] || !stablePaths["incoming/picture.png"] {
		t.Fatal("roundtrip renamed stable remote paths", stablePaths)
	}
}
