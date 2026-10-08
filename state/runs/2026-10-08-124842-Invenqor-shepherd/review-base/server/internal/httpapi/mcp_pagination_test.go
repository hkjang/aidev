package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
)

// The MCP tools are read by a model, and a model believes what the page tells
// it. `has_more` used to be `len(items) == limit`, which is a guess, not an
// answer: a result that happens to end exactly on the page boundary reported
// that more was waiting, so the caller asked once more and was handed an empty
// page. Reading one row past the page turns the guess into a fact.

type mcpAssetPage struct {
	Items      []assetView `json:"items"`
	Limit      int         `json:"limit"`
	Offset     int         `json:"offset"`
	HasMore    bool        `json:"has_more"`
	NextOffset int         `json:"next_offset"`
}

func mcpAssetSearchPage(t *testing.T, server *Server, raw string) mcpAssetPage {
	t.Helper()
	result, err := server.mcpAssetSearch(
		httptest.NewRequest("POST", "/mcp", nil),
		mcpArgumentsForTest("asset_search", []byte(raw)),
	)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(result)
	var page mcpAssetPage
	if err := json.Unmarshal(encoded, &page); err != nil {
		t.Fatal(err)
	}
	return page
}

func TestMCPAssetSearchReportsWhetherAnotherPageExists(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	now := time.Now().UTC()
	for index, name := range []string{"page-host-a", "page-host-b", "page-host-c"} {
		insertSoftwareTestAsset(t, server, name, name, "host", `{}`, 1,
			now.Add(-time.Duration(index)*time.Minute))
	}

	// Three rows read three at a time is a complete answer, even though the page
	// is full.
	exact := mcpAssetSearchPage(t, server, `{"limit":3}`)
	if len(exact.Items) != 3 || exact.HasMore || exact.NextOffset != 3 {
		t.Fatalf("a result ending exactly on the page boundary = %+v", exact)
	}

	first := mcpAssetSearchPage(t, server, `{"limit":2}`)
	if len(first.Items) != 2 || !first.HasMore || first.NextOffset != 2 {
		t.Fatalf("first page of three rows = %+v", first)
	}
	// The lookahead row must be dropped, not returned as a fourth item on a
	// two-row page.
	if first.Items[0].Name != "page-host-a" || first.Items[1].Name != "page-host-b" {
		t.Fatalf("first page items = %+v", first.Items)
	}

	last := mcpAssetSearchPage(t, server, `{"limit":2,"offset":2}`)
	if len(last.Items) != 1 || last.HasMore ||
		last.Items[0].Name != "page-host-c" {
		t.Fatalf("last page of three rows = %+v", last)
	}
}

type mcpAgentPage struct {
	Agents []struct {
		Hostname string `json:"hostname"`
	} `json:"agents"`
	Limit      int  `json:"limit"`
	Offset     int  `json:"offset"`
	Count      int  `json:"count"`
	HasMore    bool `json:"has_more"`
	NextOffset int  `json:"next_offset"`
}

func mcpAgentsListPage(t *testing.T, server *Server, raw string) mcpAgentPage {
	t.Helper()
	result, err := server.mcpAgentsList(
		httptest.NewRequest("POST", "/mcp", nil),
		mcpArgumentsForTest("agents_list", []byte(raw)),
	)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(result)
	var page mcpAgentPage
	if err := json.Unmarshal(encoded, &page); err != nil {
		t.Fatal(err)
	}
	return page
}

// agents_list promised a next page it had no way to serve: it reported
// `has_more` from the page size and accepted no offset at all, so a caller told
// there was more could only ask the identical question again.
func TestMCPAgentsListPagesPastTheFirstPage(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	now := time.Now().UTC()
	for index, name := range []string{"page-agent-a", "page-agent-b", "page-agent-c"} {
		insertSoftwareTestAsset(t, server, name, name, "host", `{}`, 1,
			now.Add(-time.Duration(index)*time.Minute))
	}

	exact := mcpAgentsListPage(t, server, `{"limit":3}`)
	if exact.Count != 3 || exact.HasMore || exact.NextOffset != 3 {
		t.Fatalf("a result ending exactly on the page boundary = %+v", exact)
	}

	first := mcpAgentsListPage(t, server, `{"limit":2}`)
	if first.Count != 2 || len(first.Agents) != 2 || !first.HasMore ||
		first.NextOffset != 2 {
		t.Fatalf("first page of three agents = %+v", first)
	}
	if first.Agents[0].Hostname != "page-agent-a" ||
		first.Agents[1].Hostname != "page-agent-b" {
		t.Fatalf("first page agents = %+v", first.Agents)
	}

	// The offset the first page hands back must actually reach the remainder.
	last := mcpAgentsListPage(t, server, `{"limit":2,"offset":2}`)
	if last.Count != 1 || last.HasMore || last.Offset != 2 ||
		last.Agents[0].Hostname != "page-agent-c" {
		t.Fatalf("page after next_offset = %+v", last)
	}
}

