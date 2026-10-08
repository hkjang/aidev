package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/presentation"
	"github.com/hkjang/umm/internal/store"
	"github.com/hkjang/umm/internal/textutil"
)

/*
Sending a space to another service.

A thought starts on umm's canvas, becomes a document in muni, slides in ptium,
a report in weekly — and at every step somebody downloaded a file and uploaded
it again. The formats already fit. What was missing was the hand.

umm is the sending side of the in-house handoff standard, and only that: it
hands over Markdown and takes nothing in. The shape here is the standard's and
is not umm's to vary — six services have to meet on it.

  POST /handoff/claims          issue a claim for one space (signed in)
  GET  /handoff/claims/{claim}  collect the document (the claim is the credential)

The receiving service is opened in the browser at
<origin>/handoff?source=<umm>&claim=<claim>, fetches the document from umm
with the claim, and the claim is spent.
*/

// handoffConfig is the `handoff` settings section: where a space may be sent.
//
// Seeded empty. The menu that sends a space somewhere does not appear until
// an administrator names a service, so a new installation is unchanged.
type handoffConfig struct {
	Targets []handoffTarget `json:"targets"`
}

// handoffTarget is one service an administrator has named. Formats is what
// that service receives, in the standard's words, so that umm offers it only
// when it can take what umm sends.
type handoffTarget struct {
	Name    string   `json:"name"`
	Origin  string   `json:"origin"`
	Formats []string `json:"formats"`
}

// handoffFormats is the standard's vocabulary of what travels between the
// services. Kept whole rather than just umm's one so an administrator can
// write down what a target receives and be told when it is not a word.
var handoffFormats = []string{"markdown", "docx", "csv", "xlsx", "txt", "pptx"}

// maxHandoffTargets bounds the list. Six services exist.
const maxHandoffTargets = 20

// validateHandoffSettings is the rule for the section as it arrives from the
// settings screen: every target has a name, an origin that is exactly a
// scheme and host, and formats from the vocabulary; no origin twice.
func validateHandoffSettings(v map[string]any) error {
	rawTargets, present := v["targets"]
	if !present {
		return errors.New("보낼 곳 목록(targets)이 필요합니다")
	}
	targets, ok := rawTargets.([]any)
	if !ok {
		return errors.New("보낼 곳 목록은 배열이어야 합니다")
	}
	if len(targets) > maxHandoffTargets {
		return fmt.Errorf("보낼 곳은 %d개까지 둘 수 있습니다", maxHandoffTargets)
	}
	seen := map[string]bool{}
	for index, raw := range targets {
		target, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("%d번째 보낼 곳의 형식이 올바르지 않습니다", index+1)
		}
		name := strings.TrimSpace(fmt.Sprint(target["name"]))
		if name == "" || target["name"] == nil {
			return fmt.Errorf("%d번째 보낼 곳의 이름이 필요합니다", index+1)
		}
		origin, err := parseBrowserOrigin(fmt.Sprint(target["origin"]))
		if err != nil || target["origin"] == nil {
			return fmt.Errorf("%s의 주소는 http(s)://호스트 형태의 오리진이어야 합니다 (경로 없이)", name)
		}
		key := strings.ToLower(origin.Scheme + "://" + origin.Host)
		if seen[key] {
			return fmt.Errorf("같은 주소가 두 번 있습니다: %s", key)
		}
		seen[key] = true
		formats, ok := target["formats"].([]any)
		if !ok || len(formats) == 0 {
			return fmt.Errorf("%s이(가) 받는 형식을 하나 이상 골라 주세요", name)
		}
		for _, rawFormat := range formats {
			format, _ := rawFormat.(string)
			if !slices.Contains(handoffFormats, format) {
				return fmt.Errorf("%s의 형식 %q은(는) 표준에 없는 형식입니다", name, rawFormat)
			}
		}
	}
	return nil
}

// handoffTargetsFor reads the section and keeps the targets that can receive
// what umm sends. A target that takes only pptx is a real service and a real
// entry, and still not somewhere a Markdown document can go.
func (s *Server) handoffTargetsFor(r *http.Request) []handoffTarget {
	var cfg handoffConfig
	_ = s.Store.GetSetting(r.Context(), "handoff", &cfg)
	out := []handoffTarget{}
	for _, target := range cfg.Targets {
		if !slices.Contains(target.Formats, store.HandoffFormat) {
			continue
		}
		origin, err := parseBrowserOrigin(target.Origin)
		if err != nil {
			continue
		}
		out = append(out, handoffTarget{Name: strings.TrimSpace(target.Name), Origin: origin.Scheme + "://" + origin.Host, Formats: target.Formats})
	}
	return out
}

// handoffSource is the address the receiving service will fetch the claim
// from: the one an administrator wrote down as the public URL, or, before
// that is set, the one this request arrived at. The receiving side checks it
// against its own list either way, so a wrong value can only fail closed.
func (s *Server) handoffSource(r *http.Request) string {
	var general struct {
		PublicURL string `json:"public_url"`
	}
	_ = s.Store.GetSetting(r.Context(), "general", &general)
	if configured, err := url.Parse(strings.TrimSpace(general.PublicURL)); err == nil && configured.Host != "" && (configured.Scheme == "http" || configured.Scheme == "https") {
		return configured.Scheme + "://" + configured.Host
	}
	return effectiveRequestScheme(r) + "://" + r.Host
}

