package server

import (
	"context"
	"encoding/json"
	"path/filepath"

	"github.com/jackc/pgx/v5"
)

func migrationStorageFingerprint(p storageProvider) string {
	return digest(string(jsonValue(map[string]any{"id": p.ID, "workspace": p.WorkspaceID, "kind": p.Kind, "enabled": p.Enabled, "config": p.Config})))
}

func (s *Server) migrationStoredItem(ctx context.Context, id string) (storedObject, string, error) {
	var raw []byte
	var fingerprint string
	e := s.DB.QueryRow(ctx, "SELECT staged_object,storage_fingerprint FROM migration_session_items WHERE id=$1", id).Scan(&raw, &fingerprint)
	var object storedObject
	if e == nil {
		e = json.Unmarshal(raw, &object)
	}
	return object, fingerprint, e
}

func (s *Server) stageMigrationPrimaryObjects(ctx context.Context, j Job, v migrationSession, items []migrationSessionItem) error {
	for _, item := range items {
		if item.Kind != "attachment" || item.Disposition == "unchanged" || str(item.Metadata, "decision") == "skip" {
			continue
		}
		if e := s.stageMigrationPrimaryObject(ctx, j, v, item); e != nil {
			return e
		}
	}
	return nil
}

func (s *Server) stageMigrationPrimaryObject(ctx context.Context, j Job, v migrationSession, item migrationSessionItem) error {
	release, e := s.migrationObjectLock(ctx, item.ID, "")
	if e != nil {
		return e
	}
	defer release()
	if _, _, e := s.migrationSessionJob(ctx, j); e != nil {
		return e
	}
	item, e = scanMigrationItem(s.DB.QueryRow(ctx, "SELECT "+migrationItemSelect+" FROM migration_session_items WHERE id=$1", item.ID))
	if e != nil {
		return e
	}
	prepared, e := s.migrationPreparedValue(item)
	if e != nil {
		return e
	}
	provider, e := s.resolveStorage(ctx, v.WorkspaceID)
	if e != nil {
		return e
	}
	fingerprint := migrationStorageFingerprint(provider)
	object, expected, e := s.migrationStoredItem(ctx, item.ID)
	if e != nil {
		return e
	}
	if expected != "" && expected != fingerprint {
		return jobPermanent("준비한 저장소 정책이 변경되었습니다. 새 이관을 준비하세요")
	}
	if object.Key == "" {
		id := newID()
		object = storedObject{ProviderID: provider.ID, Key: "attachments/" + id, Checksum: digest(string(prepared.AttachmentData)), Size: int64(len(prepared.AttachmentData))}
		if provider.Kind == "local" {
			object.Path = filepath.Join(str(provider.Config, "root"), filepath.FromSlash(object.Key))
		}
		// Persist the exact intended key before the external write. A restarted
		// process verifies that key, never allocates a second public attachment.
		_, e = s.DB.Exec(ctx, `UPDATE migration_session_items SET staged_object=$2,storage_fingerprint=$3 WHERE id=$1 AND staged_object='{}' AND EXISTS(SELECT 1 FROM migration_sessions WHERE id=$4 AND status='committing')`, item.ID, jsonValue(object), fingerprint, v.ID)
		if e != nil {
			return e
		}
		object, expected, e = s.migrationStoredItem(ctx, item.ID)
		if e != nil {
			return e
		}
		if object.Key == "" || expected != fingerprint {
			return errMigrationChanged
		}
	}
	if e = s.recoverMigrationObject(ctx, j, v, item.ID, "", provider, object, prepared); e != nil {
		return e
	}
	current, e := s.resolveStorage(ctx, v.WorkspaceID)
	if e != nil || migrationStorageFingerprint(current) != fingerprint {
		return errMigrationChanged
	}
	return nil
}

func (s *Server) commitMigrationAttachmentsTx(ctx context.Context, tx pgx.Tx, p *Principal, v migrationSession, items []migrationSessionItem) error {
	index := newMigrationLinkIndex(items)
	root, exists := index.sources["@madi-folder:attachment-root"]
	if !exists {
		return nil
	}
	// Configuration writes require ROW EXCLUSIVE table locks. This short final
	// publication phase pins both existing and newly inserted assignments.
	if _, e := tx.Exec(ctx, "LOCK TABLE storage_assignments,storage_settings,storage_providers IN SHARE MODE"); e != nil {
		return e
	}
	if _, e := tx.Exec(ctx, "SELECT id FROM settings WHERE id=1 FOR SHARE"); e != nil {
		return e
	}
	provider, e := s.resolveStorage(ctx, v.WorkspaceID)
	if e != nil {
		return e
	}
	for _, item := range items {
		if item.Kind != "attachment" || item.Disposition == "unchanged" || str(item.Metadata, "decision") == "skip" {
			continue
		}
		var raw []byte
		var fingerprint string
		if e = tx.QueryRow(ctx, "SELECT prepared_data,storage_fingerprint FROM migration_session_items WHERE id=$1", item.ID).Scan(&raw, &fingerprint); e != nil {
			return e
		}
		item.PreparedData = raw
		prepared, e := s.migrationPreparedValue(item)
		if e != nil {
			return e
		}
		object, _, e := s.migrationStoredItem(ctx, item.ID)
		if e != nil {
			return e
		}
		if fingerprint != migrationStorageFingerprint(provider) || object.Key == "" {
			return errMigrationChanged
		}
		clean, e := s.ProtectAttachmentTx(ctx, tx, p, root.TargetID, v.WorkspaceID, prepared.Title, prepared.ContentType, prepared.AttachmentData)
		if e != nil {
			return e
		}
		if clean.Changed {
			return errMigrationChanged
		}
		_, e = tx.Exec(ctx, `INSERT INTO attachments(id,document_id,user_id,name,content_type,size,path,storage_provider_id,object_key,checksum_sha256) VALUES($1,$2,$3,$4,$5,$6,$7,NULLIF($8,'')::uuid,$9,$10)`, item.TargetID, root.TargetID, p.ID, prepared.Title, prepared.ContentType, object.Size, object.Path, object.ProviderID, object.Key, object.Checksum)
		if e != nil {
			return e
		}
	}
	return s.commitMigrationReferenceObjectsTx(ctx, tx, p, v, provider)
}
