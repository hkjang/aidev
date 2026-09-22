package server

// registerRunbook is called after Approval and Jobs are initialized. The
// approval adapter and durable worker share the same immutable execution plan.
func (s *Server) registerRunbook() {
	s.RegisterApprovalAdapter("runbook", s.runbookApprovalAdapter())
	s.RegisterJobHandler("runbook.execute", s.executeRunbookJob)
	s.admin("GET /api/v1/admin/runbook/settings", s.adminRunbookSettings)
	s.admin("PUT /api/v1/admin/runbook/settings", s.adminRunbookSettings)
	s.admin("GET /api/v1/admin/runbook/runners", s.adminRunbookRunners)
	s.admin("POST /api/v1/admin/runbook/runners", s.adminSaveRunbookRunner)
	s.admin("PUT /api/v1/admin/runbook/runners/{id}", s.adminSaveRunbookRunner)
	s.admin("POST /api/v1/admin/runbook/runners/{id}/test", s.adminTestRunbookRunner)
	s.admin("GET /api/v1/admin/runbook/runners/{id}/versions", s.adminRunbookVersions)
	s.admin("GET /api/v1/admin/runbook/runners/{id}/versions/{version}", s.adminRunbookVersions)
	s.admin("GET /api/v1/admin/runbook/uncertain", s.adminRunbookUncertain)
	s.admin("POST /api/v1/admin/runbook/executions/{id}/resolve", s.adminRunbookResolve)
	s.handle("GET /api/v1/runbook/runners", s.runbookRunners)
	s.handle("GET /api/v1/documents/{id}/runbook", s.getRunbook)
	s.handle("PUT /api/v1/documents/{id}/runbook", s.saveRunbook)
	s.handle("POST /api/v1/documents/{id}/runbook/prepare", s.prepareRunbook)
	s.handle("GET /api/v1/documents/{id}/runbook/executions", s.listRunbookExecutions)
	s.handle("GET /api/v1/runbook/executions/{id}", s.getRunbookExecution)
	s.handle("POST /api/v1/runbook/executions/{id}/approval", s.submitRunbookApproval)
	s.handle("POST /api/v1/runbook/executions/{id}/execute", s.executeRunbook)
	s.handle("POST /api/v1/runbook/executions/{id}/cancel", s.cancelRunbookExecution)
	s.handle("GET /api/v1/runbook/executions/{id}/events", s.streamRunbookEvents)
}
