package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
)

type visualizationResponse struct {
	WindowDays int `json:"window_days"`
	Totals     struct {
		Assets  int64 `json:"assets"`
		Stale   int64 `json:"stale"`
		Fresh   int64 `json:"fresh"`
		Unowned int64 `json:"unowned"`
	} `json:"totals"`
	Dimensions map[string][]struct {
		Label string `json:"label"`
		Count int64  `json:"count"`
	} `json:"dimensions"`
	Matrix struct {
		Rows    []string `json:"rows"`
		Columns []string `json:"columns"`
		Maximum int64    `json:"maximum"`
		Cells   []struct {
			Row    string `json:"row"`
			Column string `json:"column"`
			Count  int64  `json:"count"`
			Stale  int64  `json:"stale"`
		} `json:"cells"`
	} `json:"matrix"`
	Freshness []struct {
		Label    string `json:"label"`
		MaxHours int    `json:"max_hours"`
		Count    int64  `json:"count"`
	} `json:"freshness"`
	Hierarchy []struct {
		Label    string `json:"label"`
		Count    int64  `json:"count"`
		Children []struct {
			Label string `json:"label"`
			Count int64  `json:"count"`
		} `json:"children"`
	} `json:"hierarchy"`
	Flow []struct {
		Date    string `json:"date"`
		Added   int64  `json:"added"`
		Removed int64  `json:"removed"`
		Total   int64  `json:"total"`
	} `json:"flow"`
	Graph struct {
		Nodes []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Type   string `json:"type"`
			Degree int    `json:"degree"`
			Stale  bool   `json:"stale"`
		} `json:"nodes"`
		Edges []struct {
			Source string `json:"source"`
			Target string `json:"target"`
			Type   string `json:"type"`
		} `json:"edges"`
		TotalRelations int64 `json:"total_relations"`
	} `json:"graph"`
}

type seededAsset struct {
	id          string
	name        string
	assetType   string
	environment string
	criticality string
	owner       string
	ageHours    int
	firstSeenAt time.Time
}

func seedVisualizationAssets(t *testing.T, server *Server) []seededAsset {
	t.Helper()
	now := time.Now().UTC()
	assets := []seededAsset{
		{name: "web-01", assetType: "host", environment: "production", criticality: "critical", owner: "infra", ageHours: 1},
		{name: "web-02", assetType: "host", environment: "production", criticality: "critical", owner: "", ageHours: 100},
		{name: "db-01", assetType: "host", environment: "production", criticality: "high", owner: "dba", ageHours: 2},
		{name: "svc-01", assetType: "service", environment: "production", criticality: "high", owner: "apps", ageHours: 800},
		{name: "test-01", assetType: "host", environment: "test", criticality: "low", owner: "qa", ageHours: 3000},
		{name: "pkg-01", assetType: "software", environment: "development", criticality: "normal", owner: "", ageHours: 5},
	}
	for index := range assets {
		assets[index].id = uuid.NewString()
		assets[index].firstSeenAt = now.AddDate(0, 0, -(index + 1))
		if _, err := server.database.DB().Exec(
			`INSERT INTO assets(
				id,asset_key,name,type,status,criticality,environment,
				owner_department,confidence,attributes_json,custom_fields_json,
				source,first_seen_at,last_seen_at,created_at,updated_at
			 ) VALUES($1,$2,$3,$4,'active',$5,$6,$7,1.0,'{}','{}','agent',
			          $8,$9,$8,$9)`,
			assets[index].id,
			fmt.Sprintf("key-%02d", index),
			assets[index].name,
			assets[index].assetType,
			assets[index].criticality,
			assets[index].environment,
			assets[index].owner,
			assets[index].firstSeenAt,
			now.Add(-time.Duration(assets[index].ageHours)*time.Hour),
		); err != nil {
			t.Fatalf("seed asset %s error = %v", assets[index].name, err)
		}
	}
	// One retired asset so the change-flow series has a removal to draw.
	if _, err := server.database.DB().Exec(
		`INSERT INTO assets(
			id,asset_key,name,type,status,criticality,environment,
			confidence,attributes_json,custom_fields_json,source,
			first_seen_at,last_seen_at,created_at,updated_at,deleted_at
		 ) VALUES($1,'key-retired','retired-01','host','removed','normal',
		          'production',1.0,'{}','{}','agent',$2,$3,$2,$3,$3)`,
		uuid.NewString(),
		now.AddDate(0, 0, -3),
		now.Add(-2*time.Hour),
	); err != nil {
		t.Fatalf("seed retired asset error = %v", err)
	}
	// A relation so the topology view has an edge with both ends present.
	if _, err := server.database.DB().Exec(
		`INSERT INTO asset_relations(
			id,source_asset_id,relation_type,target_asset_id,source,confidence
		 ) VALUES($1,$2,'runs_on',$3,'manual',1.0)`,
		uuid.NewString(),
		assets[3].id,
		assets[2].id,
	); err != nil {
		t.Fatalf("seed relation error = %v", err)
	}
	return assets
}