// handoffTargets answers the canvas: where can this be sent, and as whom.
func (s *Server) handoffTargets(w http.ResponseWriter, r *http.Request) {
	targets := s.handoffTargetsFor(r)
	out := make([]map[string]string, 0, len(targets))
	for _, target := range targets {
		out = append(out, map[string]string{"name": target.Name, "origin": target.Origin})
	}
	writeJSON(w, http.StatusOK, map[string]any{"targets": out, "source": s.handoffSource(r), "format": store.HandoffFormat})
}

// handoffFilename is the name the document travels under. The same characters
// are dropped as from any download name umm gives, and for the same reason: a
// name that labels on one screen becomes a path the moment another service
// writes it to disk.
func handoffFilename(spaceName string) string {
	stem := textutil.LimitUTF8Bytes(dispositionSafe(oneLine(spaceName)), maxDispositionStemBytes)
	if stem == "" {
		stem = "umm"
	}
	return stem + ".md"
}

// issueHandoffClaim renders the space and issues a claim for it.
//
// The claim is bound to what this person may read: the space is checked the
// way every export checks it, through the same approval gate, and the
// document is rendered as them. Making a claim widens nothing.
func (s *Server) issueHandoffClaim(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:read") {
		return
	}
	var req struct {
		Resource string `json:"resource"`
		Format   string `json:"format"`
	}
	if err := decodeJSON(w, r, &req); err != nil {
		writeError(w, http.StatusBadRequest, "요청 형식이 올바르지 않습니다.")
		return
	}
	if req.Format != store.HandoffFormat {
		writeError(w, http.StatusBadRequest, "umm은 markdown 형식으로만 보낼 수 있습니다.")
		return
	}
	spaceID, err := uuid.Parse(strings.TrimSpace(req.Resource))
	if err != nil {
		writeError(w, http.StatusBadRequest, "resource는 공간 id여야 합니다.")
		return
	}
	p := principal(r)
	if !s.Store.CanViewSpace(r.Context(), p.User.ID, spaceID) {
		writeError(w, http.StatusNotFound, "공간을 찾을 수 없습니다.")
		return
	}
	if !s.exportAllowed(r, spaceID, p.User.ID) {
		writeJSON(w, http.StatusConflict, map[string]any{"error": "팀장 승인이 필요합니다.", "code": "approval_required"})
		return
	}
	var spaceName string
	if err := s.Store.Pool.QueryRow(r.Context(), `SELECT name FROM spaces WHERE id=$1`, spaceID).Scan(&spaceName); err != nil {
		writeError(w, http.StatusNotFound, "공간을 찾을 수 없습니다.")
		return
	}
	// The outline, not the backup: a document or a deck starts from the
	// person's sentences in order, not from ids and canvas coordinates.
	outline, err := s.presentations(r).Outline(r.Context(), p.User.ID, presentation.Request{SpaceID: spaceID})
	if err != nil {
		if errors.Is(err, presentation.ErrNothingToPresent) {
			writeError(w, http.StatusBadRequest, "보낼 생각이 없습니다.")
			return
		}
		writePresentationError(w, r, err, "보낼 문서를 만들지 못했습니다.")
		return
	}
	body := []byte(outline)
	doc := store.HandoffDocument{
		SpaceID:     spaceID,
		IssuedBy:    p.User.ID,
		Filename:    handoffFilename(spaceName),
		ContentType: "text/markdown; charset=utf-8",
		Body:        body,
	}
	claim, expires, err := s.Store.IssueHandoffClaim(r.Context(), doc)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "표를 발급하지 못했습니다.")
		return
	}
	// The audit row says a space left, as every export's does. It never holds
	// the claim: a token in a log is a token.
	s.Store.Audit(r.Context(), &p.User.ID, "space.export", "space", spaceID.String(), map[string]any{"format": "handoff", "bytes": len(body)})
	writeJSON(w, http.StatusCreated, map[string]any{
		"claim":        claim,
		"source":       s.handoffSource(r),
		"filename":     doc.Filename,
		"content_type": doc.ContentType,
		"bytes":        len(body),
		"expires_at":   expires.Format(time.RFC3339),
	})
}

// redeemHandoffClaim hands the document to whoever brings the claim.
//
// No sign-in: the claim is the credential, which is why it is short, single
// use and bound to one document. Spent, expired and never issued all answer
// 404 alike.
func (s *Server) redeemHandoffClaim(w http.ResponseWriter, r *http.Request) {
	doc, err := s.Store.RedeemHandoffClaim(r.Context(), chiParam(r, "claim"))
	if err != nil {
		if errors.Is(err, store.ErrHandoffClaimNotFound) {
			writeError(w, http.StatusNotFound, "표를 찾을 수 없습니다.")
			return
		}
		writeError(w, http.StatusInternalServerError, "표를 확인하지 못했습니다.")
		return
	}
	w.Header().Set("Content-Type", doc.ContentType)
	w.Header().Set("Content-Disposition", attachmentDisposition(strings.TrimSuffix(doc.Filename, ".md"), ".md"))
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(doc.Body)
}

// handoffClaimRoute is the one path that carries a credential in it.
const handoffClaimRoute = "/api/v1/handoff/claims/"

// loggedPath is the path as the access log may print it. A claim is a
// credential for five minutes, and a log line lives longer than that.
func loggedPath(path string) string {
	if strings.HasPrefix(path, handoffClaimRoute) && len(path) > len(handoffClaimRoute) {
		return handoffClaimRoute + "{claim}"
	}
	return path
}
