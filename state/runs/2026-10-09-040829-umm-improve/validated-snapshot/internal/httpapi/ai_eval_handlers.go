package httpapi

import (
	"encoding/json"
	"net/http"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/hkjang/umm/internal/dream"
)

var evalDreamTypes = []string{"connection", "question", "expansion", "contrarian", "rediscovery", "action", "pattern", "free"}

type evalCaseRequest struct {
	Name           string   `json:"name"`
	DreamType      string   `json:"dreamType"`
	InputNotes     []string `json:"inputNotes"`
	ExpectedTerms  []string `json:"expectedTerms"`
	ForbiddenTerms []string `json:"forbiddenTerms"`
	Active         bool     `json:"active"`
}

func validateEvalCase(value *evalCaseRequest) bool {
	value.Name = strings.TrimSpace(value.Name)
	if utf8.RuneCountInString(value.Name) < 1 || utf8.RuneCountInString(value.Name) > 200 || !slices.Contains(evalDreamTypes, value.DreamType) || len(value.InputNotes) < 2 || len(value.InputNotes) > 20 || len(value.ExpectedTerms) > 20 || len(value.ForbiddenTerms) > 20 {
		return false
	}
	for index, note := range value.InputNotes {
		value.InputNotes[index] = strings.TrimSpace(note)
		if utf8.RuneCountInString(value.InputNotes[index]) < 1 || utf8.RuneCountInString(value.InputNotes[index]) > 1200 {
			return false
		}
	}
	for _, terms := range [][]string{value.ExpectedTerms, value.ForbiddenTerms} {
		for _, term := range terms {
			if utf8.RuneCountInString(strings.TrimSpace(term)) > 100 {
				return false
			}
		}
	}
	return true
}

func (s *Server) listAIEvals(w http.ResponseWriter, r *http.Request) {
	rows, err := s.Store.Pool.Query(r.Context(), `
		SELECT c.id,c.name,c.dream_type,c.input_notes,c.expected_terms,c.forbidden_terms,c.active,c.created_at,c.updated_at,
		       lr.id,lr.status,lr.score,lr.model,lr.prompt_version,lr.content,lr.details,lr.latency_ms,lr.created_at
		FROM ai_eval_cases c
		LEFT JOIN LATERAL (SELECT * FROM ai_eval_runs WHERE case_id=c.id ORDER BY created_at DESC LIMIT 1) lr ON true
		ORDER BY c.created_at DESC`)
	if err != nil {
		writeError(w, 500, "AI 평가 케이스를 불러오지 못했습니다.")
		return
	}
	defer rows.Close()
	items := []map[string]any{}
	for rows.Next() {
		var id uuid.UUID
		var name, dreamType string
		var inputRaw []byte
		var expected, forbidden []string
		var active bool
		var created, updated time.Time
		var runID *uuid.UUID
		var status, model, prompt, content *string
		var score *float64
		var details []byte
		var latency *int
		var runCreated *time.Time
		if err := rows.Scan(&id, &name, &dreamType, &inputRaw, &expected, &forbidden, &active, &created, &updated, &runID, &status, &score, &model, &prompt, &content, &details, &latency, &runCreated); err != nil {
			writeError(w, 500, "AI 평가 케이스를 읽지 못했습니다.")
			return
		}
		var inputs []string
		if err := json.Unmarshal(inputRaw, &inputs); err != nil {
			writeError(w, 500, "AI 평가 입력 형식이 올바르지 않습니다.")
			return
		}
		var latest any
		if runID != nil {
			var parsedDetails any
			if err := json.Unmarshal(details, &parsedDetails); err != nil {
				writeError(w, 500, "AI 평가 결과 형식이 올바르지 않습니다.")
				return
			}
			latest = map[string]any{"id": runID, "status": status, "score": score, "model": model, "promptVersion": prompt, "content": content, "details": parsedDetails, "latencyMs": latency, "createdAt": runCreated}
		}
		items = append(items, map[string]any{"id": id, "name": name, "dreamType": dreamType, "inputNotes": inputs, "expectedTerms": expected, "forbiddenTerms": forbidden, "active": active, "createdAt": created, "updatedAt": updated, "latestRun": latest})
	}
	if err := rows.Err(); err != nil {
		writeError(w, 500, "AI 평가 케이스를 불러오지 못했습니다.")
		return
	}
	writeJSON(w, 200, map[string]any{"cases": items, "dreamTypes": evalDreamTypes})
}