func fetchVisualization(
	t *testing.T,
	server *Server,
	query string,
	cookie *http.Cookie,
) visualizationResponse {
	t.Helper()
	response := performAuthenticatedJSON(
		t, server, http.MethodGet,
		"/api/v1/assets/visualization"+query,
		nil, cookie, "",
	)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
	var payload visualizationResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode visualization response: %v", err)
	}
	return payload
}

func TestAssetVisualizationAggregatesEveryView(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, _ := authenticateInitialAdmin(t, server, runtime)
	assets := seedVisualizationAssets(t, server)

	payload := fetchVisualization(t, server, "?days=14", cookie)

	if payload.Totals.Assets != int64(len(assets)) {
		t.Fatalf("totals.assets = %d, want %d", payload.Totals.Assets, len(assets))
	}
	// web-02, svc-01 and test-01 are older than the 24 hour default.
	if payload.Totals.Stale != 3 {
		t.Fatalf("totals.stale = %d, want 3", payload.Totals.Stale)
	}
	if payload.Totals.Fresh != payload.Totals.Assets-payload.Totals.Stale {
		t.Fatalf("fresh and stale do not add up: %+v", payload.Totals)
	}
	if payload.Totals.Unowned != 2 {
		t.Fatalf("totals.unowned = %d, want 2", payload.Totals.Unowned)
	}

	// The matrix axes must follow the severity and environment scales, not the
	// row counts, or the reader loses the ordering the data has.
	if len(payload.Matrix.Rows) < 2 || payload.Matrix.Rows[0] != "critical" {
		t.Fatalf("matrix rows = %v, want critical first", payload.Matrix.Rows)
	}
	if payload.Matrix.Columns[0] != "production" {
		t.Fatalf("matrix columns = %v, want production first", payload.Matrix.Columns)
	}
	if payload.Matrix.Maximum < 1 {
		t.Fatal("matrix maximum was not reported, so no fill scale is possible")
	}
	criticalProduction := int64(0)
	criticalStale := int64(0)
	for _, cell := range payload.Matrix.Cells {
		if cell.Row == "critical" && cell.Column == "production" {
			criticalProduction = cell.Count
			criticalStale = cell.Stale
		}
	}
	if criticalProduction != 2 || criticalStale != 1 {
		t.Fatalf(
			"critical/production cell = %d assets, %d stale; want 2 and 1",
			criticalProduction, criticalStale,
		)
	}

	// Freshness buckets must partition the inventory exactly once.
	var bucketTotal int64
	for _, bucket := range payload.Freshness {
		bucketTotal += bucket.Count
	}
	if bucketTotal != payload.Totals.Assets {
		t.Fatalf(
			"freshness buckets total %d, want %d (buckets must not overlap or drop rows)",
			bucketTotal, payload.Totals.Assets,
		)
	}
	if len(payload.Freshness) != 5 || payload.Freshness[0].Count != 3 {
		t.Fatalf("freshness = %+v", payload.Freshness)
	}

	// Hierarchy: the largest environment leads, and leaf counts sum to it.
	if len(payload.Hierarchy) == 0 || payload.Hierarchy[0].Label != "production" {
		t.Fatalf("hierarchy = %+v", payload.Hierarchy)
	}
	var leafTotal int64
	for _, leaf := range payload.Hierarchy[0].Children {
		leafTotal += leaf.Count
	}
	if leafTotal != payload.Hierarchy[0].Count {
		t.Fatalf(
			"hierarchy leaves total %d but branch reports %d",
			leafTotal, payload.Hierarchy[0].Count,
		)
	}

	if len(payload.Flow) != 14 {
		t.Fatalf("flow length = %d, want 14 days", len(payload.Flow))
	}
	last := payload.Flow[len(payload.Flow)-1]
	if last.Total != payload.Totals.Assets {
		t.Fatalf(
			"flow ends at %d but the inventory holds %d; the curve must agree with the tiles",
			last.Total, payload.Totals.Assets,
		)
	}
	var added, removed int64
	for _, day := range payload.Flow {
		added += day.Added
		removed += day.Removed
	}
	if added != int64(len(assets)) || removed != 1 {
		t.Fatalf("flow added = %d, removed = %d", added, removed)
	}

	if payload.Graph.TotalRelations != 1 {
		t.Fatalf("graph.total_relations = %d, want 1", payload.Graph.TotalRelations)
	}
	if len(payload.Graph.Edges) != 1 {
		t.Fatalf("graph edges = %+v", payload.Graph.Edges)
	}
	degreeSeen := false
	for _, node := range payload.Graph.Nodes {
		if node.Degree > 0 {
			degreeSeen = true
		}
	}
	if !degreeSeen {
		t.Fatal("no node reported a relation degree, so node size cannot encode it")
	}
	if len(payload.Graph.Nodes) != len(assets) {
		t.Fatalf("graph nodes = %d, want %d", len(payload.Graph.Nodes), len(assets))
	}
}

