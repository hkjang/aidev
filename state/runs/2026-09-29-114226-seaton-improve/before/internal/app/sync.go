package app

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type syncPayload struct {
	Organizations []struct {
		ExternalID       string `json:"externalId"`
		Name             string `json:"name"`
		ParentExternalID string `json:"parentExternalId"`
		Color            string `json:"color"`
	} `json:"organizations"`
	Employees []struct {
		EmployeeNo             string `json:"employeeNo"`
		Name                   string `json:"name"`
		Email                  string `json:"email"`
		OrganizationExternalID string `json:"organizationExternalId"`
		Title                  string `json:"title"`
		Position               string `json:"position"`
		Workplace              string `json:"workplace"`
		Status                 string `json:"status"`
	} `json:"employees"`
}

func (s *Server) syncEmployeesNow(w http.ResponseWriter, r *http.Request) {
	result, err := s.runEmployeeSync(r.Context())
	if err != nil {
		writeError(w, http.StatusBadGateway, "hr_sync_failed", err.Error())
		return
	}
	u, _ := userFrom(r)
	s.audit(r.Context(), u.ID, "hr.sync", "employee", "", r.RemoteAddr, result)
	writeJSON(w, 200, result)
}

func (s *Server) runEmployeeSync(parent context.Context) (map[string]any, error) {
	apiURL, err := s.getSetting(parent, "hr.api_url")
	if err != nil || apiURL == "" {
		return nil, errors.New("인사 API URL을 설정하세요")
	}
	parsed, err := url.Parse(apiURL)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, errors.New("인사 API URL은 http 또는 https 주소여야 합니다")
	}
	token, _ := s.getSetting(parent, "hr.api_token")
	runID := newID()
	_, _ = s.db.Exec(parent, `INSERT INTO employee_sync_runs(id,status) VALUES($1,'running')`, runID)
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	response, err := (&http.Client{Timeout: 2 * time.Minute}).Do(req)
	if err != nil {
		s.finishSync(parent, runID, 0, 0, err)
		return nil, fmt.Errorf("인사 API 연결 실패: %w", err)
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		err = fmt.Errorf("인사 API 응답 코드 %d", response.StatusCode)
		s.finishSync(parent, runID, 0, 0, err)
		return nil, err
	}
	raw, err := io.ReadAll(io.LimitReader(response.Body, 20<<20))
	if err != nil {
		return nil, err
	}
	var payload syncPayload
	if err = json.Unmarshal(raw, &payload); err != nil {
		s.finishSync(parent, runID, 0, 0, err)
		return nil, errors.New("인사 API JSON 형식을 확인하세요")
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)
	orgIDs := map[string]string{}
	for _, org := range payload.Organizations {
		if strings.TrimSpace(org.ExternalID) == "" || strings.TrimSpace(org.Name) == "" {
			continue
		}
		id := newID()
		color := org.Color
		if color == "" {
			color = "#2563EB"
		}
		if err = tx.QueryRow(ctx, `INSERT INTO organizations(id,external_id,name,color) VALUES($1,$2,$3,$4) ON CONFLICT(external_id) DO UPDATE SET name=EXCLUDED.name,color=EXCLUDED.color,updated_at=now() RETURNING id`, id, org.ExternalID, org.Name, color).Scan(&id); err != nil {
			return nil, err
		}
		orgIDs[org.ExternalID] = id
	}
	for _, org := range payload.Organizations {
		if id := orgIDs[org.ExternalID]; id != "" && org.ParentExternalID != "" {
			_, err = tx.Exec(ctx, `UPDATE organizations SET parent_id=$2 WHERE id=$1`, id, orgIDs[org.ParentExternalID])
			if err != nil {
				return nil, err
			}
		}
	}
	retired := []string{}
	employeeCount := 0
	for _, employee := range payload.Employees {
		if strings.TrimSpace(employee.EmployeeNo) == "" || strings.TrimSpace(employee.Name) == "" {
			continue
		}
		status := employee.Status
		if status == "" {
			status = "active"
		}
		if status != "active" && status != "leave" && status != "retired" {
			return nil, fmt.Errorf("직원 %s의 status 값이 올바르지 않습니다", employee.EmployeeNo)
		}
		id := newID()
		orgID := orgIDs[employee.OrganizationExternalID]
		err = tx.QueryRow(ctx, `INSERT INTO employees(id,employee_no,name,email,organization_id,title,position,workplace,status) VALUES($1,$2,$3,NULLIF($4,''),NULLIF($5,''),NULLIF($6,''),NULLIF($7,''),NULLIF($8,''),$9) ON CONFLICT(employee_no) DO UPDATE SET name=EXCLUDED.name,email=EXCLUDED.email,organization_id=EXCLUDED.organization_id,title=EXCLUDED.title,position=EXCLUDED.position,workplace=EXCLUDED.workplace,status=EXCLUDED.status,updated_at=now() RETURNING id`, id, employee.EmployeeNo, employee.Name, employee.Email, orgID, employee.Title, employee.Position, employee.Workplace, status).Scan(&id)
		if err != nil {
			return nil, err
		}
		employeeCount++
		if status == "retired" {
			retired = append(retired, id)
		}
	}
	for _, employeeID := range retired {
		var seatID string
		err = tx.QueryRow(ctx, `UPDATE seat_assignments SET ended_at=now() WHERE employee_id=$1 AND ended_at IS NULL RETURNING seat_id`, employeeID).Scan(&seatID)
		if err == nil {
			_, _ = tx.Exec(ctx, `UPDATE seats SET status='available',updated_at=now() WHERE id=$1`, seatID)
			_, _ = tx.Exec(ctx, `INSERT INTO seat_history(id,employee_id,previous_seat_id,reason,source) VALUES($1,$2,$3,'퇴직자 자동 좌석 해제','hr_sync')`, newID(), employeeID, seatID)
		}
	}
	if err = tx.Commit(ctx); err != nil {
		s.finishSync(parent, runID, 0, 0, err)
		return nil, err
	}
	s.finishSync(parent, runID, employeeCount, len(orgIDs), nil)
	return map[string]any{"runId": runID, "employees": employeeCount, "organizations": len(orgIDs), "retiredAssignmentsReleased": len(retired)}, nil
}

func (s *Server) finishSync(ctx context.Context, id string, employees, organizations int, syncErr error) {
	message := ""
	status := "completed"
	if syncErr != nil {
		status = "failed"
		message = syncErr.Error()
	}
	_, _ = s.db.Exec(ctx, `UPDATE employee_sync_runs SET status=$2,employee_count=$3,organization_count=$4,error=NULLIF($5,''),completed_at=now() WHERE id=$1`, id, status, employees, organizations, message)
}
