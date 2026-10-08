package httpapi

import (
	"net/http"
	"sort"
	"time"

	"github.com/hkjang/invenqor/server/internal/apitime"
)

var enrollmentDiagnosticComponents = []string{
	"agent_enrollment",
	"agent_transport",
	"agent_preflight",
}

type enrollmentCodeSummary struct {
	EventCode      string       `json:"event_code"`
	Level          string       `json:"level"`
	Count          int          `json:"count"`
	Message        string       `json:"message"`
	Remediation    string       `json:"remediation"`
	LastOccurredAt apitime.Time `json:"last_occurred_at"`
	LastRequestID  string       `json:"last_request_id"`
}

type enrollmentSourceSummary struct {
	SourceIP       string       `json:"source_ip"`
	AgentID        string       `json:"agent_id"`
	Attempts       int          `json:"attempts"`
	Failures       int          `json:"failures"`
	LastEventCode  string       `json:"last_event_code"`
	LastLevel      string       `json:"last_level"`
	LastMessage    string       `json:"last_message"`
	LastRequestID  string       `json:"last_request_id"`
	LastInstanceID string       `json:"last_instance_id"`
	LastOccurredAt apitime.Time `json:"last_occurred_at"`
	Remediation    string       `json:"remediation"`
	AgentVersion   string       `json:"agent_version"`
}

type enrollmentAgentSummary struct {
	ID              string       `json:"id"`
	AgentID         string       `json:"agent_id"`
	Hostname        string       `json:"hostname"`
	Status          string       `json:"status"`
	AuthMethod      string       `json:"auth_method"`
	Version         string       `json:"version"`
	CreatedAt       apitime.Time `json:"created_at"`
	LastSeenAt      apitime.Time `json:"last_seen_at"`
	LastInventoryAt apitime.Time `json:"last_inventory_at"`
}

// enrollmentDiagnostics answers the question an operator actually has when a
// fleet rollout goes quiet: which machines tried to register, what stopped
// them, and which ones registered but never delivered inventory. Searching the
// raw diagnostic log requires already knowing the answer.
func (s *Server) enrollmentDiagnostics(
	response http.ResponseWriter,
	request *http.Request,
) {
	hours := queryInt(request, "hours", 24, 1, 24*30)
	since := time.Now().UTC().Add(-time.Duration(hours) * time.Hour)
	items, err := s.diagnosticStore.Since(
		request.Context(),
		enrollmentDiagnosticComponents,
		since,
		2_000,
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	codes := map[string]*enrollmentCodeSummary{}
	sources := map[string]*enrollmentSourceSummary{}
	totals := map[string]int{
		"events":            len(items),
		"succeeded":         0,
		"rejected":          0,
		"preflight_checks":  0,
		"preflight_blocked": 0,
		"transport_failed":  0,
	}
	// Items arrive newest first, so the first sighting of a key is its latest.
	for _, item := range items {
		summary, found := codes[item.EventCode]
		if !found {
			summary = &enrollmentCodeSummary{
				EventCode:      item.EventCode,
				Level:          item.Level,
				Message:        item.Message,
				Remediation:    enrollmentRemediation(item.EventCode),
				LastOccurredAt: item.OccurredAt,
				LastRequestID:  item.RequestID,
			}
			codes[item.EventCode] = summary
		}
		summary.Count++
		switch {
		case item.EventCode == "AGENT_ENROLLMENT_SUCCEEDED":
			totals["succeeded"]++
		case item.EventCode == "AGENT_PREFLIGHT_READY":
			totals["preflight_checks"]++
		case item.EventCode == "AGENT_PREFLIGHT_BLOCKED":
			totals["preflight_checks"]++
			totals["preflight_blocked"]++
		case item.Component == "agent_transport":
			totals["transport_failed"]++
		case item.Level != "info":
			totals["rejected"]++
		}
		key := item.SourceIP + "|" + item.AgentID
		source, found := sources[key]
		if !found {
			source = &enrollmentSourceSummary{
				SourceIP:       item.SourceIP,
				AgentID:        item.AgentID,
				LastEventCode:  item.EventCode,
				LastLevel:      item.Level,
				LastMessage:    item.Message,
				LastRequestID:  item.RequestID,
				LastInstanceID: item.InstanceID,
				LastOccurredAt: item.OccurredAt,
				Remediation:    enrollmentRemediation(item.EventCode),
				AgentVersion:   detailString(item.Details, "agent_version"),
			}
			sources[key] = source
		}
		source.Attempts++
		if item.Level != "info" {
			source.Failures++
		}
	}
	agentsAwaiting, err := s.agentsMissingInventory(request)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	policy, _, policyErr := s.loadAgentEnrollmentPolicy(request.Context())
	payload := map[string]any{
		"generated_at":       time.Now().UTC(),
		"window_hours":       hours,
		"instance_id":        s.diagnosticStore.InstanceID(),
		"database_mode":      s.database.Mode(),
		"totals":             totals,
		"by_event_code":      sortedCodeSummaries(codes),
		"sources":            sortedSourceSummaries(sources),
		"awaiting_inventory": agentsAwaiting,
		"retention":          map[string]any{"days": 30, "maximum_events": 10000},
	}
	if policyErr == nil {
		payload["enrollment"] = policy.publicValue()
	} else {
		payload["enrollment_policy_error"] = policyErr.Error()
	}
	writeJSON(response, http.StatusOK, payload)
}

// agentsMissingInventory reports Agents that hold a credential but never
// delivered an inventory event, which is the signature of a one-way network
// path or an Agent that cannot read its own state directory.
func (s *Server) agentsMissingInventory(
	request *http.Request,
) ([]enrollmentAgentSummary, error) {
	rows, err := s.database.DB().QueryContext(
		request.Context(),
		`SELECT id, agent_id, hostname, status, auth_method, version,
		 created_at, last_seen_at, last_inventory_at
		 FROM agents
		 WHERE last_inventory_at IS NULL
		 ORDER BY created_at DESC LIMIT 200`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]enrollmentAgentSummary, 0)
	for rows.Next() {
		var item enrollmentAgentSummary
		if err := rows.Scan(
			&item.ID,
			&item.AgentID,
			&item.Hostname,
			&item.Status,
			&item.AuthMethod,
			&item.Version,
			&item.CreatedAt,
			&item.LastSeenAt,
			&item.LastInventoryAt,
		); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func sortedCodeSummaries(
	values map[string]*enrollmentCodeSummary,
) []enrollmentCodeSummary {
	result := make([]enrollmentCodeSummary, 0, len(values))
	for _, value := range values {
		result = append(result, *value)
	}
	sort.Slice(result, func(left, right int) bool {
		if result[left].Count != result[right].Count {
			return result[left].Count > result[right].Count
		}
		return result[left].EventCode < result[right].EventCode
	})
	return result
}

func sortedSourceSummaries(
	values map[string]*enrollmentSourceSummary,
) []enrollmentSourceSummary {
	result := make([]enrollmentSourceSummary, 0, len(values))
	for _, value := range values {
		result = append(result, *value)
	}
	sort.Slice(result, func(left, right int) bool {
		if (result[left].Failures > 0) != (result[right].Failures > 0) {
			return result[left].Failures > 0
		}
		if result[left].Attempts != result[right].Attempts {
			return result[left].Attempts > result[right].Attempts
		}
		return result[left].SourceIP < result[right].SourceIP
	})
	return result
}

func detailString(details map[string]any, key string) string {
	if value, found := details[key].(string); found {
		return value
	}
	return ""
}
