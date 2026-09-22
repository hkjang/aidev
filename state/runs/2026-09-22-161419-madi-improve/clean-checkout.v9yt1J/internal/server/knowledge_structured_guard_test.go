package server

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestPostgresStructuredQuietStreamRevokesChangedDependencies(t *testing.T) {
	for _, reason := range []string{"source", "schema", "writer_role", "provider", "session"} {
		t.Run(reason, func(t *testing.T) {
			f := newStructuredFixture(t)
			s, c := f.server, f.owner
			cancelled := make(chan struct{}, 1)
			provider := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				fmt.Fprint(w, "data: {\"choices\":[{\"delta\":{\"content\":\"FIRST_STRUCTURED\"}}]}\n\n")
				w.(http.Flusher).Flush()
				<-r.Context().Done()
				cancelled <- struct{}{}
			}))
			defer provider.Close()
			c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_base_url": provider.URL + "/v1"}, 200)
			path := "/api/v1/documents/" + f.docID
			meta := testJSONObject(t, c.request("GET", path+"/structured-context?database_id="+f.dbID, nil, 200))
			input := map[string]any{"database_id": f.dbID, "expected_version": 1, "start_byte": 0, "end_byte": f.ticket.End, "selected_text": "근거 수량은 2개입니다.", "property_ids": []string{"amount"}, "provider_fingerprint": meta["provider"].(map[string]any)["fingerprint"], "schema_hash": meta["schema_hash"], "destination_hash": meta["destination_hash"], "consent": true}
			ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
			defer cancel()
			req, e := http.NewRequestWithContext(ctx, "POST", c.base+path+"/structured-draft", bytes.NewReader(jsonValue(input)))
			if e != nil {
				t.Fatal(e)
			}
			req.Header.Set("X-Madi-Request", "1")
			req.Header.Set("Content-Type", "application/json")
			response, e := c.client.Do(req)
			if e != nil {
				t.Fatal(e)
			}
			defer response.Body.Close()
			if response.StatusCode != 200 {
				raw, _ := io.ReadAll(response.Body)
				t.Fatal(response.StatusCode, string(raw))
			}
			reader := bufio.NewReader(response.Body)
			for {
				line, e := reader.ReadString('\n')
				if e != nil {
					t.Fatal(e)
				}
				if strings.Contains(line, "FIRST_STRUCTURED") || strings.Contains(line, `"text":`) {
					t.Fatal("unvalidated raw delta was exposed", line)
				}
				if strings.Contains(line, `"phase":"generating"`) {
					break
				}
			}
			switch reason {
			case "source":
				c.request("PUT", path, map[string]any{"version": 1, "markdown": "변경한 수량"}, 200)
			case "schema":
				c.request("PUT", "/api/v1/databases/"+f.dbID, map[string]any{"properties": []any{map[string]any{"id": "amount", "name": "숫자 아닌값", "type": "text"}}}, 200)
			case "writer_role":
				if _, e = s.DB.Exec(t.Context(), `UPDATE workspace_members SET role='viewer' WHERE workspace_id=$1 AND user_id=$2`, f.wid, f.ticket.OwnerID); e != nil {
					t.Fatal(e)
				}
			case "provider":
				c.request("PUT", "/api/v1/admin/settings", map[string]any{"ai_model": "changed"}, 200)
			case "session":
				c.request("POST", "/api/v1/auth/logout", nil, 200)
			}
			select {
			case <-cancelled:
			case <-time.After(5 * time.Second):
				t.Fatal("quiet stream provider was not cancelled")
			}
			remaining, e := io.ReadAll(reader)
			if e != nil {
				t.Fatal(e)
			}
			if !strings.Contains(string(remaining), `"retract":true`) || strings.Contains(string(remaining), `"proposal":`) || strings.Contains(string(remaining), "FIRST_STRUCTURED") {
				t.Fatal(string(remaining))
			}
			var count int
			if e = s.DB.QueryRow(t.Context(), `SELECT count(*) FROM knowledge_structured_drafts`).Scan(&count); e != nil || count != 1 {
				t.Fatal("interrupted generation persisted a draft", count, e)
			}
		})
	}
}
