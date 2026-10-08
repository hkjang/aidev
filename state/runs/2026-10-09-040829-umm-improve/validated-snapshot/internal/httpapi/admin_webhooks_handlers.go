package httpapi

import (
	"errors"
	"net/http"
	"net/url"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

// destination is enough of a webhook's URL to recognise it, and no more.
//
// The URL is often the credential itself — a Slack or Discord incoming hook is
// a secret in the shape of an address — so handing the whole thing to every
// administrator would be giving away the thing the webhook is protected by.
// The host says where deliveries are going, which is what someone looking at a
// failing webhook needs; the path stays with its owner.
func destination(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" {
		return "(주소를 읽을 수 없음)"
	}
	if parsed.Path == "" || parsed.Path == "/" {
		return parsed.Scheme + "://" + parsed.Host
	}
	return parsed.Scheme + "://" + parsed.Host + "/…"
}

// adminWebhooks is every webhook in the installation and how it is faring.
//
// The metrics screen carried one number: how many deliveries failed in the last
// day. Which webhook, whose, and failing with what were all recorded and none
// of them were shown, so the number told an administrator that something was
// wrong and nothing about where to look.
func (s *Server) adminWebhooks(w http.ResponseWriter, r *http.Request) {
	onlyFailing := r.URL.Query().Get("failing") == "true"
	clause := ""
	if onlyFailing {
		clause = " WHERE ws.failure_count > 0 OR ws.last_error <> ''"
	}
	rows, err := s.Store.Pool.Query(r.Context(), `
		SELECT ws.id, ws.name, ws.url, ws.active, ws.failure_count, ws.last_error, ws.last_delivered_at,
		       u.username, u.active,
		       (SELECT count(*) FROM webhook_deliveries d
		          WHERE d.subscription_id=ws.id AND d.status='failed' AND d.attempted_at >= now()-interval '24 hours'),
		       (SELECT count(*) FROM webhook_deliveries d
		          WHERE d.subscription_id=ws.id AND d.status IN ('queued','processing'))
		FROM webhook_subscriptions ws JOIN users u ON u.id=ws.owner_id`+clause+`
		ORDER BY ws.failure_count DESC, ws.updated_at DESC, ws.id`)
	if err != nil {
		writeError(w, 500, "웹훅 목록을 불러오지 못했습니다.")
		return
	}
	defer rows.Close()
	out := []map[string]any{}
	for rows.Next() {
		var id uuid.UUID
		var name, raw, lastError, owner string
		var active, ownerActive bool
		var failures int
		var lastDelivered any
		var failed24h, waiting int64
		if err := rows.Scan(&id, &name, &raw, &active, &failures, &lastError, &lastDelivered, &owner, &ownerActive, &failed24h, &waiting); err != nil {
			writeError(w, 500, "웹훅 목록을 읽지 못했습니다.")
			return
		}
		// The reason it failed, kept short enough to read in a table without
		// losing what it says.
		if len([]rune(lastError)) > 300 {
			lastError = string([]rune(lastError)[:300]) + "…"
		}
		out = append(out, map[string]any{
			"id": id, "name": name, "destination": destination(raw), "active": active,
			"failureCount": failures, "lastError": lastError, "lastDeliveredAt": lastDelivered,
			"owner": owner, "ownerActive": ownerActive,
			"failed24h": failed24h, "waiting": waiting,
		})
	}
	if err := rows.Err(); err != nil {
		writeError(w, 500, "웹훅 목록을 불러오지 못했습니다.")
		return
	}
	writeJSON(w, 200, map[string]any{"webhooks": out})
}

// pauseWebhook stops a webhook that is failing, without destroying it.
//
// Deleting belongs to whoever set it up: it takes the address, the secret and
// the choice of events with it, and an administrator putting out a fire should
// not also throw away someone's configuration. Pausing stops the deliveries and
// leaves everything else where the owner can see it and turn it back on.
func (s *Server) pauseWebhook(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "webhookID")
	if !ok {
		return
	}
	var owner uuid.UUID
	var name string
	err := s.Store.Pool.QueryRow(r.Context(),
		`UPDATE webhook_subscriptions SET active=false, updated_at=now() WHERE id=$1 AND active RETURNING owner_id, name`, id).
		Scan(&owner, &name)
	if errors.Is(err, pgx.ErrNoRows) {
		// Either it is not there or it is already paused. Both mean there is
		// nothing to stop, and neither is a failure worth alarming anyone with.
		var exists bool
		if lookupErr := s.Store.Pool.QueryRow(r.Context(), `SELECT EXISTS(SELECT 1 FROM webhook_subscriptions WHERE id=$1)`, id).Scan(&exists); lookupErr == nil && exists {
			writeJSON(w, 200, map[string]any{"active": false, "alreadyPaused": true})
			return
		}
		writeError(w, 404, "웹훅을 찾을 수 없습니다.")
		return
	}
	if err != nil {
		writeError(w, 500, "웹훅을 멈추지 못했습니다.")
		return
	}
	p := principal(r)
	s.Store.Audit(r.Context(), &p.User.ID, "webhook.pause", "webhook", id.String(),
		map[string]any{"owner": owner.String(), "name": name})
	writeJSON(w, 200, map[string]any{"active": false, "alreadyPaused": false})
}
