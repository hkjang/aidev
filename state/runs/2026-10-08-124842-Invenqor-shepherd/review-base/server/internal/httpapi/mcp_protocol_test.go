package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/hkjang/invenqor/server/internal/storagetest"

	"github.com/hkjang/invenqor/server/internal/version"
)

func TestMCPModernStatelessDiscoveryAndToolCalls(t *testing.T) {
	server, secret := mcpProtocolTestServer(t)

	discovery := performMCPRequest(t, server, secret, map[string]any{
		"jsonrpc": "2.0",
		"id":      "discover-1",
		"method":  "server/discover",
		"params": map[string]any{
			"_meta": modernMCPMetadata(),
		},
	}, map[string]string{
		"MCP-Protocol-Version": mcpModernProtocolVersion,
		"Mcp-Method":           "server/discover",
	})
	if discovery.Code != http.StatusOK {
		t.Fatalf("discovery status = %d body = %s", discovery.Code, discovery.Body)
	}
	if got := discovery.Header().Get("MCP-Protocol-Version"); got != mcpModernProtocolVersion {
		t.Fatalf("discovery response protocol = %q", got)
	}
	discoveryResult := decodeMCPResult(t, discovery)
	if discoveryResult["resultType"] != "complete" ||
		discoveryResult["cacheScope"] != "private" ||
		discoveryResult["ttlMs"] != float64(0) {
		t.Fatalf("modern discovery contract = %#v", discoveryResult)
	}
	supported, ok := discoveryResult["supportedVersions"].([]any)
	if !ok || len(supported) != 1 || supported[0] != mcpModernProtocolVersion {
		t.Fatalf("supported versions = %#v", discoveryResult["supportedVersions"])
	}
	assertModernMCPServerInfo(t, discoveryResult)

	listed := performMCPRequest(t, server, secret, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params": map[string]any{
			"_meta": modernMCPMetadata(),
		},
	}, map[string]string{
		"MCP-Protocol-Version": mcpModernProtocolVersion,
		"Mcp-Method":           "tools/list",
	})
	if listed.Code != http.StatusOK {
		t.Fatalf("tools/list status = %d body = %s", listed.Code, listed.Body)
	}
	listResult := decodeMCPResult(t, listed)
	if listResult["resultType"] != "complete" ||
		listResult["cacheScope"] != "private" ||
		listResult["ttlMs"] != float64(0) {
		t.Fatalf("modern tools/list contract = %#v", listResult)
	}
	tools, ok := listResult["tools"].([]any)
	if !ok || len(tools) != 3 {
		t.Fatalf("scope-filtered tools = %#v", listResult["tools"])
	}
	for _, item := range tools {
		tool := item.(map[string]any)
		if tool["name"] == "agents_list" || tool["name"] == "asset_relations" {
			t.Fatalf("tools/list exposed a scope the API key does not hold: %#v", tool)
		}
	}
	assertModernMCPServerInfo(t, listResult)

	called := performMCPRequest(t, server, secret, map[string]any{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "asset_search",
			"arguments": map[string]any{"limit": 1},
			"_meta":     modernMCPMetadata(),
		},
	}, map[string]string{
		"MCP-Protocol-Version": mcpModernProtocolVersion,
		"Mcp-Method":           "tools/call",
		"Mcp-Name":             "=?base64?YXNzZXRfc2VhcmNo?=",
	})
	if called.Code != http.StatusOK {
		t.Fatalf("tools/call status = %d body = %s", called.Code, called.Body)
	}
	callResult := decodeMCPResult(t, called)
	if callResult["resultType"] != "complete" || callResult["isError"] != false {
		t.Fatalf("modern tools/call contract = %#v", callResult)
	}
	if _, ok := callResult["structuredContent"].(map[string]any); !ok {
		t.Fatalf("tools/call omitted structuredContent: %#v", callResult)
	}
	assertModernMCPServerInfo(t, callResult)
}

