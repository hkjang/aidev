package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/intelligence"
)

// What a band change would do, before it is saved.
//
// The two bands decide how much of the graph everyone sees. Their effect
// depends on the corpus, so the administrator screen offered two spin buttons
// whose only documentation was to save them and go and look — at which point
// every canvas in the installation had already changed.
func TestBandPreviewMeasuresTheProposalBeforeItIsSavedIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	admin := uuid.New()
	username := "bp_" + strings.ReplaceAll(admin.String(), "-", "")
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO users(id,username,display_name,role) VALUES($1,$2::citext,$2::text,'admin')`, admin, username); err != nil {
		t.Fatal(err)
	}
	space := uuid.New()
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'밴드 미리보기')`, space, admin); err != nil {
		t.Fatal(err)
	}

	// Two subjects that share no vocabulary, so the pair scores really do fall
	// into two populations rather than one flat one. Without that the bands have
	// nothing to separate and the preview could only ever report one answer.
	subjects := []string{
		"고양이 사료 급여 간격과 사료 보관 방법",
		"쿠버네티스 배포 롤백 절차와 헬름 차트 버전",
	}
	for i := 0; i < 24; i++ {
		note := uuid.New()
		content := fmt.Sprintf("%s %d번 메모", subjects[i%2], i)
		if _, err := db.Pool.Exec(ctx,
			`INSERT INTO notes(id,space_id,author_id,content,version) VALUES($1,$2,$3,$4,1)`,
			note, space, admin, content); err != nil {
			t.Fatal(err)
		}
		vector := intelligence.Embed(content)
		if _, err := db.Pool.Exec(ctx,
			`INSERT INTO note_embeddings(note_id,algorithm,dimensions,vector,content_version)
			 VALUES($1,'umm-local-chargram-v1',$2,$3,1)`,
			note, len(vector), vector); err != nil {
			t.Fatal(err)
		}
	}

	authService := &auth.Service{Store: db}
	server := &Server{Store: db, Auth: authService}
	router := chi.NewRouter()
	router.Get("/admin/intelligence/preview", server.bandPreview)
	handler := authService.Middleware(auth.Require(router))
	session, err := authService.CreateSession(ctx, admin, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	get := func(query string) (int, string) {
		request := httptest.NewRequest("GET", "/admin/intelligence/preview"+query, nil)
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response.Code, response.Body.String()
	}
	type outcome struct {
		RelatedBand    float64 `json:"relatedBand"`
		ClusterBand    float64 `json:"clusterBand"`
		WithoutRelated int     `json:"withoutRelated"`
		MedianRelated  int     `json:"medianRelated"`
		MostRelated    int     `json:"mostRelated"`
		Clusters       int     `json:"clusters"`
		Grouped        int     `json:"grouped"`
		LargestCluster int     `json:"largestCluster"`
		Ungrouped      int     `json:"ungrouped"`
	}
	var preview struct {
		Spaces   int     `json:"spaces"`
		Notes    int     `json:"notes"`
		Embedded int     `json:"embedded"`
		Semantic bool    `json:"semantic"`
		Current  outcome `json:"current"`
		Proposed outcome `json:"proposed"`
	}

	// A band far above the current one: fewer neighbours clear it, so more cards
	// show nothing. If that is not what comes back, the preview is not measuring
	// the number it was handed.
	status, body := get("?related_band=3.5&cluster_band=3.5")
	if status != 200 {
		t.Fatalf("preview: %d %s", status, body)
	}
	if err := json.Unmarshal([]byte(body), &preview); err != nil {
		t.Fatalf("preview: %v\n%s", err, body)
	}
	if preview.Spaces < 1 || preview.Notes < 24 {
		t.Fatalf("sampled %d spaces and %d notes, expected the space just written: %s", preview.Spaces, preview.Notes, body)
	}
	if preview.Embedded != preview.Notes {
		t.Fatalf("%d of %d notes embedded, all of them were written with a vector: %s", preview.Embedded, preview.Notes, body)
	}
	if preview.Proposed.RelatedBand != 3.5 || preview.Proposed.ClusterBand != 3.5 {
		t.Fatalf("the proposal came back as %v/%v: %s", preview.Proposed.RelatedBand, preview.Proposed.ClusterBand, body)
	}
	if preview.Current.RelatedBand == preview.Proposed.RelatedBand {
		t.Fatalf("current and proposed are the same band, so the comparison shows nothing: %s", body)
	}
	if preview.Proposed.WithoutRelated != preview.Notes {
		t.Fatalf("a band of 3.5 admitted neighbours for %d of %d cards: %s",
			preview.Notes-preview.Proposed.WithoutRelated, preview.Notes, body)
	}
	// The local char-gram backend is not judged fit to compare meaning, so the
	// canvas groups by position and the cluster band decides nothing. The
	// figures must be absent rather than zero-shaped counts of a grouping that
	// will not happen.
	if preview.Semantic {
		t.Fatalf("the test backend reported itself semantic, so this case no longer covers the fallback: %s", body)
	}
	for label, value := range map[string]outcome{"current": preview.Current, "proposed": preview.Proposed} {
		if value.Clusters != 0 || value.Grouped != 0 || value.LargestCluster != 0 || value.Ungrouped != 0 {
			t.Fatalf("%s reported cluster figures on a backend that will not cluster by meaning: %s", label, body)
		}
	}

	// The related figures are scored whatever the backend, so they stay.
	for label, value := range map[string]outcome{"current": preview.Current, "proposed": preview.Proposed} {
		if value.MedianRelated > value.MostRelated {
			t.Fatalf("%s: median %d above the highest %d: %s", label, value.MedianRelated, value.MostRelated, body)
		}
	}

	// The other end of the same claim: a band low enough to admit nearly
	// everything. Together the two ends prove the preview answers the band it
	// was handed rather than reporting the settings in force whatever it is
	// asked.
	status, body = get("?related_band=0.2&cluster_band=0.2")
	if status != 200 {
		t.Fatalf("low band: %d %s", status, body)
	}
	var low struct {
		Current  outcome `json:"current"`
		Proposed outcome `json:"proposed"`
	}
	if err := json.Unmarshal([]byte(body), &low); err != nil {
		t.Fatal(err)
	}
	if low.Proposed.WithoutRelated >= preview.Proposed.WithoutRelated {
		t.Fatalf("band 0.2 left %d cards empty, band 3.5 left %d: %s", low.Proposed.WithoutRelated, preview.Proposed.WithoutRelated, body)
	}
	// Whatever is proposed, the current column stays the settings in force.
	if low.Current != preview.Current {
		t.Fatalf("the current column moved with the proposal: %+v then %+v", preview.Current, low.Current)
	}

	// Nothing is written. The preview exists so that looking is not the same as
	// deciding.
	after := db.IntelligenceSettings(ctx)
	if after.RelatedBand != preview.Current.RelatedBand || after.ClusterBand != preview.Current.ClusterBand {
		t.Fatalf("the preview changed the settings it was previewing: %v", after)
	}

	// A band outside the range the settings accept is refused rather than
	// silently replaced — a preview of a substituted value describes a setting
	// nobody asked for.
	for _, bad := range []string{"?related_band=9", "?related_band=-1", "?cluster_band=abc", "?cluster_band=NaN"} {
		if status, body := get(bad); status != 400 {
			t.Fatalf("%s: %d %s", bad, status, body)
		}
	}
}

