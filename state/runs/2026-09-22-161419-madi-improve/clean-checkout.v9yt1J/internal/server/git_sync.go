package server

import (
	"context"
	_ "embed"
)

//go:embed git_sync.sql
var gitSyncSchema string

func (s *Server) migrateGitSync(ctx context.Context) error {
	_, e := s.DB.Exec(ctx, gitSyncSchema)
	return e
}

func defaultGitSyncSettings() map[string]any {
	return map[string]any{"enabled": false, "allowed_hosts": []string{}, "allow_private_networks": false, "max_snapshot_mb": 100}
}

type gitSyncFile struct {
	Path         string `json:"path"`
	Data         []byte `json:"data"`
	Hash         string `json:"hash"`
	DocumentID   string `json:"document_id,omitempty"`
	AttachmentID string `json:"attachment_id,omitempty"`
	ContentType  string `json:"content_type,omitempty"`
}
type gitSyncSnapshot struct {
	Files               []gitSyncFile    `json:"files"`
	RemoteCommit        string           `json:"remote_commit"`
	SelectedDocumentIDs []string         `json:"selected_document_ids"`
	SourceVersions      map[string]int64 `json:"source_versions"`
	ConfigFingerprint   string           `json:"config_fingerprint"`
}
