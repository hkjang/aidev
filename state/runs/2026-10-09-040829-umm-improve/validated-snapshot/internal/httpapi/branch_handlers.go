package httpapi

import (
	"errors"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/store"
)

// Lines of thinking, and what became of them.
//
// The point of recording this is not tidiness. A thought that was tried and set
// aside comes back from search looking exactly like a current one, and the cost
// of that is acting on the option you already rejected. Everything here is
// marked by a person: umm never decides a line was abandoned because nothing has
// been added to it lately.

// listBranches returns the lines in a space. Reading is allowed to anyone who
// can read the space.
func (s *Server) listBranches(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:read") {
		return
	}
	spaceID, ok := parseID(w, r, "spaceID")
	if !ok {
		return
	}
	p := principal(r)
	branches, err := s.Store.ListBranches(r.Context(), p.User.ID, spaceID)
	if err != nil {
		slog.Warn("list branches failed", "user_id", p.User.ID, "space_id", spaceID, "error", err)
		writeError(w, http.StatusInternalServerError, "갈래를 불러오지 못했습니다.")
		return
	}
	assignments, err := s.Store.BranchAssignments(r.Context(), p.User.ID, spaceID)
	if err != nil {
		slog.Warn("branch assignments failed", "user_id", p.User.ID, "space_id", spaceID, "error", err)
		writeError(w, http.StatusInternalServerError, "갈래를 불러오지 못했습니다.")
		return
	}
	// Sent as a map keyed by thought so the canvas can label a note without
	// walking every branch.
	labels := map[string]string{}
	for noteID, branchID := range assignments {
		labels[noteID.String()] = branchID.String()
	}
	writeJSON(w, http.StatusOK, map[string]any{"branches": branches, "assignments": labels})
}

func (s *Server) createBranch(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:write") {
		return
	}
	spaceID, ok := parseID(w, r, "spaceID")
	if !ok {
		return
	}
	var body struct {
		Name       string     `json:"name"`
		RootNoteID *uuid.UUID `json:"rootNoteId"`
	}
	if decodeJSON(w, r, &body) != nil || strings.TrimSpace(body.Name) == "" {
		writeError(w, http.StatusBadRequest, "갈래 이름을 입력해 주세요.")
		return
	}
	p := principal(r)
	branch, err := s.Store.CreateBranch(r.Context(), p.User.ID, spaceID, body.Name, body.RootNoteID)
	if err != nil {
		// No row came back when the person cannot edit the space, or the root
		// thought lives somewhere else. Both are the caller's mistake.
		slog.Info("create branch refused", "user_id", p.User.ID, "space_id", spaceID, "error", err)
		writeError(w, http.StatusBadRequest, "갈래를 만들 수 없습니다. 편집 권한과 뿌리 생각을 확인해 주세요.")
		return
	}
	writeJSON(w, http.StatusCreated, branch)
}

// resolveBranch records what became of a line, and why.
//
// The reason is required, and the API says so rather than storing an empty one.
// A branch marked abandoned with no reason is the same forgetting this exists to
// prevent, one step later.
func (s *Server) resolveBranch(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:write") {
		return
	}
	branchID, ok := parseID(w, r, "branchID")
	if !ok {
		return
	}
	var body struct {
		Status     string `json:"status"`
		Resolution string `json:"resolution"`
	}
	if decodeJSON(w, r, &body) != nil {
		writeError(w, http.StatusBadRequest, "요청이 올바르지 않습니다.")
		return
	}
	p := principal(r)
	branch, err := s.Store.ResolveBranch(r.Context(), p.User.ID, branchID, body.Status, body.Resolution)
	if err != nil {
		if errors.Is(err, store.ErrUnknownBranchStatus) {
			writeProblem(w, r, http.StatusBadRequest, "unknown-branch-status", "갈래 상태가 올바르지 않습니다",
				"채택했는지 접어 두었는지를 골라 주세요.",
				map[string]any{"allowedStatuses": []string{store.BranchAdopted, store.BranchAbandoned}})
			return
		}
		if errors.Is(err, store.ErrResolutionRequired) {
			writeProblem(w, r, http.StatusBadRequest, "branch-resolution-required", "이유를 적어 주세요",
				"왜 채택했는지 또는 왜 접어 두는지를 남기지 않으면, 나중에 결정만 남고 이유는 사라집니다.", nil)
			return
		}
		slog.Info("resolve branch refused", "user_id", p.User.ID, "branch_id", branchID, "error", err)
		writeError(w, http.StatusBadRequest, "갈래를 정리할 수 없습니다.")
		return
	}
	writeJSON(w, http.StatusOK, branch)
}

func (s *Server) reopenBranch(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:write") {
		return
	}
	branchID, ok := parseID(w, r, "branchID")
	if !ok {
		return
	}
	p := principal(r)
	branch, err := s.Store.ReopenBranch(r.Context(), p.User.ID, branchID)
	if err != nil {
		slog.Info("reopen branch refused", "user_id", p.User.ID, "branch_id", branchID, "error", err)
		writeError(w, http.StatusBadRequest, "갈래를 다시 열 수 없습니다.")
		return
	}
	writeJSON(w, http.StatusOK, branch)
}

func (s *Server) deleteBranch(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:write") {
		return
	}
	branchID, ok := parseID(w, r, "branchID")
	if !ok {
		return
	}
	p := principal(r)
	if err := s.Store.DeleteBranch(r.Context(), p.User.ID, branchID); err != nil {
		slog.Info("delete branch refused", "user_id", p.User.ID, "branch_id", branchID, "error", err)
		writeError(w, http.StatusBadRequest, "갈래를 지울 수 없습니다.")
		return
	}
	// The thoughts stay; only the line stops being tracked.
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// setNoteBranch files a thought under a line, or takes it out of one.
func (s *Server) setNoteBranch(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:write") {
		return
	}
	noteID, ok := parseID(w, r, "noteID")
	if !ok {
		return
	}
	var body struct {
		BranchID *uuid.UUID `json:"branchId"`
	}
	if decodeJSON(w, r, &body) != nil {
		writeError(w, http.StatusBadRequest, "요청이 올바르지 않습니다.")
		return
	}
	p := principal(r)
	if err := s.Store.SetNoteBranch(r.Context(), p.User.ID, noteID, body.BranchID); err != nil {
		slog.Info("set note branch refused", "user_id", p.User.ID, "note_id", noteID, "error", err)
		writeError(w, http.StatusBadRequest, "생각을 갈래에 넣을 수 없습니다. 같은 공간의 갈래인지 확인해 주세요.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// turningPoints is the record of what someone decided, in the order they
// decided it.
//
// Only what was marked appears here. umm does not read a burst of activity and
// call it a turning point: a record that mixes decisions with guesses is worse
// than no record, because when you need it you cannot tell which is which.
func (s *Server) turningPoints(w http.ResponseWriter, r *http.Request) {
	if !requireScope(w, r, "notes:read") {
		return
	}
	var spaceID *uuid.UUID
	if raw := strings.TrimSpace(r.URL.Query().Get("spaceId")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			writeError(w, http.StatusBadRequest, "공간 ID가 올바르지 않습니다.")
			return
		}
		spaceID = &parsed
	}
	p := principal(r)
	points, more, err := s.Store.TurningPoints(r.Context(), p.User.ID, spaceID)
	if err != nil {
		slog.Warn("turning points failed", "user_id", p.User.ID, "error", err)
		writeError(w, http.StatusInternalServerError, "기록을 불러오지 못했습니다.")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"points": points, "hasMore": more})
}