func (s *Server) createAIEval(w http.ResponseWriter, r *http.Request) {
	var body evalCaseRequest
	if decodeJSON(w, r, &body) != nil || !validateEvalCase(&body) {
		writeError(w, 400, "AI 평가 케이스 형식이 올바르지 않습니다.")
		return
	}
	inputRaw, _ := json.Marshal(body.InputNotes)
	p := principal(r)
	var id uuid.UUID
	err := s.Store.Pool.QueryRow(r.Context(), `INSERT INTO ai_eval_cases(name,dream_type,input_notes,expected_terms,forbidden_terms,active,created_by) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`, body.Name, body.DreamType, inputRaw, body.ExpectedTerms, body.ForbiddenTerms, body.Active, p.User.ID).Scan(&id)
	if err != nil {
		writeError(w, 500, "AI 평가 케이스를 저장하지 못했습니다.")
		return
	}
	s.Store.Audit(r.Context(), &p.User.ID, "ai_eval.create", "ai_eval_case", id.String(), map[string]any{"dreamType": body.DreamType})
	writeJSON(w, 201, map[string]any{"id": id})
}

func (s *Server) deleteAIEval(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "caseID")
	if !ok {
		return
	}
	p := principal(r)
	command, err := s.Store.Pool.Exec(r.Context(), `DELETE FROM ai_eval_cases WHERE id=$1`, id)
	if err != nil {
		writeError(w, 500, "AI 평가 케이스를 삭제하지 못했습니다.")
		return
	}
	if command.RowsAffected() == 0 {
		writeError(w, 404, "AI 평가 케이스를 찾을 수 없습니다.")
		return
	}
	s.Store.Audit(r.Context(), &p.User.ID, "ai_eval.delete", "ai_eval_case", id.String(), map[string]any{})
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) runAIEval(w http.ResponseWriter, r *http.Request) {
	id, ok := parseID(w, r, "caseID")
	if !ok {
		return
	}
	var request dream.EvalRequest
	var inputRaw []byte
	err := s.Store.Pool.QueryRow(r.Context(), `SELECT dream_type,input_notes,expected_terms,forbidden_terms FROM ai_eval_cases WHERE id=$1`, id).Scan(&request.DreamType, &inputRaw, &request.ExpectedTerms, &request.ForbiddenTerms)
	if err != nil || json.Unmarshal(inputRaw, &request.InputNotes) != nil {
		writeError(w, 404, "AI 평가 케이스를 찾을 수 없습니다.")
		return
	}
	p := principal(r)
	result, runErr := s.Dreams.Evaluate(r.Context(), p.User.ID, request)
	status := "failed"
	errorMessage := ""
	if result.Passed {
		status = "passed"
	}
	if runErr != nil {
		status, errorMessage = "error", runErr.Error()
	}
	details, _ := json.Marshal(result.Details)
	var runID uuid.UUID
	insertErr := s.Store.Pool.QueryRow(r.Context(), `INSERT INTO ai_eval_runs(case_id,model,prompt_version,status,content,score,details,input_tokens,output_tokens,latency_ms,error,created_by) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) RETURNING id`, id, result.Model, result.PromptVersion, status, result.Content, result.Score, details, result.InputTokens, result.OutputTokens, result.LatencyMS, errorMessage, p.User.ID).Scan(&runID)
	if insertErr != nil {
		writeError(w, 500, "AI 평가 실행 결과를 저장하지 못했습니다.")
		return
	}
	s.Store.Audit(r.Context(), &p.User.ID, "ai_eval.run", "ai_eval_case", id.String(), map[string]any{"runId": runID, "status": status, "score": result.Score})
	if runErr != nil {
		if writeAIQuotaProblem(w, r, runErr) {
			return
		}
		writeProblem(w, r, 502, "ai-eval-gateway", "AI 평가 실행 실패", "Gateway 응답을 받지 못했지만 실패 기록은 저장했습니다: "+runErr.Error(), map[string]any{"runId": runID})
		return
	}
	writeJSON(w, 200, map[string]any{"id": runID, "status": status, "result": result})
}