type mcpRelationPage struct {
	Relations []struct {
		TargetAssetID string `json:"target_asset_id"`
	} `json:"relations"`
	Limit      int  `json:"limit"`
	Offset     int  `json:"offset"`
	Count      int  `json:"count"`
	HasMore    bool `json:"has_more"`
	NextOffset int  `json:"next_offset"`
}

func mcpAssetRelationsPage(t *testing.T, server *Server, raw string) mcpRelationPage {
	t.Helper()
	result, err := server.mcpAssetRelations(
		httptest.NewRequest("POST", "/mcp", nil),
		mcpArgumentsForTest("asset_relations", []byte(raw)),
	)
	if err != nil {
		t.Fatal(err)
	}
	encoded, _ := json.Marshal(result)
	var page mcpRelationPage
	if err := json.Unmarshal(encoded, &page); err != nil {
		t.Fatal(err)
	}
	return page
}

// asset_relations was the one read tool with no ceiling at all: it returned
// every active edge of the asset, so a single host with thousands of `runs_on`
// children answered with a reply large enough to swallow the model's context,
// and the client was left to truncate it.
func TestMCPAssetRelationsPagesInsteadOfReturningEveryEdge(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	now := time.Now().UTC()
	hostID := insertSoftwareTestAsset(t, server, "relation-host", "relation-host",
		"host", `{}`, 1, now)
	targets := make([]string, 0, 3)
	for index, name := range []string{"edge-a", "edge-b", "edge-c"} {
		childID := insertSoftwareTestAsset(t, server, name, name, "process", `{}`, 1,
			now.Add(-time.Duration(index)*time.Minute))
		targets = append(targets, childID)
		if _, err := server.database.DB().Exec(
			`INSERT INTO asset_relations(
				id,source_asset_id,relation_type,target_asset_id,source,confidence
			 ) VALUES($1,$2,'runs_on',$3,'manual',1.0)`,
			uuid.NewString(), hostID, childID,
		); err != nil {
			t.Fatalf("seed relation %s: %v", name, err)
		}
	}

	all := mcpAssetRelationsPage(t, server,
		`{"asset_id":"`+hostID+`","limit":3}`)
	if all.Count != 3 || all.HasMore || all.NextOffset != 3 || all.Limit != 3 {
		t.Fatalf("three edges read three at a time = %+v", all)
	}

	first := mcpAssetRelationsPage(t, server,
		`{"asset_id":"`+hostID+`","limit":2}`)
	if first.Count != 2 || len(first.Relations) != 2 || !first.HasMore ||
		first.NextOffset != 2 {
		t.Fatalf("first page of three edges = %+v", first)
	}

	last := mcpAssetRelationsPage(t, server,
		`{"asset_id":"`+hostID+`","limit":2,"offset":2}`)
	if last.Count != 1 || last.HasMore || last.Offset != 2 {
		t.Fatalf("page after next_offset = %+v", last)
	}
	// The three pages together must be the three edges, each once: a lookahead
	// row that leaks into a page would repeat one of them.
	seen := map[string]int{}
	for _, relation := range append(first.Relations, last.Relations...) {
		seen[relation.TargetAssetID]++
	}
	for _, target := range targets {
		if seen[target] != 1 {
			t.Fatalf("edge %s appeared %d times across the pages", target, seen[target])
		}
	}
}

// A relation list with no limit at all is what this tool used to return, so the
// default has to be a real ceiling rather than a value the caller must pass.
func TestMCPAssetRelationsDefaultsToABoundedPage(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	now := time.Now().UTC()
	hostID := insertSoftwareTestAsset(t, server, "wide-host", "wide-host",
		"host", `{}`, 1, now)
	for index := 0; index < 55; index++ {
		name := fmt.Sprintf("wide-edge-%02d", index)
		childID := insertSoftwareTestAsset(t, server, name, name, "process", `{}`, 1, now)
		if _, err := server.database.DB().Exec(
			`INSERT INTO asset_relations(
				id,source_asset_id,relation_type,target_asset_id,source,confidence
			 ) VALUES($1,$2,'runs_on',$3,'manual',1.0)`,
			uuid.NewString(), hostID, childID,
		); err != nil {
			t.Fatalf("seed relation %s: %v", name, err)
		}
	}

	page := mcpAssetRelationsPage(t, server, `{"asset_id":"`+hostID+`"}`)
	if page.Count != 50 || page.Limit != 50 || !page.HasMore ||
		page.NextOffset != 50 {
		t.Fatalf("default page of 55 edges = %+v", page)
	}
}

// The offset only reaches the remainder because the tool advertises it; a
// parameter the schema omits is rejected before the handler ever sees it.
func TestMCPAgentsListAdvertisesItsOffset(t *testing.T) {
	for _, tool := range mcpTools {
		if tool.Name != "agents_list" {
			continue
		}
		properties, _ := tool.InputSchema["properties"].(map[string]any)
		if _, declared := properties["offset"]; !declared {
			t.Fatal("agents_list reports has_more but declares no offset to act on it")
		}
		return
	}
	t.Fatal("agents_list is not offered")
}