// The other side of the same panel: a backend that will cluster by meaning.
//
// v0.58.1 stopped the preview reporting cluster figures on a backend the canvas
// will not use them on, and pinned that absence — but an assertion that numbers
// are missing passes just as happily if they can never appear at all. Nothing
// covered the path where they should. The bars that decide it are settings, and
// the administrator guide already describes lowering them for a narrow corpus,
// so this is a configuration umm supports rather than a hole poked in the test.
func TestBandPreviewReportsClusterFiguresWhenTheCanvasWillUseThemIntegration(t *testing.T) {
	dsn := os.Getenv("POSTGRES_DSN")
	if dsn == "" {
		t.Skip("POSTGRES_DSN is not configured")
	}
	ctx := context.Background()
	db := isolatedHTTPStore(t, dsn)

	admin := uuid.New()
	username := "bs_" + strings.ReplaceAll(admin.String(), "-", "")
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO users(id,username,display_name,role) VALUES($1,$2::citext,$2::text,'admin')`, admin, username); err != nil {
		t.Fatal(err)
	}
	space := uuid.New()
	if _, err := db.Pool.Exec(ctx,
		`INSERT INTO spaces(id,owner_id,name) VALUES($1,$2,'묶음 미리보기')`, space, admin); err != nil {
		t.Fatal(err)
	}

	// Words drawn from two vocabularies rather than one fixed sentence repeated:
	// near-identical notes collapse the cluster band, which is measured in
	// internal/store and is not what this test is about.
	pools := [][]string{
		{"고양이", "사료", "급여", "간격", "보관", "습식", "건식", "체중"},
		{"쿠버네티스", "배포", "롤백", "헬름", "차트", "파드", "네임스페이스", "인그레스"},
	}
	rng := rand.New(rand.NewSource(19))
	for i := 0; i < 40; i++ {
		content := ""
		for w := 0; w < 12; w++ {
			pool := pools[i%len(pools)]
			content += pool[rng.Intn(len(pool))] + " "
		}
		note := uuid.New()
		if _, err := db.Pool.Exec(ctx,
			`INSERT INTO notes(id,space_id,author_id,content,version) VALUES($1,$2,$3,$4,1)`,
			note, space, admin, content); err != nil {
			t.Fatal(err)
		}
		vector := intelligence.Embed(content)
		if _, err := db.Pool.Exec(ctx,
			`INSERT INTO note_embeddings(note_id,algorithm,dimensions,vector,content_version)
			 VALUES($1,'umm-local-chargram-v1',$2,$3,1)`, note, len(vector), vector); err != nil {
			t.Fatal(err)
		}
	}

	// A backend that really does separate meaning from vocabulary. Lowering the
	// two quality bars would not do instead: Semantic also requires positive
	// discrimination, which is not a setting, and the shipped local algorithm
	// measures -0.25 — so no configuration makes the default installation
	// semantic. Only the vectors the quality probe is answered with can.
	gateway := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			Input []string `json:"input"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		type embedding struct {
			Embedding []float64 `json:"embedding"`
			Index     int       `json:"index"`
		}
		payload := struct {
			Data []embedding `json:"data"`
		}{}
		for index, text := range request.Input {
			payload.Data = append(payload.Data, embedding{Embedding: idealVector(text), Index: index})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(payload)
	}))
	defer gateway.Close()
	if err := db.PutSetting(ctx, "ai_gateway", map[string]any{
		"base_url":        gateway.URL,
		"embedding_model": "stub-semantic",
		"timeout_seconds": 10,
	}, admin); err != nil {
		t.Fatal(err)
	}
	db.InvalidateIntelligenceSettings()

	authService := &auth.Service{Store: db}
	server := &Server{Store: db, Auth: authService}
	router := chi.NewRouter()
	router.Get("/admin/intelligence/preview", server.bandPreview)
	handler := authService.Middleware(auth.Require(router))
	session, err := authService.CreateSession(ctx, admin, auth.SessionOrigin{UserAgent: "integration-test", ClientIP: "127.0.0.1"})
	if err != nil {
		t.Fatal(err)
	}
	get := func(query string) (int, string) {
		request := httptest.NewRequest("GET", "/admin/intelligence/preview"+query, nil)
		request.AddCookie(&http.Cookie{Name: auth.CookieName, Value: session})
		response := httptest.NewRecorder()
		handler.ServeHTTP(response, request)
		return response.Code, response.Body.String()
	}
	type outcome struct {
		ClusterBand    float64 `json:"clusterBand"`
		Clusters       int     `json:"clusters"`
		Grouped        int     `json:"grouped"`
		LargestCluster int     `json:"largestCluster"`
		Ungrouped      int     `json:"ungrouped"`
	}
	var preview struct {
		Embedded int     `json:"embedded"`
		Semantic bool    `json:"semantic"`
		Current  outcome `json:"current"`
		Proposed outcome `json:"proposed"`
	}

	status, body := get("?cluster_band=3.5")
	if status != 200 {
		t.Fatalf("preview: %d %s", status, body)
	}
	if err := json.Unmarshal([]byte(body), &preview); err != nil {
		t.Fatalf("preview: %v\n%s", err, body)
	}
	if !preview.Semantic {
		t.Fatalf("the backend was not accepted as semantic, so this test covers the same case as the other one: %s", body)
	}
	if preview.Embedded != 40 {
		t.Fatalf("%d of 40 notes embedded: %s", preview.Embedded, body)
	}

	// The figures the other test proves absent must be present here, and must
	// account for every embedded note.
	if preview.Current.Clusters < 1 || preview.Current.LargestCluster < 2 {
		t.Fatalf("the band in force formed %d groups, largest %d: %s",
			preview.Current.Clusters, preview.Current.LargestCluster, body)
	}
	for label, value := range map[string]outcome{"current": preview.Current, "proposed": preview.Proposed} {
		if value.Grouped+value.Ungrouped != preview.Embedded {
			t.Fatalf("%s: %d grouped + %d alone != %d embedded: %s",
				label, value.Grouped, value.Ungrouped, preview.Embedded, body)
		}
	}
	// And the proposal is measured rather than echoed: a band this high admits
	// almost nothing, so the grouping has to fall.
	if preview.Proposed.ClusterBand != 3.5 {
		t.Fatalf("the proposal came back as %v: %s", preview.Proposed.ClusterBand, body)
	}
	if preview.Proposed.Grouped >= preview.Current.Grouped {
		t.Fatalf("band 3.5 grouped %d, the band in force grouped %d: %s",
			preview.Proposed.Grouped, preview.Current.Grouped, body)
	}
}