// A filter has to scope every block, otherwise two views on one screen disagree.
func TestAssetVisualizationFilterScopesEveryBlock(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, _ := authenticateInitialAdmin(t, server, runtime)
	seedVisualizationAssets(t, server)

	payload := fetchVisualization(t, server, "?days=7&environment=production", cookie)

	if payload.Totals.Assets != 4 {
		t.Fatalf("filtered totals.assets = %d, want 4", payload.Totals.Assets)
	}
	for _, column := range payload.Matrix.Columns {
		if column != "production" {
			t.Fatalf("matrix column %q survived the environment filter", column)
		}
	}
	for _, branch := range payload.Hierarchy {
		if branch.Label != "production" {
			t.Fatalf("hierarchy branch %q survived the environment filter", branch.Label)
		}
	}
	var bucketTotal int64
	for _, bucket := range payload.Freshness {
		bucketTotal += bucket.Count
	}
	if bucketTotal != payload.Totals.Assets {
		t.Fatalf("filtered freshness total = %d, want %d", bucketTotal, payload.Totals.Assets)
	}
	for _, node := range payload.Graph.Nodes {
		if node.Name == "test-01" {
			t.Fatal("topology kept an asset outside the filter")
		}
	}
	for _, environment := range payload.Dimensions["environment"] {
		if environment.Label != "production" {
			t.Fatalf("dimension %q survived the environment filter", environment.Label)
		}
	}
	if len(payload.Flow) != 7 {
		t.Fatalf("flow length = %d, want 7", len(payload.Flow))
	}
}

// The static path must win over /api/v1/assets/{assetID}; otherwise the view
// would be answered by the single-asset handler with a not-found.
func TestAssetVisualizationPathIsNotCapturedByTheAssetDetailRoute(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, _ := authenticateInitialAdmin(t, server, runtime)
	response := performAuthenticatedJSON(
		t, server, http.MethodGet, "/api/v1/assets/visualization", nil, cookie, "",
	)
	if response.Code != http.StatusOK {
		t.Fatalf("status = %d body = %s", response.Code, response.Body.String())
	}
}
