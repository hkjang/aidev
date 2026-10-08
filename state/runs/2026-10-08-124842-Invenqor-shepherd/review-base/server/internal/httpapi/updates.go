package httpapi

import (
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/hkjang/invenqor/server/internal/agents"
	"github.com/hkjang/invenqor/server/internal/updates"
)

func (s *Server) agentUpdateManifest(w http.ResponseWriter, r *http.Request) {
	agent, err := s.authenticateAgent(r)
	if errors.Is(err, agents.ErrUnauthorized) {
		writeAPIError(w, r, 401, "AGENT_UNAUTHORIZED", "The agent credential is invalid.")
		return
	}
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	if s.updateStore == nil {
		writeAPIError(w, r, 503, "UPDATE_STORE_UNAVAILABLE", "The update store is unavailable.")
		return
	}
	if r.URL.Query().Get("agent_id") != agent.AgentID {
		writeAPIError(w, r, 403, "AGENT_ID_MISMATCH", "The agent identity does not match.")
		return
	}
	candidates, err := s.updateStore.Candidates(
		r.URL.Query().Get("channel"),
		r.URL.Query().Get("os"),
		r.URL.Query().Get("arch"),
		agent.AgentID,
	)
	if err != nil {
		s.internalError(w, r, err)
		return
	}
	current := r.URL.Query().Get("current_version")
	manifest := selectAgentUpdateOffer(candidates, current)
	if manifest == nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	writeJSON(w, 200, manifest)
}

func selectAgentUpdateOffer(
	candidates []updates.Manifest,
	current string,
) *updates.Manifest {
	for index := range candidates {
		if agentUpdateOfferAllowed(&candidates[index], current) {
			return &candidates[index]
		}
	}
	return nil
}

func (s *Server) agentUpdateArtifact(w http.ResponseWriter, r *http.Request) {
	if _, err := s.authenticateAgent(r); err != nil {
		writeAPIError(w, r, 401, "AGENT_UNAUTHORIZED", "The agent credential is invalid.")
		return
	}
	if s.updateStore == nil {
		writeAPIError(w, r, 503, "UPDATE_STORE_UNAVAILABLE", "The update store is unavailable.")
		return
	}
	path, err := s.updateStore.Artifact(chi.URLParam(r, "artifact"))
	if err != nil {
		writeAPIError(w, r, 404, "UPDATE_NOT_FOUND", "The update artifact does not exist.")
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	w.Header().Set("Cache-Control", "private, max-age=300")
	http.ServeFile(w, r, path)
}

func (s *Server) publishAgentUpdate(w http.ResponseWriter, r *http.Request) {
	if s.updateStore == nil {
		writeAPIError(w, r, 503, "UPDATE_STORE_UNAVAILABLE", "The update store is unavailable.")
		return
	}
	// New v2 publications carry both a bridge artifact signature and a
	// metadata-bound manifest signature. Accepting either without checking it
	// would only defer a bad publication to every Agent in the fleet, so the
	// administrative publish API is unavailable until the pinned public key is
	// configured. Existing stored releases remain readable for migration.
	if !s.updateStore.SigningKeyConfigured() {
		writeAPIError(
			w, r, http.StatusServiceUnavailable,
			"UPDATE_SIGNING_KEY_MISSING",
			"INVENQOR_UPDATE_PUBLIC_KEY is required to publish Agent updates.",
		)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 130*1024*1024)
	if err := r.ParseMultipartForm(130 * 1024 * 1024); err != nil {
		writeAPIError(w, r, 400, "INVALID_UPDATE", "The multipart update is invalid.")
		return
	}
	file, _, err := r.FormFile("artifact")
	if err != nil {
		writeAPIError(w, r, 400, "MISSING_ARTIFACT", "The artifact file is required.")
		return
	}
	defer file.Close()
	options, err := publishOptions(r)
	if err != nil {
		writeAPIError(w, r, 400, "INVALID_UPDATE", err.Error())
		return
	}
	manifest, err := s.updateStore.Publish(updates.Manifest{
		Version:           options.Version,
		Channel:           options.Channel,
		OS:                options.OS,
		Architecture:      options.Architecture,
		SHA256:            options.SHA256,
		Size:              options.Size,
		Signature:         options.Signature,
		ManifestSignature: options.ManifestSignature,
		SignatureScheme:   updates.SignatureSchemeEd25519,
		SignatureVersion:  updates.SignatureVersionV2,
		Rollout:           options.Rollout,
		AllowDowngrade:    options.AllowDowngrade,
		Notes:             options.Notes,
		PublishedBy:       principalFromContext(r.Context()).User.Username,
	}, io.LimitReader(file, 128*1024*1024+1))
	if err != nil {
		// A rejected signature is the one failure worth its own code: it means the
		// operator signed the wrong file or pasted the wrong value, and every
		// agent would otherwise have failed silently.
		code := "INVALID_UPDATE"
		status := 400
		switch {
		case errors.Is(err, updates.ErrSignatureRejected):
			code = "UPDATE_SIGNATURE_REJECTED"
		case errors.Is(err, updates.ErrSignatureUnverifiable):
			code = "UPDATE_SIGNING_KEY_MISSING"
			status = 503
		}
		writeAPIError(w, r, status, code, err.Error())
		return
	}
	s.recordAdminAudit(r, "agent_update.publish", "agent_update",
		manifest.Version+"-"+manifest.OS+"-"+manifest.Architecture,
		nil, manifest, options.Notes)
	writeJSON(w, 201, manifest)
}

func updateVersionParts(value string) ([3]uint64, bool) {
	var result [3]uint64
	pieces := strings.Split(strings.TrimPrefix(value, "v"), ".")
	if len(pieces) != len(result) {
		return result, false
	}
	for index, piece := range pieces {
		if piece == "" || strings.IndexFunc(piece, func(char rune) bool {
			return char < '0' || char > '9'
		}) >= 0 {
			return result, false
		}
		parsed, err := strconv.ParseUint(piece, 10, 64)
		if err != nil {
			return result, false
		}
		result[index] = parsed
	}
	return result, true
}

func agentUpdateOfferAllowed(manifest *updates.Manifest, current string) bool {
	if manifest == nil {
		return false
	}
	currentParts, currentOK := updateVersionParts(current)
	candidateParts, candidateOK := updateVersionParts(manifest.Version)
	if !currentOK || !candidateOK {
		return false
	}
	comparison := compareUpdateVersionParts(candidateParts, currentParts)
	if manifest.AllowDowngrade {
		// Artifact-only signatures do not authenticate allow_downgrade. Never
		// send an old stored v1 rollback to any Agent. A v2 rollback is safe only
		// for an Agent that knows to verify manifest_signature; v0.2.14 and
		// earlier would trust only the bridge artifact signature and unsigned
		// rollback flag. Apply this gate even when the target happens to be newer
		// than an old Agent: an explicitly rollback-marked release is never part
		// of the legacy bridge path.
		if manifest.SignatureVersion < updates.SignatureVersionV2 || comparison == 0 {
			return false
		}
		minimumV2, _ := updateVersionParts("0.2.15")
		return compareUpdateVersionParts(currentParts, minimumV2) >= 0
	}
	return comparison > 0
}

func compareUpdateVersionParts(left, right [3]uint64) int {
	for index := range left {
		switch {
		case left[index] < right[index]:
			return -1
		case left[index] > right[index]:
			return 1
		}
	}
	return 0
}
