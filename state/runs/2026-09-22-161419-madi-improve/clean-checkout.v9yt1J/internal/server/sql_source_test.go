package server

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestSQLSourceClosedGrammar(t *testing.T) {
	cols := []sqlColumn{{Name: "name", Type: "text"}, {Name: "amount", Type: "number"}}
	plan := SQLSourcePlan{Schema: "public", Table: "orders", Columns: []string{"name", "amount"}, Limit: 10, Filters: []SQLSourceFilter{{Column: "name", Operator: "contains", Value: "%'; DROP TABLE secrets;--"}}, Order: []SQLSourceOrder{{Column: "amount", Direction: "desc"}}}
	for _, kind := range []string{"postgres", "mysql", "mariadb", "mssql", "oracle"} {
		query, args, e := buildSQLSourceQuery(kind, plan, cols, 500)
		if e != nil || len(args) != 1 || strings.Contains(query, "DROP") {
			t.Fatalf("%s unsafe %s %#v %v", kind, query, args, e)
		}
	}
	for _, bad := range []string{"name);DELETE FROM users;--", "*", "name --", "pg_sleep(1)", "name FROM secrets"} {
		modified := plan
		modified.Columns = []string{bad}
		if _, _, e := buildSQLSourceQuery("postgres", modified, cols, 500); e == nil {
			t.Fatal("SQL injection grammar accepted", bad)
		}
	}
	if _, e := sqlSourcePlan(map[string]any{"schema": "public", "table": "orders", "sql": "DROP TABLE users"}); e == nil {
		t.Fatal("arbitrary SQL field accepted")
	}
}
func TestPostgresSQLSourceReadonlyMetadataQueryAndRevocation(t *testing.T) {
	s, client, ctx, _, wid := jobTestFixture(t)
	cfg := s.DB.Config().ConnConfig
	var schema string
	if e := s.DB.QueryRow(ctx, "SELECT current_schema()").Scan(&schema); e != nil {
		t.Fatal(e)
	}
	role := "madi_sql_" + strings.ReplaceAll(newID(), "-", "")
	password := randomToken()
	quotedRole := pgx.Identifier{role}.Sanitize()
	quotedSchema := pgx.Identifier{schema}.Sanitize()
	table := pgx.Identifier{schema, "source_orders"}.Sanitize()
	// Utility statements cannot bind PASSWORD directly. Let PostgreSQL quote
	// both values, and use the same random credential for the actual connection
	// so the fixture also works with CI's SCRAM authentication (not only trust).
	var createRole string
	if e := s.DB.QueryRow(ctx, "SELECT format('CREATE ROLE %I LOGIN PASSWORD %L', $1::text, $2::text)", role, password).Scan(&createRole); e != nil {
		t.Fatal(e)
	}
	if _, e := s.DB.Exec(ctx, createRole); e != nil {
		t.Fatal(e)
	}
	t.Cleanup(func() {
		_, _ = s.DB.Exec(context.Background(), "REVOKE ALL ON TABLE "+table+" FROM "+quotedRole)
		_, _ = s.DB.Exec(context.Background(), "REVOKE USAGE ON SCHEMA "+quotedSchema+" FROM "+quotedRole)
		_, _ = s.DB.Exec(context.Background(), "DROP ROLE "+quotedRole)
	})
	if _, e := s.DB.Exec(ctx, "CREATE TABLE "+table+"(name text,amount integer); INSERT INTO "+table+" VALUES('운영',10),('개발',20),('테스트',30); GRANT USAGE ON SCHEMA "+quotedSchema+" TO "+quotedRole+"; GRANT SELECT ON "+table+" TO "+quotedRole); e != nil {
		t.Fatal(e)
	}
	client.request("PUT", "/api/v1/admin/connectors/settings", map[string]any{"enabled": true, "allowed_hosts": []string{cfg.Host}}, 200)
	service := testJSONObject(t, client.request("POST", "/api/v1/admin/users", map[string]any{"email": "sql-source@example.test", "name": "SQL 연동 계정", "kind": "service", "role": "editor"}, 200))
	client.request("PUT", "/api/v1/workspaces/"+wid+"/members", map[string]any{"email": "sql-source@example.test", "role": "editor"}, 200)
	input := map[string]any{"workspace_id": wid, "space_id": "", "service_account_id": service["id"], "name": "실제 PostgreSQL 읽기 전용", "kind": "postgres", "enabled": true, "credentials": map[string]any{"username": role, "password": password}, "config": map[string]any{"host": cfg.Host, "port": int(cfg.Port), "database": cfg.Database, "allow_plaintext": true, "acknowledge_readonly": true, "acknowledge_acl": true, "timeout_seconds": 10, "max_rows": 50, "tables": []string{schema + ".source_orders"}}}
	source := testJSONObject(t, client.request("POST", "/api/v1/data-sources", input, 200))
	id := str(source, "id")
	if strings.Contains(string(jsonValue(source)), password) || strings.Contains(string(jsonValue(source)), role) {
		t.Fatal("SQL credentials returned")
	}
	inspected := testJSONObject(t, client.request("POST", "/api/v1/data-sources/"+id+"/inspect", map[string]any{}, 200))
	if !boolean(inspected, "readonly_verified") {
		t.Fatal("read-only grant verification missing")
	}
	plan := SQLSourcePlan{Schema: schema, Table: "source_orders", Columns: []string{"name", "amount"}, Order: []SQLSourceOrder{{Column: "amount", Direction: "desc"}}, Limit: 2}
	query := testJSONObject(t, client.request("POST", "/api/v1/data-sources/"+id+"/queries", map[string]any{"name": "상위 금액", "plan": plan, "enabled": true}, 200))
	result := testJSONObject(t, client.request("POST", "/api/v1/data-sources/"+id+"/queries/"+str(query, "id")+"/execute", map[string]any{}, 200))
	if !boolean(result, "truncated") || len(result["rows"].([]any)) != 2 || str(result["rows"].([]any)[0].(map[string]any), "name") != "테스트" {
		t.Fatalf("actual query result %#v", result)
	}
	issued := testJSONObject(t, client.request("POST", "/api/v1/keys", map[string]any{"name": "SQL 읽기 검증", "workspace_id": wid, "scopes": []string{"database:read"}}, 201))
	keyClient := newIntegrationTestClient(t, client.base)
	keyClient.token = str(issued, "token")
	keyClient.request("GET", "/api/v1/data-sources/"+id, nil, 200)
	keyClient.request("POST", "/api/v1/data-sources/"+id+"/queries/"+str(query, "id")+"/execute", map[string]any{}, 200)
	keyClient.request("POST", "/api/v1/data-sources/"+id+"/inspect", map[string]any{}, 403)
	keyClient.request("POST", "/api/v1/data-sources/"+id+"/queries", map[string]any{"name": "금지", "plan": plan}, 403)
	config, _ := s.loadSQLSource(ctx, id)
	db, e := s.openSQLSource(ctx, config)
	if e != nil {
		t.Fatal(e)
	}
	defer db.Close()
	tx, e := beginSQLSource(ctx, db, config.Kind)
	if e != nil {
		t.Fatal(e)
	}
	var readonly string
	if e = tx.QueryRowContext(ctx, "SHOW transaction_read_only").Scan(&readonly); e != nil || readonly != "on" {
		t.Fatal("database transaction not readonly", e)
	}
	if _, e = tx.ExecContext(ctx, "INSERT INTO "+table+" VALUES('forbidden',1)"); e == nil {
		t.Fatal("read-only transaction wrote data")
	}
	_ = tx.Rollback()
	plan.Columns = []string{"pg_sleep(1)"}
	client.request("POST", "/api/v1/data-sources/"+id+"/queries", map[string]any{"name": "거부", "plan": plan, "enabled": true}, 400)
	client.request("PUT", "/api/v1/admin/users/"+str(service, "id"), map[string]any{"disabled": true}, 200)
	client.request("POST", "/api/v1/data-sources/"+id+"/queries/"+str(query, "id")+"/execute", map[string]any{}, 403)
	client.request("POST", "/api/v1/data-sources/"+id+"/inspect", map[string]any{}, 403)
	client.request("PUT", "/api/v1/admin/users/"+str(service, "id"), map[string]any{"disabled": false}, 200)
	// Adding a write grant invalidates the read-only account on its next operation.
	if _, e = s.DB.Exec(ctx, "GRANT INSERT ON "+table+" TO "+quotedRole); e != nil {
		t.Fatal(e)
	}
	client.request("POST", "/api/v1/data-sources/"+id+"/inspect", map[string]any{}, 400)
	var count int
	if e = s.DB.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&count); e != nil || count != 3 {
		t.Fatal("remote rows changed")
	}
	var runs []byte
	_ = s.DB.QueryRow(ctx, "SELECT coalesce(jsonb_agg(to_jsonb(r)),'[]') FROM sql_source_runs r WHERE source_id=$1", id).Scan(&runs)
	var log []map[string]any
	_ = json.Unmarshal(runs, &log)
	if len(log) < 1 || strings.Contains(string(runs), "테스트") {
		t.Fatal("query log absent or result values leaked")
	}
}
