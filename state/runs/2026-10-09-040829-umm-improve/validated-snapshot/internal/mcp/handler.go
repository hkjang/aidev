package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/auth"
	"github.com/hkjang/umm/internal/dream"
	"github.com/hkjang/umm/internal/presentation"
	"github.com/hkjang/umm/internal/store"
)

const CurrentProtocol = "2026-07-28"

type Handler struct {
	Store  *store.Store
	Dreams *dream.Service
	// Cipher unwraps the stored Ptium credential. Without it the bridge would
	// send ciphertext as a bearer token and Ptium would answer 401.
	Cipher presentation.Decrypter
	// Tokens accepts a Keycloak access token in place of a key, once an
	// administrator has turned that on. Nil means keys only.
	Tokens  AccessTokens
	Version string
}

// AccessTokens is the OAuth half of the door: it verifies a token the auth
// middleware did not recognise as a key, and describes this server to a client
// that has none yet.
type AccessTokens interface {
	AuthenticateAccessToken(ctx context.Context, raw string) (auth.Principal, error)
	MCPResource(ctx context.Context, toolScopes []string) (auth.ProtectedResource, bool)
}

// toolScopes are the scopes the tools below ask for, advertised to a client
// choosing what to request at sign-in. Sorted, like the tools.
var toolScopes = []string{"dreams:read", "notes:read", "notes:write", "spaces:read"}

// keyPrefix is how every key umm has ever issued begins. A bearer value
// without it is not a key, and the only other thing it can be is a token.
const keyPrefix = "umm_key_"

// resource reports whether Keycloak sign-in is on for MCP and, if so, how a
// client is told about it.
func (h *Handler) resource(ctx context.Context) (auth.ProtectedResource, bool) {
	if h.Tokens == nil {
		return auth.ProtectedResource{}, false
	}
	return h.Tokens.MCPResource(ctx, toolScopes)
}

