package httpapi

import (
	"sort"
	"testing"
)

// An attribute an agent reported as a JSON number has to be selectable by
// equality in both storage modes. PostgreSQL's #>> renders every stored value
// as text, so "attributes.cpu_count = 8" compared '8' with '8' and matched.
// SQLite's json_extract hands back the value with its SQL type, and comparing
// an INTEGER with a TEXT parameter is false in every case there - so the same
// query answered HTTP 200 with an empty list, as though no host had reported
// the attribute at all.
func TestNumericAttributeEqualityMatchesInBothStorageModes(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	insertAssetWithAttributes(t, server, "eight", `{"cpu_count":8}`)
	insertAssetWithAttributes(t, server, "sixteen", `{"cpu_count":16}`)
	insertAssetWithAttributes(t, server, "unreported", `{"os_name":"Ubuntu"}`)

	if found := executeQueryNames(
		t, server, cookie, csrf, `attributes.cpu_count = 8`,
	); len(found) != 1 || found[0] != "eight" {
		t.Fatalf("attributes.cpu_count = 8 returned %v", found)
	}

	// The value is the same number however it is written in the query.
	if found := executeQueryNames(
		t, server, cookie, csrf, `attributes.cpu_count = "16"`,
	); len(found) != 1 || found[0] != "sixteen" {
		t.Fatalf(`attributes.cpu_count = "16" returned %v`, found)
	}

	// The mirror image: an inequality used to keep every asset, including the
	// one holding the excluded number. An asset that never reported the key is
	// still included, which is what != means for an attribute path here.
	excluded := executeQueryNames(
		t, server, cookie, csrf, `attributes.cpu_count != 8`,
	)
	sort.Strings(excluded)
	if len(excluded) != 2 ||
		excluded[0] != "sixteen" || excluded[1] != "unreported" {
		t.Fatalf("attributes.cpu_count != 8 returned %v", excluded)
	}

	// The ordering comparison still compares numbers as numbers rather than
	// as the digit strings it fell back to before.
	if found := executeQueryNames(
		t, server, cookie, csrf, `attributes.cpu_count >= 16`,
	); len(found) != 1 || found[0] != "sixteen" {
		t.Fatalf("attributes.cpu_count >= 16 returned %v", found)
	}
}

// A JSON boolean is rendered as 'true'/'false' by PostgreSQL's #>> and as the
// integers 1/0 by SQLite's json_extract, so the spelling an operator has to
// type used to depend on which engine the deployment runs.
func TestBooleanAttributeEqualityMatchesInBothStorageModes(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	insertAssetWithAttributes(t, server, "encrypted", `{"disk_encrypted":true}`)
	insertAssetWithAttributes(t, server, "plain", `{"disk_encrypted":false}`)

	if found := executeQueryNames(
		t, server, cookie, csrf, `attributes.disk_encrypted = "true"`,
	); len(found) != 1 || found[0] != "encrypted" {
		t.Fatalf(`attributes.disk_encrypted = "true" returned %v`, found)
	}
	if found := executeQueryNames(
		t, server, cookie, csrf, `attributes.disk_encrypted = "false"`,
	); len(found) != 1 || found[0] != "plain" {
		t.Fatalf(`attributes.disk_encrypted = "false" returned %v`, found)
	}
}

// A text attribute is the case the extraction was written for, and it has to
// keep answering the same way after numbers and booleans were folded to text.
func TestTextAttributeEqualityIsUnchanged(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	insertAssetWithAttributes(t, server, "ubuntu", `{"os_name":"Ubuntu"}`)
	insertAssetWithAttributes(t, server, "rhel", `{"os_name":"RHEL"}`)

	if found := executeQueryNames(
		t, server, cookie, csrf, `attributes.os_name = "Ubuntu"`,
	); len(found) != 1 || found[0] != "ubuntu" {
		t.Fatalf(`attributes.os_name = "Ubuntu" returned %v`, found)
	}
	// A version written as a string stays a string comparison: reading "1.10"
	// as a number would stop an attribute holding a version matching itself.
	insertAssetWithAttributes(t, server, "versioned", `{"os_version":"1.10"}`)
	if found := executeQueryNames(
		t, server, cookie, csrf, `attributes.os_version = "1.10"`,
	); len(found) != 1 || found[0] != "versioned" {
		t.Fatalf(`attributes.os_version = "1.10" returned %v`, found)
	}
}
