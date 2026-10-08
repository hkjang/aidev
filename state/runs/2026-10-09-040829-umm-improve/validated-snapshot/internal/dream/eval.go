package dream

import (
	"context"
	"errors"
	"math"
	"strings"

	"github.com/google/uuid"
)

type EvalRequest struct {
	DreamType      string
	InputNotes     []string
	ExpectedTerms  []string
	ForbiddenTerms []string
}

type EvalResult struct {
	Content       string         `json:"content"`
	Model         string         `json:"model"`
	PromptVersion string         `json:"promptVersion"`
	Score         float64        `json:"score"`
	Passed        bool           `json:"passed"`
	Details       map[string]any `json:"details"`
	InputTokens   int            `json:"inputTokens"`
	OutputTokens  int            `json:"outputTokens"`
	LatencyMS     int64          `json:"latencyMs"`
}

func (s *Service) Evaluate(ctx context.Context, userID uuid.UUID, request EvalRequest) (EvalResult, error) {
	if len(request.InputNotes) < 2 {
		return EvalResult{}, errors.New("evaluation requires at least two input notes")
	}
	var cfg Config
	if err := s.Store.GetSetting(ctx, "dream", &cfg); err != nil {
		return EvalResult{}, err
	}
	var gateway GatewayConfig
	if err := s.Store.GetSetting(ctx, "ai_gateway", &gateway); err != nil {
		return EvalResult{}, err
	}
	sources := make([]sourceNote, 0, len(request.InputNotes))
	for _, content := range request.InputNotes {
		sources = append(sources, sourceNote{ID: uuid.New(), Content: strings.TrimSpace(content)})
	}
	raw, inputTokens, outputTokens, model, latency, err := s.callGatewayWithGuidance(ctx, userID, cfg, gateway, sources, request.DreamType, "")
	if err != nil {
		return EvalResult{Model: model, PromptVersion: gateway.PromptVersion, InputTokens: inputTokens, OutputTokens: outputTokens, LatencyMS: latency.Milliseconds()}, err
	}
	output := parseDreamOutput(raw, len(sources))
	assessment := assessQuality(output, sources)
	content := strings.ToLower(output.Content + " " + output.Rationale + " " + output.SuggestedAction)
	expectedHits := []string{}
	missing := []string{}
	for _, term := range request.ExpectedTerms {
		term = strings.TrimSpace(term)
		if term == "" {
			continue
		}
		if strings.Contains(content, strings.ToLower(term)) {
			expectedHits = append(expectedHits, term)
		} else {
			missing = append(missing, term)
		}
	}
	forbiddenHits := []string{}
	for _, term := range request.ForbiddenTerms {
		term = strings.TrimSpace(term)
		if term != "" && strings.Contains(content, strings.ToLower(term)) {
			forbiddenHits = append(forbiddenHits, term)
		}
	}
	expectedScore := 1.0
	if total := len(expectedHits) + len(missing); total > 0 {
		expectedScore = float64(len(expectedHits)) / float64(total)
	}
	forbiddenScore := 1.0
	if len(forbiddenHits) > 0 {
		forbiddenScore = 0
	}
	score := math.Round((assessment.Score*.6+expectedScore*.3+forbiddenScore*.1)*10000) / 10000
	passed := score >= .7 && len(missing) == 0 && len(forbiddenHits) == 0 && assessment.PassesGrounding
	details := map[string]any{
		"quality": assessment.Score, "groundedness": assessment.Groundedness,
		"novelty": assessment.Novelty, "specificity": assessment.Specificity,
		"passesGrounding": assessment.PassesGrounding, "expectedHits": expectedHits,
		"missingTerms": missing, "forbiddenHits": forbiddenHits, "dreamType": output.Type,
		"rationale": output.Rationale, "suggestedAction": output.SuggestedAction,
	}
	return EvalResult{Content: output.Content, Model: model, PromptVersion: gateway.PromptVersion, Score: score, Passed: passed, Details: details, InputTokens: inputTokens, OutputTokens: outputTokens, LatencyMS: latency.Milliseconds()}, nil
}
