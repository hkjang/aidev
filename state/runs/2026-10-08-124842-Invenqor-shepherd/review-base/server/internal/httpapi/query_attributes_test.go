package httpapi

import (
	"net/http"
	"sort"
	"testing"

	"github.com/google/uuid"
)

func insertAssetWithAttributes(
	t *testing.T, server *Server, name string, attributes string,
) {
	t.Helper()
	if _, err := server.database.DB().Exec(
		`INSERT INTO assets(
			id, asset_key, name, type, status, criticality, environment,
			confidence, attributes_json, custom_fields_json, source,
			first_seen_at, last_seen_at, created_at, updated_at
		 ) VALUES($1,$2,$3,'host','active','normal','other',
		          1.0,$4,'{}','manual',
		          CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,CURRENT_TIMESTAMP,
		          CURRENT_TIMESTAMP)`,
		uuid.NewString(), "attributes-"+name, name, attributes,
	); err != nil {
		t.Fatal(err)
	}
}

// Attribute keys are case sensitive in both storage modes, and an asset created
// through the asset API carries whatever keys the caller wrote. The parser
// folded the whole field to lower case, so a query naming such a key compiled
// to a path the document does not have: HTTP 200, no rows, no error, and
// /query/validate had already reported the expression as valid.
func TestAttributeQueryFindsAKeyWrittenWithCapitals(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	insertAssetWithAttributes(
		t, server, "tagged", `{"assetTag":"AT-1","os":{"Family":"rhel"}}`,
	)
	insertAssetWithAttributes(t, server, "untagged", `{"assetTag":"AT-2"}`)

	found := executeQueryNames(
		t, server, cookie, csrf, `attributes.assetTag = "AT-1"`,
	)
	if len(found) != 1 || found[0] != "tagged" {
		t.Fatalf("attributes.assetTag = %v", found)
	}
	nested := executeQueryNames(
		t, server, cookie, csrf, `attributes.os.Family = "rhel"`,
	)
	if len(nested) != 1 || nested[0] != "tagged" {
		t.Fatalf("attributes.os.Family = %v", nested)
	}
	// The lower-cased spelling is a different key, and answering with the
	// capitalised key's rows would only move the surprise elsewhere.
	if folded := executeQueryNames(
		t, server, cookie, csrf, `attributes.assettag = "AT-1"`,
	); len(folded) != 0 {
		t.Fatalf("attributes.assettag = %v", folded)
	}
}

// A column is still named case-insensitively; only the JSON key after
// "attributes." carries meaning in its case.
func TestColumnClauseStaysCaseInsensitiveOverHTTP(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	insertAssetWithAttributes(t, server, "shouty", `{}`)

	if found := executeQueryNames(
		t, server, cookie, csrf, `NAME = "shouty"`,
	); len(found) != 1 || found[0] != "shouty" {
		t.Fatalf(`NAME = "shouty" returned %v`, found)
	}

	validate := performAuthenticatedJSON(
		t, server, http.MethodPost, "/api/v1/query/validate",
		map[string]any{"query": `Attributes.assetTag = "AT-1"`}, cookie, csrf,
	)
	if validate.Code != http.StatusOK {
		t.Fatalf("validate status = %d body = %s",
			validate.Code, validate.Body.String())
	}
}

// An attribute is stored in a JSON document and extracted as text, so an
// ordering comparison compared digit strings: "16000000000" sorts before
// "2000000000" because '1' < '2'. A host with 16 GB was therefore reported as
// holding less than the 2 GB bound, with HTTP 200 and no hint of it.
func TestOrderingAnAttributeComparesNumbersAsNumbers(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	insertAssetWithAttributes(
		t, server, "big", `{"memory_bytes":16000000000,"cpu_count":10}`,
	)
	insertAssetWithAttributes(
		t, server, "small", `{"memory_bytes":1000000000,"cpu_count":9}`,
	)

	roomy := executeQueryNames(
		t, server, cookie, csrf, `attributes.memory_bytes >= 2000000000`,
	)
	if len(roomy) != 1 || roomy[0] != "big" {
		t.Fatalf("attributes.memory_bytes >= 2000000000 returned %v", roomy)
	}
	// "10" > "9" only when the two are read as numbers.
	if busy := executeQueryNames(
		t, server, cookie, csrf, `attributes.cpu_count > 9`,
	); len(busy) != 1 || busy[0] != "big" {
		t.Fatalf("attributes.cpu_count > 9 returned %v", busy)
	}
}

// A value stored as text still orders as text, so reading a numeric-looking
// bound as a number cannot take rows away from a clause that works today.
func TestOrderingAnAttributeStoredAsTextStaysATextComparison(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	insertAssetWithAttributes(t, server, "jammy", `{"os_version":"22.04"}`)
	insertAssetWithAttributes(t, server, "focal", `{"os_version":"18.04"}`)

	found := executeQueryNames(
		t, server, cookie, csrf, `attributes.os_version > "20.04"`,
	)
	if len(found) != 1 || found[0] != "jammy" {
		t.Fatalf(`attributes.os_version > "20.04" returned %v`, found)
	}
}

// An attribute an asset never reported extracts SQL NULL, and "!=" against NULL
// is unknown rather than true, so the asset fell out of a clause it plainly
// satisfies. An operator asking which assets are not production got the answer
// without the ones nobody had labelled at all, with HTTP 200 and no hint of it.
func TestAttributeInequalityReturnsAssetsMissingTheKey(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	insertAssetWithAttributes(t, server, "labelled", `{"env":"prod"}`)
	insertAssetWithAttributes(t, server, "staged", `{"env":"staging"}`)
	insertAssetWithAttributes(t, server, "unlabelled", `{"os_name":"Ubuntu"}`)

	found := executeQueryNames(
		t, server, cookie, csrf, `attributes.env != "prod"`,
	)
	sort.Strings(found)
	if len(found) != 2 || found[0] != "staged" || found[1] != "unlabelled" {
		t.Fatalf(`attributes.env != "prod" returned %v`, found)
	}
	// Equality still means the attribute says so, so the fix cannot hand an
	// unlabelled asset to a clause asking for a value it never reported.
	if labelled := executeQueryNames(
		t, server, cookie, csrf, `attributes.env = "prod"`,
	); len(labelled) != 1 || labelled[0] != "labelled" {
		t.Fatalf(`attributes.env = "prod" returned %v`, labelled)
	}
}