// ServeResourceMetadata answers RFC 9728's well-known document for /mcp, and
// 404 while Keycloak sign-in for MCP is off: a client must not be sent to an
// authorization server whose tokens would then be refused.
func (h *Handler) ServeResourceMetadata(w http.ResponseWriter, r *http.Request) {
	resource, on := h.resource(r.Context())
	if !on {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache")
	_ = json.NewEncoder(w).Encode(resource)
}

// authenticate finds who is calling. The middleware has already recognised a
// key or a session; a bearer value it did not recognise is handed to Keycloak
// verification, when that is on. A session is refused either way: MCP takes
// a credential with scopes on it, and a browser cookie has all of them.
func (h *Handler) authenticate(r *http.Request, resource auth.ProtectedResource, oauthOn bool) (auth.Principal, bool) {
	p, ok := auth.PrincipalFrom(r.Context())
	if ok {
		return p, p.AuthType == "api_key" || p.AuthType == auth.AuthTypeOAuth
	}
	if !oauthOn {
		return auth.Principal{}, false
	}
	header := r.Header.Get("Authorization")
	raw := strings.TrimSpace(strings.TrimPrefix(header, "Bearer "))
	if raw == "" || raw == header || strings.HasPrefix(raw, keyPrefix) {
		return auth.Principal{}, false
	}
	p, err := h.Tokens.AuthenticateAccessToken(r.Context(), raw)
	if err != nil {
		return auth.Principal{}, false
	}
	return p, true
}

// unauthorized is the refusal at the door. With Keycloak sign-in on it carries
// the challenge an MCP client follows to sign in, naming the metadata document
// and, when a token was presented and refused, saying so — and no more than
// so. Without it the answer is the one keys-only clients have always read.
func unauthorized(w http.ResponseWriter, resource auth.ProtectedResource, oauthOn, tokenPresented bool) {
	message := "Bearer API key required"
	if oauthOn {
		message = "Bearer API key or Keycloak access token required"
		challenge := `Bearer realm="umm", resource_metadata="` + resource.MetadataURL + `"`
		if tokenPresented {
			challenge += `, error="invalid_token"`
		}
		w.Header().Set("WWW-Authenticate", challenge)
	}
	w.WriteHeader(http.StatusUnauthorized)
	writeRPC(w, response{JSONRPC: "2.0", Error: &rpcError{Code: -32001, Message: message}})
}

type request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params"`
}
type callParams struct {
	Name      string         `json:"name"`
	Arguments map[string]any `json:"arguments"`
}
type response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  any             `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}
type rpcError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("MCP-Protocol-Version", CurrentProtocol)
	if r.Method != http.MethodPost {
		writeRPC(w, response{JSONRPC: "2.0", Error: &rpcError{Code: -32600, Message: "POST required"}})
		return
	}
	resource, oauthOn := h.resource(r.Context())
	p, ok := h.authenticate(r, resource, oauthOn)
	if !ok {
		_, hadPrincipal := auth.PrincipalFrom(r.Context())
		unauthorized(w, resource, oauthOn, !hadPrincipal && r.Header.Get("Authorization") != "")
		return
	}
	if origin := r.Header.Get("Origin"); origin != "" && !h.validOrigin(r, origin) {
		w.WriteHeader(http.StatusForbidden)
		writeRPC(w, response{JSONRPC: "2.0", Error: &rpcError{Code: -32002, Message: "invalid Origin"}})
		return
	}
	var req request
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&req); err != nil {
		writeRPC(w, response{JSONRPC: "2.0", Error: &rpcError{Code: -32700, Message: "parse error"}})
		return
	}
	if req.JSONRPC != "2.0" {
		writeRPC(w, response{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32600, Message: "invalid JSON-RPC version"}})
		return
	}
	protocol := r.Header.Get("MCP-Protocol-Version")
	if protocol == CurrentProtocol {
		if r.Header.Get("Mcp-Method") != req.Method {
			writeRPC(w, response{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32600, Message: "Mcp-Method header mismatch"}})
			return
		}
	}
	var result any
	var err error
	switch req.Method {
	case "server/discover":
		result = map[string]any{"protocolVersion": CurrentProtocol, "serverInfo": map[string]string{"name": "umm", "version": h.Version}, "capabilities": map[string]any{"tools": map[string]any{"listChanged": false}}}
	case "initialize":
		result = map[string]any{"protocolVersion": "2025-11-25", "serverInfo": map[string]string{"name": "umm", "version": h.Version}, "capabilities": map[string]any{"tools": map[string]any{"listChanged": false}}}
	case "notifications/initialized":
		w.WriteHeader(http.StatusAccepted)
		return
	case "ping":
		result = map[string]any{}
	case "tools/list":
		result = map[string]any{"tools": toolDefinitions()}
	case "tools/call":
		var params callParams
		if json.Unmarshal(req.Params, &params) != nil {
			err = errors.New("invalid tool arguments")
		} else if protocol == CurrentProtocol && r.Header.Get("Mcp-Name") != params.Name {
			err = errors.New("Mcp-Name header mismatch")
		} else {
			result, err = h.call(r, p, params)
		}
	default:
		writeRPC(w, response{JSONRPC: "2.0", ID: req.ID, Error: &rpcError{Code: -32601, Message: "method not found"}})
		return
	}
	if err != nil {
		writeRPC(w, response{JSONRPC: "2.0", ID: req.ID, Result: map[string]any{"content": []map[string]string{{"type": "text", "text": err.Error()}}, "isError": true}})
		return
	}
	writeRPC(w, response{JSONRPC: "2.0", ID: req.ID, Result: result})
}

func (h *Handler) validOrigin(r *http.Request, origin string) bool {
	var general struct {
		PublicURL string `json:"public_url"`
	}
	if h.Store.GetSetting(r.Context(), "general", &general) != nil {
		return false
	}
	a, err1 := url.Parse(origin)
	b, err2 := url.Parse(general.PublicURL)
	return err1 == nil && err2 == nil && strings.EqualFold(a.Scheme, b.Scheme) && strings.EqualFold(a.Host, b.Host)
}

func toolDefinitions() []map[string]any {
	return []map[string]any{
		{"name": "capture_thought", "title": "Capture a thought", "description": "Write a thought down without deciding where it belongs. It lands in the owner's inbox, ready to be filed later. Use this when you have something worth keeping but no reason to choose a space — create_note is for when you do.", "inputSchema": schema(map[string]any{"content": str("Thought text")}, "content")},
		{"name": "connect_notes", "title": "Connect thoughts", "description": "Connect two notes in the same space. The connection is recorded as written by an agent, which the owner can see; it cannot be presented as one they drew themselves.", "inputSchema": schema(map[string]any{"space_id": str("Space UUID"), "source_note_id": str("Source note UUID"), "target_note_id": str("Target note UUID"), "relation": str("What the connection asserts: related (default), supports, contradicts, refines, expands or follows. Anything else is rejected.")}, "space_id", "source_note_id", "target_note_id")},
		{"name": "create_note", "title": "Drop a thought", "description": "Create a post-it in a space.", "inputSchema": schema(map[string]any{"space_id": str("Space UUID"), "content": str("Thought text"), "x": num("Canvas x"), "y": num("Canvas y"), "color": str("Semantic color")}, "space_id", "content")},
		{"name": "find_contradictions", "title": "Find recorded disagreements", "description": "List thoughts that were recorded as contradicting each other. These are connections a person or an agent drew, not something umm inferred: an empty result means nobody has marked any disagreement, not that the workspace is free of them. Pass space_id to narrow to one space.", "inputSchema": schema(map[string]any{"space_id": str("Space UUID; omit for every space you can read")})},
		{"name": "find_open_questions", "title": "Find unanswered questions", "description": "List thoughts marked as questions that nothing has been recorded as answering. Both halves are marked by a person or an agent, not inferred: umm does not read a note and decide it is asking something. An empty result means nothing is marked open, not that everything has been answered. Pass space_id to narrow to one space.", "inputSchema": schema(map[string]any{"space_id": str("Space UUID; omit for every space you can read")})},
		{"name": "get_connections", "title": "Walk the memory graph", "description": "List the connections attached to a note: what it points at, what points at it, what each connection asserts, and who made it. get_related_notes finds thoughts that merely resemble each other; this returns the connections that were actually recorded, including ones umm inferred, which are marked as such and carry a confidence.", "inputSchema": schema(map[string]any{"note_id": str("Note UUID")}, "note_id")},
		{"name": "get_related_notes", "title": "Discover related thoughts", "description": "Find related notes using the offline similarity embedding.", "inputSchema": schema(map[string]any{"note_id": str("Source note UUID")}, "note_id")},
		{"name": "list_clusters", "title": "List thought clusters", "description": "Discover coherent groups of notes in a space.", "inputSchema": schema(map[string]any{"space_id": str("Space UUID")}, "space_id")},
		{"name": "list_dreams", "title": "List Dream notes", "description": "List the user's recent Dream notes.", "inputSchema": schema(map[string]any{})},
		{"name": "list_lines", "title": "List lines of thinking", "description": "List the lines of thinking in a space: what each was called, whether it is still open, was taken or was set aside, and the reason recorded when it was resolved. Marked by a person, never inferred — umm does not decide a line was abandoned because nothing was added to it, so an empty result means nothing is marked, not that no decisions were made.", "inputSchema": schema(map[string]any{"space_id": str("Space UUID")}, "space_id")},
		{"name": "list_notes", "title": "List thoughts", "description": "List notes and connections in one space. note_lines maps a note to the line of thinking it belongs to; a note whose line is 'abandoned' was considered and set aside, with the reason recorded, and must not be treated as current.", "inputSchema": schema(map[string]any{"space_id": str("Space UUID")}, "space_id")},
		{"name": "list_presentations", "title": "List talks made from a space", "description": "List the presentations a space has produced, with how many of each deck's slides quote a thought that has since been rewritten or deleted. Moving a thought does not count: the check is on the words. A deck made before umm recorded fingerprints reports none, which means unknown rather than fresh.", "inputSchema": schema(map[string]any{"space_id": str("Space UUID")}, "space_id")},
		{"name": "list_spaces", "title": "List spaces", "description": "List spaces available to this user.", "inputSchema": schema(map[string]any{})},
		{"name": "make_presentation", "title": "Make a talk in Ptium", "description": "Compile a space into a presentation in the configured Ptium. Requires an administrator to have connected one. The thoughts reach the slides word for word; nothing here asks a model to write anything. Fails rather than making an empty deck when nothing in the space can become a talk.", "inputSchema": schema(map[string]any{"space_id": str("Space UUID"), "title": str("Title for the talk; the space's name is used when omitted"), "note_ids": map[string]any{"type": "array", "items": map[string]any{"type": "string"}, "description": "Restrict the talk to these thoughts, for building from a selection or a cluster"}, "include_excluded": map[string]any{"type": "boolean", "description": "Include thoughts the owner held back from analysis"}}, "space_id")},
		{"name": "preview_presentation", "title": "See a space as a talk", "description": "Compile a space's thoughts and connections into an ordered talk and return it, changing nothing. Slide text is the author's own sentences, never a paraphrase: this orders and groups what they wrote, it does not rewrite it. Use this before make_presentation so the person can see what their space would become.", "inputSchema": schema(map[string]any{"space_id": str("Space UUID"), "title": str("Title for the talk; the space's name is used when omitted"), "include_excluded": map[string]any{"type": "boolean", "description": "Include thoughts the owner held back from analysis. Off unless asked for: a thought held back is being held back from having things done to it."}}, "space_id")},
		{"name": "search_notes", "title": "Search thoughts", "description": "Search notes in one space. note_lines maps a note to the line of thinking it belongs to; a note whose line is 'abandoned' was considered and set aside, with the reason recorded, and must not be treated as current.", "inputSchema": schema(map[string]any{"space_id": str("Space UUID"), "query": str("Search text")}, "space_id", "query")},
	}
}
func str(description string) map[string]any {
	return map[string]any{"type": "string", "description": description}
}
func num(description string) map[string]any {
	return map[string]any{"type": "number", "description": description}
}
func schema(properties map[string]any, required ...string) map[string]any {
	return map[string]any{"type": "object", "properties": properties, "required": required, "additionalProperties": false}
}

func (h *Handler) call(r *http.Request, p auth.Principal, params callParams) (any, error) {
	ctx := r.Context()
	args := params.Arguments
	success := func(v any) (any, error) {
		raw, _ := json.Marshal(v)
		return map[string]any{"content": []map[string]string{{"type": "text", "text": string(raw)}}, "structuredContent": v, "isError": false}, nil
	}
	require := func(scope string) error {
		if p.Scopes["*"] || p.Scopes[scope] {
			return nil
		}
		if p.AuthType == auth.AuthTypeOAuth {
			return fmt.Errorf("the Keycloak token lacks the %s scope", scope)
		}
		return fmt.Errorf("API key requires %s scope", scope)
	}
	// Written into the audit row of every change, so a reader can tell a
	// key from a Keycloak sign-in.
	credential := map[string]any{"key": p.AuthType == "api_key", "auth": p.AuthType}
	spaceID := func() (uuid.UUID, error) { return uuid.Parse(fmt.Sprint(args["space_id"])) }
	// fmt.Sprint renders a missing key as the literal "<nil>", which read as a
	// value turns an omitted argument into a real one. These three read it as
	// absent instead.
	text := func(key string) string {
		value, ok := args[key]
		if !ok || value == nil {
			return ""
		}
		return strings.TrimSpace(fmt.Sprint(value))
	}
	flag := func(key string) bool {
		on, _ := args[key].(bool)
		return on
	}
	strings_ := func(key string) []string {
		raw, _ := args[key].([]any)
		out := make([]string, 0, len(raw))
		for _, item := range raw {
			if item == nil {
				continue
			}
			out = append(out, fmt.Sprint(item))
		}
		return out
	}
	switch params.Name {
	case "list_spaces":
		if err := require("spaces:read"); err != nil {
			return nil, err
		}
		v, err := h.Store.ListSpaces(ctx, p.User.ID)
		if err != nil {
			return nil, err
		}
		return success(map[string]any{"spaces": v})
	case "list_notes", "search_notes":
		if err := require("notes:read"); err != nil {
			return nil, err
		}
		sid, err := spaceID()
		if err != nil {
			return nil, errors.New("valid space_id required")
		}
		query := ""
		if params.Name == "search_notes" {
			query = fmt.Sprint(args["query"])
		}
		notes, edges, err := h.Store.ListNotes(ctx, p.User.ID, sid, query)
		if err != nil {
			return nil, err
		}
		// A thought in a line that was decided against reads exactly like a
		// current one without this, and an agent acting on it would be acting on
		// the option its owner already rejected. The web and the built-in
		// assistant were taught to say so in v0.24-v0.27; an agent reading over
		// MCP is the same reader coming through a different door.
		lines, err := h.Store.ListBranches(ctx, p.User.ID, sid)
		if err != nil {
			return nil, err
		}
		assignments, err := h.Store.BranchAssignments(ctx, p.User.ID, sid)
		if err != nil {
			return nil, err
		}
		byID := make(map[uuid.UUID]store.Branch, len(lines))
		for _, line := range lines {
			byID[line.ID] = line
		}
		noteLines := map[string]any{}
		for noteID, branchID := range assignments {
			if line, ok := byID[branchID]; ok {
				noteLines[noteID.String()] = map[string]any{
					"name": line.Name, "status": line.Status, "resolution": line.Resolution,
				}
			}
		}
		return success(map[string]any{"notes": notes, "edges": edges, "note_lines": noteLines})
	case "list_lines":
		if err := require("notes:read"); err != nil {
			return nil, err
		}
		sid, err := spaceID()
		if err != nil {
			return nil, errors.New("valid space_id required")
		}
		lines, err := h.Store.ListBranches(ctx, p.User.ID, sid)
		if err != nil {
			return nil, err
		}
		return success(map[string]any{"lines": lines})
	case "get_related_notes":
		if err := require("notes:read"); err != nil {
			return nil, err
		}
		noteID, err := uuid.Parse(fmt.Sprint(args["note_id"]))
		if err != nil {
			return nil, errors.New("valid note_id required")
		}
		related, err := h.Store.RelatedNotes(ctx, p.User.ID, noteID, 8)
		if err != nil {
			return nil, err
		}
		return success(map[string]any{"related": related})
	case "preview_presentation", "make_presentation":
		// Reading is enough to look; making a deck sends the owner's thoughts to
		// another service and creates something there, so it asks for more.
		scope := "notes:read"
		if params.Name == "make_presentation" {
			scope = "notes:write"
		}
		if err := require(scope); err != nil {
			return nil, err
		}
		sid, err := spaceID()
		if err != nil {
			return nil, errors.New("valid space_id required")
		}
		req := presentation.Request{
			SpaceID:         sid,
			Title:           strings.TrimSpace(text("title")),
			IncludeExcluded: flag("include_excluded"),
		}
		for _, raw := range strings_("note_ids") {
			id, err := uuid.Parse(strings.TrimSpace(raw))
			if err != nil {
				return nil, errors.New("note_ids must be UUIDs")
			}
			req.Only = append(req.Only, id)
		}

		svc := &presentation.Service{Spaces: h.Store, Links: h.Store, Settings: h.Store, Cipher: h.Cipher}
		if params.Name == "preview_presentation" {
			preview, err := svc.Preview(ctx, p.User.ID, req)
			if err != nil {
				return nil, err
			}
			return success(map[string]any{"storyline": preview.Storyline, "source": preview.Source, "checked": preview.Checked})
		}
		result, err := svc.Create(ctx, p.User.ID, req)
		if err != nil {
			return nil, err
		}
		return success(map[string]any{"presentation": result.Link, "warnings": result.Warnings})
	case "list_presentations":
		if err := require("notes:read"); err != nil {
			return nil, err
		}
		sid, err := spaceID()
		if err != nil {
			return nil, errors.New("valid space_id required")
		}
		links, err := h.Store.ListPresentationLinks(ctx, p.User.ID, sid)
		if err != nil {
			return nil, err
		}
		// The staleness count is what makes the list worth reading: a deck that
		// was true when it was made is not the same as one that still is.
		counts, err := h.Store.StaleCounts(ctx, p.User.ID, sid)
		if err != nil {
			return nil, err
		}
		rows := make([]map[string]any, 0, len(links))
		for _, link := range links {
			rows = append(rows, map[string]any{"presentation": link, "stale_slides": counts[link.ID]})
		}
		return success(map[string]any{"presentations": rows})
	case "list_clusters":
		if err := require("notes:read"); err != nil {
			return nil, err
		}
		sid, err := spaceID()
		if err != nil {
			return nil, errors.New("valid space_id required")
		}
		clusters, err := h.Store.Clusters(ctx, p.User.ID, sid)
		if err != nil {
			return nil, err
		}
		return success(map[string]any{"clusters": clusters})
	case "create_note":
		if err := require("notes:write"); err != nil {
			return nil, err
		}
		sid, err := spaceID()
		if err != nil {
			return nil, errors.New("valid space_id required")
		}
		content := strings.TrimSpace(fmt.Sprint(args["content"]))
		if content == "" {
			return nil, errors.New("content required")
		}
		n := store.Note{SpaceID: sid, Content: content, Source: "mcp", Color: valueString(args, "color", "yellow"), X: valueFloat(args, "x"), Y: valueFloat(args, "y")}
		n, err = h.Store.CreateNote(ctx, p.User.ID, n)
		if err != nil {
			return nil, err
		}
		h.Store.Audit(ctx, &p.User.ID, "mcp.note.create", "note", n.ID.String(), credential)
		return success(n)
	case "capture_thought":
		if err := require("notes:write"); err != nil {
			return nil, err
		}
		content := valueString(args, "content", "")
		if strings.TrimSpace(content) == "" {
			return nil, errors.New("content required")
		}
		n, err := h.Store.CaptureThought(ctx, p.User.ID, content)
		if err != nil {
			return nil, err
		}
		h.Store.Audit(ctx, &p.User.ID, "mcp.thought.capture", "note", n.ID.String(), credential)
		return success(n)
	case "connect_notes":
		if err := require("notes:write"); err != nil {
			return nil, err
		}
		sid, err := spaceID()
		if err != nil {
			return nil, errors.New("valid space_id required")
		}
		source, err := uuid.Parse(fmt.Sprint(args["source_note_id"]))
		if err != nil {
			return nil, errors.New("valid source_note_id required")
		}
		target, err := uuid.Parse(fmt.Sprint(args["target_note_id"]))
		if err != nil {
			return nil, errors.New("valid target_note_id required")
		}
		e, err := h.Store.CreateAgentEdge(ctx, p.User.ID, store.Edge{SpaceID: sid, SourceID: source, TargetID: target,
			Relation: store.Relation(valueString(args, "relation", string(store.RelationRelated)))})
		if err != nil {
			return nil, err
		}
		return success(e)
	case "find_contradictions":
		if err := require("notes:read"); err != nil {
			return nil, err
		}
		var scope *uuid.UUID
		if raw := valueString(args, "space_id", ""); strings.TrimSpace(raw) != "" {
			parsed, parseErr := uuid.Parse(raw)
			if parseErr != nil {
				return nil, errors.New("valid space_id required")
			}
			scope = &parsed
		}
		items, err := h.Store.Contradictions(ctx, p.User.ID, scope)
		if err != nil {
			return nil, err
		}
		found := make([]map[string]any, 0, len(items))
		for _, item := range items {
			found = append(found, map[string]any{
				"space":           item.Space,
				"recorded_by":     item.Origin,
				"claim_note_id":   item.Claim.ID,
				"claim":           item.Claim.Content,
				"counter_note_id": item.Counter.ID,
				"counter":         item.Counter.Content,
			})
		}
		return success(map[string]any{"contradictions": found})
	case "find_open_questions":
		if err := require("notes:read"); err != nil {
			return nil, err
		}
		var scope *uuid.UUID
		if raw := valueString(args, "space_id", ""); strings.TrimSpace(raw) != "" {
			parsed, parseErr := uuid.Parse(raw)
			if parseErr != nil {
				return nil, errors.New("valid space_id required")
			}
			scope = &parsed
		}
		questions, err := h.Store.OpenQuestions(ctx, p.User.ID, scope)
		if err != nil {
			return nil, err
		}
		open := make([]map[string]any, 0, len(questions))
		for _, item := range questions {
			open = append(open, map[string]any{
				"note_id":  item.Note.ID,
				"question": item.Note.Content,
				"space":    item.Space,
				// Connections that circle the question without settling it.
				"attempts": item.Attempts,
			})
		}
		return success(map[string]any{"open_questions": open})
	case "get_connections":
		if err := require("notes:read"); err != nil {
			return nil, err
		}
		noteID, err := uuid.Parse(fmt.Sprint(args["note_id"]))
		if err != nil {
			return nil, errors.New("valid note_id required")
		}
		links, err := h.Store.Backlinks(ctx, p.User.ID, noteID)
		if err != nil {
			return nil, err
		}
		// Flattened deliberately: an agent reading this needs the assertion and
		// its provenance next to the thought, not a nested edge object to unpack.
		connections := make([]map[string]any, 0, len(links))
		for _, link := range links {
			connection := map[string]any{
				"direction": link.Direction,
				"relation":  link.Edge.Relation,
				"origin":    link.Edge.Origin,
				"inferred":  link.Edge.Origin.Inferred(),
				"note_id":   link.Note.ID,
				"content":   link.Note.Content,
			}
			if link.Edge.Confidence != nil {
				connection["confidence"] = *link.Edge.Confidence
			}
			connections = append(connections, connection)
		}
		return success(map[string]any{"connections": connections})
	case "list_dreams":
		if err := require("dreams:read"); err != nil {
			return nil, err
		}
		v, err := h.Dreams.History(ctx, p.User.ID)
		if err != nil {
			return nil, err
		}
		return success(map[string]any{"dreams": v})
	default:
		return nil, errors.New("unknown tool")
	}
}

func valueString(m map[string]any, key, fallback string) string {
	v := strings.TrimSpace(fmt.Sprint(m[key]))
	if v == "" || v == "<nil>" {
		return fallback
	}
	return v
}
func valueFloat(m map[string]any, key string) float64 { v, _ := m[key].(float64); return v }
func writeRPC(w http.ResponseWriter, v response)      { _ = json.NewEncoder(w).Encode(v) }