func TestMCPModernRequestsRejectMissingOrMismatchedRoutingHeaders(t *testing.T) {
	server, secret := mcpProtocolTestServer(t)
	tests := []struct {
		name    string
		method  string
		params  map[string]any
		headers map[string]string
	}{
		{
			name:   "missing method",
			method: "tools/list",
			params: map[string]any{"_meta": modernMCPMetadata()},
			headers: map[string]string{
				"MCP-Protocol-Version": mcpModernProtocolVersion,
			},
		},
		{
			name:   "mismatched method",
			method: "tools/list",
			params: map[string]any{"_meta": modernMCPMetadata()},
			headers: map[string]string{
				"MCP-Protocol-Version": mcpModernProtocolVersion,
				"Mcp-Method":           "tools/call",
			},
		},
		{
			name:   "missing tool name",
			method: "tools/call",
			params: map[string]any{
				"name": "asset_search", "arguments": map[string]any{},
				"_meta": modernMCPMetadata(),
			},
			headers: map[string]string{
				"MCP-Protocol-Version": mcpModernProtocolVersion,
				"Mcp-Method":           "tools/call",
			},
		},
		{
			name:   "mismatched tool name",
			method: "tools/call",
			params: map[string]any{
				"name": "asset_search", "arguments": map[string]any{},
				"_meta": modernMCPMetadata(),
			},
			headers: map[string]string{
				"MCP-Protocol-Version": mcpModernProtocolVersion,
				"Mcp-Method":           "tools/call",
				"Mcp-Name":             "asset_get",
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performMCPRequest(t, server, secret, map[string]any{
				"jsonrpc": "2.0", "id": 1,
				"method": test.method, "params": test.params,
			}, test.headers)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d body = %s", response.Code, response.Body)
			}
			var payload struct {
				Error struct {
					Code int `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if payload.Error.Code != -32020 {
				t.Fatalf("error code = %d body = %s", payload.Error.Code, response.Body)
			}
		})
	}
}

func TestMCPLegacyHandshakeRemainsCompatible(t *testing.T) {
	server, secret := mcpProtocolTestServer(t)
	response := performMCPRequest(t, server, secret, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": mcpLegacyProtocolVersion,
			"capabilities":    map[string]any{},
			"clientInfo": map[string]string{
				"name": "legacy-test", "version": "1.0.0",
			},
		},
	}, nil)
	if response.Code != http.StatusOK {
		t.Fatalf("legacy initialize status = %d body = %s", response.Code, response.Body)
	}
	result := decodeMCPResult(t, response)
	if result["protocolVersion"] != mcpLegacyProtocolVersion {
		t.Fatalf("legacy protocol = %#v", result["protocolVersion"])
	}
	if _, modernOnly := result["resultType"]; modernOnly {
		t.Fatalf("legacy result unexpectedly contains resultType: %#v", result)
	}
}

func TestMCPUnsupportedProtocolVersionReturnsNegotiationData(t *testing.T) {
	server, secret := mcpProtocolTestServer(t)
	response := performMCPRequest(t, server, secret, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "server/discover",
		"params": map[string]any{"_meta": map[string]any{
			"io.modelcontextprotocol/protocolVersion":    "2099-01-01",
			"io.modelcontextprotocol/clientCapabilities": map[string]any{},
		}},
	}, map[string]string{
		"MCP-Protocol-Version": "2099-01-01",
		"Mcp-Method":           "server/discover",
	})
	if response.Code != http.StatusBadRequest {
		t.Fatalf("unsupported version status = %d body = %s", response.Code, response.Body)
	}
	var payload struct {
		Error struct {
			Code int `json:"code"`
			Data struct {
				Requested string   `json:"requested"`
				Supported []string `json:"supported"`
			} `json:"data"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Error.Code != -32022 || payload.Error.Data.Requested != "2099-01-01" ||
		len(payload.Error.Data.Supported) != 2 ||
		payload.Error.Data.Supported[0] != mcpModernProtocolVersion ||
		payload.Error.Data.Supported[1] != mcpLegacyProtocolVersion {
		t.Fatalf("unsupported version response = %#v", payload)
	}
}

func TestMCPRequestIDContractAndLegacyNotificationSemantics(t *testing.T) {
	server, secret := mcpProtocolTestServer(t)
	invalidIDs := []struct {
		name string
		body string
	}{
		{"missing", `{"jsonrpc":"2.0","method":"ping"}`},
		{"null", `{"jsonrpc":"2.0","id":null,"method":"ping"}`},
		{"boolean", `{"jsonrpc":"2.0","id":true,"method":"ping"}`},
		{"object", `{"jsonrpc":"2.0","id":{},"method":"ping"}`},
		{"array", `{"jsonrpc":"2.0","id":[],"method":"ping"}`},
		{"fraction", `{"jsonrpc":"2.0","id":1.5,"method":"ping"}`},
		{"huge negative exponent", `{"jsonrpc":"2.0","id":1e-99999999999999999999,"method":"ping"}`},
	}
	for _, test := range invalidIDs {
		t.Run(test.name, func(t *testing.T) {
			response := performRawMCPRequest(t, server, secret, test.body, nil)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d body = %s", response.Code, response.Body)
			}
			var payload struct {
				ID    any `json:"id"`
				Error struct {
					Code int `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if payload.Error.Code != -32600 || payload.ID != nil {
				t.Fatalf("invalid ID response = %#v", payload)
			}
		})
	}
	for _, body := range []string{
		`{"jsonrpc":"2.0","id":"request-1","method":"ping"}`,
		`{"jsonrpc":"2.0","id":1000e-3,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":1e99999999999999999999,"method":"ping"}`,
	} {
		response := performRawMCPRequest(t, server, secret, body, nil)
		if response.Code != http.StatusOK {
			t.Fatalf("valid ID status = %d body = %s", response.Code, response.Body)
		}
	}

	legacyNotification := performRawMCPRequest(
		t,
		server,
		secret,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		nil,
	)
	if legacyNotification.Code != http.StatusAccepted {
		t.Fatalf(
			"legacy notification status = %d body = %s",
			legacyNotification.Code,
			legacyNotification.Body,
		)
	}
	legacyRequest := performRawMCPRequest(
		t,
		server,
		secret,
		`{"jsonrpc":"2.0","id":1,"method":"notifications/initialized"}`,
		nil,
	)
	if legacyRequest.Code != http.StatusBadRequest {
		t.Fatalf("legacy notification with ID status = %d", legacyRequest.Code)
	}

	modernBody := map[string]any{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
		"params":  map[string]any{"_meta": modernMCPMetadata()},
	}
	modernNotification := performMCPRequest(t, server, secret, modernBody, map[string]string{
		"MCP-Protocol-Version": mcpModernProtocolVersion,
		"Mcp-Method":           "notifications/initialized",
	})
	if modernNotification.Code != http.StatusAccepted || modernNotification.Body.Len() != 0 {
		t.Fatalf(
			"modern removed notification status = %d body = %s",
			modernNotification.Code,
			modernNotification.Body,
		)
	}
	modernRequest := performMCPRequest(t, server, secret, map[string]any{
		"jsonrpc": "2.0", "id": "removed-notification",
		"method": "notifications/initialized",
		"params": map[string]any{"_meta": modernMCPMetadata()},
	}, map[string]string{
		"MCP-Protocol-Version": mcpModernProtocolVersion,
		"Mcp-Method":           "notifications/initialized",
	})
	if modernRequest.Code != http.StatusNotFound {
		t.Fatalf(
			"modern removed notification request status = %d body = %s",
			modernRequest.Code,
			modernRequest.Body,
		)
	}
}

func TestMCPModernRequiresClientCapabilitiesAndRejectsRemovedMethods(t *testing.T) {
	server, secret := mcpProtocolTestServer(t)
	missingCapabilities := performMCPRequest(t, server, secret, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "ping",
		"params": map[string]any{"_meta": map[string]any{
			"io.modelcontextprotocol/protocolVersion": mcpModernProtocolVersion,
		}},
	}, map[string]string{
		"MCP-Protocol-Version": mcpModernProtocolVersion,
		"Mcp-Method":           "ping",
	})
	if missingCapabilities.Code != http.StatusBadRequest {
		t.Fatalf(
			"missing capabilities status = %d body = %s",
			missingCapabilities.Code,
			missingCapabilities.Body,
		)
	}
	var missingPayload struct {
		Error struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(missingCapabilities.Body.Bytes(), &missingPayload); err != nil {
		t.Fatal(err)
	}
	if missingPayload.Error.Code != -32602 {
		t.Fatalf("missing capabilities code = %d", missingPayload.Error.Code)
	}

	removedPing := performMCPRequest(t, server, secret, map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "ping",
		"params": map[string]any{"_meta": modernMCPMetadata()},
	}, map[string]string{
		"MCP-Protocol-Version": mcpModernProtocolVersion,
		"Mcp-Method":           "ping",
	})
	if removedPing.Code != http.StatusNotFound {
		t.Fatalf("removed modern ping status = %d body = %s", removedPing.Code, removedPing.Body)
	}

	unknown := performMCPRequest(t, server, secret, map[string]any{
		"jsonrpc": "2.0", "id": 3, "method": "resources/read",
		"params": map[string]any{
			"uri": "file:///unsupported", "_meta": modernMCPMetadata(),
		},
	}, map[string]string{
		"MCP-Protocol-Version": mcpModernProtocolVersion,
		"Mcp-Method":           "resources/read",
		"Mcp-Name":             "file:///unsupported",
	})
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown modern method status = %d body = %s", unknown.Code, unknown.Body)
	}
}

func TestMCPRejectsTrailingJSONMessages(t *testing.T) {
	server, secret := mcpProtocolTestServer(t)
	request := httptest.NewRequest(
		http.MethodPost,
		"/mcp",
		bytes.NewBufferString(
			`{"jsonrpc":"2.0","id":1,"method":"ping"}`+
				`{"jsonrpc":"2.0","id":2,"method":"ping"}`,
		),
	)
	request.Header.Set("Authorization", "Bearer "+secret)
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("trailing JSON status = %d body = %s", response.Code, response.Body)
	}
	var payload struct {
		Error struct {
			Code int `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Error.Code != -32700 {
		t.Fatalf("trailing JSON error code = %d", payload.Error.Code)
	}
}

func TestMCPDistinguishesParseErrorsFromInvalidRequests(t *testing.T) {
	server, secret := mcpProtocolTestServer(t)
	tests := []struct {
		name string
		body string
		code int
	}{
		{"malformed JSON", `{"jsonrpc":"2.0"`, -32700},
		{"multiple JSON values", `{} {}`, -32700},
		{"unknown member", `{"jsonrpc":"2.0","id":1,"method":"ping","unknown":true}`, -32600},
		{"wrong jsonrpc version", `{"jsonrpc":"1.0","id":1,"method":"ping"}`, -32600},
		{"missing method", `{"jsonrpc":"2.0","id":1}`, -32600},
		{"wrong method type", `{"jsonrpc":"2.0","id":1,"method":1}`, -32600},
		{"top level array", `[]`, -32600},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			response := performRawMCPRequest(t, server, secret, test.body, nil)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d body = %s", response.Code, response.Body)
			}
			var payload struct {
				Error struct {
					Code int `json:"code"`
				} `json:"error"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
				t.Fatal(err)
			}
			if payload.Error.Code != test.code {
				t.Fatalf("error code = %d, want %d", payload.Error.Code, test.code)
			}
		})
	}
}

func mcpProtocolTestServer(t *testing.T) (*Server, string) {
	t.Helper()
	runtime := storagetest.Open(t)
	t.Cleanup(func() { _ = runtime.Close() })
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)
	created := performAuthenticatedJSON(
		t, server, http.MethodPost, "/api/v1/admin/api-keys",
		map[string]any{
			"name":   "mcp-protocol-test",
			"scopes": []string{"mcp.access", "assets.read"},
		},
		cookie, csrf,
	)
	if created.Code != http.StatusCreated {
		t.Fatalf("create API key status = %d body = %s", created.Code, created.Body)
	}
	var payload struct {
		Secret string `json:"secret"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Secret == "" {
		t.Fatal("create API key response omitted secret")
	}
	return server, payload.Secret
}

func modernMCPMetadata() map[string]any {
	return map[string]any{
		"io.modelcontextprotocol/protocolVersion": mcpModernProtocolVersion,
		"io.modelcontextprotocol/clientInfo": map[string]string{
			"name": "invenqor-protocol-test", "version": "1.0.0",
		},
		"io.modelcontextprotocol/clientCapabilities": map[string]any{},
	}
}

func performMCPRequest(
	t *testing.T,
	server *Server,
	secret string,
	body any,
	headers map[string]string,
) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	return performRawMCPRequest(t, server, secret, string(encoded), headers)
}

func performRawMCPRequest(
	t *testing.T,
	server *Server,
	secret string,
	body string,
	headers map[string]string,
) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewBufferString(body))
	request.Header.Set("Authorization", "Bearer "+secret)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Accept", "application/json, text/event-stream")
	for name, value := range headers {
		request.Header.Set(name, value)
	}
	response := httptest.NewRecorder()
	server.Handler().ServeHTTP(response, request)
	return response
}

func decodeMCPResult(t *testing.T, response *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var payload struct {
		Result map[string]any `json:"result"`
	}
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload.Result == nil {
		t.Fatalf("MCP response omitted result: %s", response.Body)
	}
	return payload.Result
}

func assertModernMCPServerInfo(t *testing.T, result map[string]any) {
	t.Helper()
	meta, ok := result["_meta"].(map[string]any)
	if !ok {
		t.Fatalf("modern result omitted _meta: %#v", result)
	}
	serverInfo, ok := meta["io.modelcontextprotocol/serverInfo"].(map[string]any)
	if !ok || serverInfo["name"] != "invenqor" || serverInfo["version"] != version.Version {
		t.Fatalf("modern serverInfo = %#v", meta["io.modelcontextprotocol/serverInfo"])
	}
}
