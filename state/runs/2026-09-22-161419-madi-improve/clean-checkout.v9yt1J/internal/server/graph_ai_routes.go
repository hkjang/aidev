package server

func (s *Server) registerGraphAI() {
	s.handle("GET /api/v1/workspaces/{id}/graph-ai/context", s.getGraphAIContext)
	s.handle("POST /api/v1/workspaces/{id}/graph-ai/analyze", s.analyzeGraphAI)
	s.handle("GET /api/v1/workspaces/{id}/graph-ai/runs", s.listGraphAIRuns)
	s.handle("GET /api/v1/graph-ai/runs/{id}", s.getGraphAIRun)
	s.handle("POST /api/v1/graph-ai/runs/{id}/cancel", s.cancelGraphAIRun)
	s.handle("DELETE /api/v1/graph-ai/runs/{id}", s.deleteGraphAIRun)
	s.handle("POST /api/v1/graph-ai/actions/{id}/confirm", s.confirmGraphAIAction)
}
