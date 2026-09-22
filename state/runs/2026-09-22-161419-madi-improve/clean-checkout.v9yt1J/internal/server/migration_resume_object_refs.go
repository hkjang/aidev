package server

import (
	"context"
	"encoding/json"
	"path/filepath"

	"github.com/jackc/pgx/v5"
)

func (s *Server) stageMigrationReference(ctx context.Context, j Job, v migrationSession, item migrationSessionItem, docID string, prepared migrationPrepared) error {
	release, err := s.migrationObjectLock(ctx, item.ID, docID)
	if err != nil {
		return err
	}
	defer release()
	if _, _, e := s.migrationSessionJob(ctx, j); e != nil {
		return e
	}
	id := migrationAttachmentID(item.TargetID, docID)
	var exists bool
	if s.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM attachments WHERE id=$1 AND document_id=$2)", id, docID).Scan(&exists) != nil {
		return errMigrationChanged
	}
	if exists {
		return nil
	}
	provider, e := s.resolveStorage(ctx, v.WorkspaceID)
	if e != nil {
		return e
	}
	fingerprint := migrationStorageFingerprint(provider)
	object := storedObject{ProviderID: provider.ID, Key: "attachments/" + newID(), Checksum: digest(string(prepared.AttachmentData)), Size: int64(len(prepared.AttachmentData))}
	if provider.Kind == "local" {
		object.Path = filepath.Join(str(provider.Config, "root"), filepath.FromSlash(object.Key))
	}
	_, e = s.DB.Exec(ctx, `INSERT INTO migration_session_objects(item_id,document_id,attachment_id,object,provider_fingerprint) SELECT $1,$2,$3,$4,$5 WHERE EXISTS(SELECT 1 FROM migration_sessions WHERE id=$6 AND status='committing') ON CONFLICT(item_id,document_id) DO NOTHING`, item.ID, docID, id, jsonValue(object), fingerprint, v.ID)
	if e != nil {
		return e
	}
	var raw []byte
	var expected, status string
	if e = s.DB.QueryRow(ctx, "SELECT object,provider_fingerprint,status FROM migration_session_objects WHERE item_id=$1 AND document_id=$2", item.ID, docID).Scan(&raw, &expected, &status); e != nil {
		return e
	}
	if e = json.Unmarshal(raw, &object); e != nil {
		return e
	}
	if expected != fingerprint || status == "discarded" || object.Checksum != digest(string(prepared.AttachmentData)) {
		return errMigrationChanged
	}
	if e = s.recoverMigrationObject(ctx, j, v, item.ID, docID, provider, object, prepared); e != nil {
		return e
	}
	current, e := s.resolveStorage(ctx, v.WorkspaceID)
	if e != nil || migrationStorageFingerprint(current) != fingerprint {
		return errMigrationChanged
	}
	_, e = s.DB.Exec(ctx, "UPDATE migration_session_objects SET status='ready' WHERE item_id=$1 AND document_id=$2 AND status='planned'", item.ID, docID)
	return e
}

func (s *Server) stageMigrationObjects(ctx context.Context, j Job, v migrationSession, items []migrationSessionItem) error {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return err
	}
	_, _, err = migrationStoragePlanTx(ctx, tx, v.ID)
	_ = tx.Rollback(ctx)
	if err != nil {
		return err
	}
	if e := s.stageMigrationPrimaryObjects(ctx, j, v, items); e != nil {
		return e
	}
	index := newMigrationLinkIndex(items)
	for _, doc := range items {
		if !oneOf(doc.Kind, "document", "folder") || doc.Disposition == "unchanged" || str(doc.Metadata, "decision") == "skip" {
			continue
		}
		var raw []byte
		if e := s.DB.QueryRow(ctx, "SELECT prepared_data FROM migration_session_items WHERE id=$1", doc.ID).Scan(&raw); e != nil {
			return e
		}
		doc.PreparedData = raw
		prepared, e := s.migrationPreparedValue(doc)
		if e != nil {
			return e
		}
		for _, source := range prepared.Attachments {
			asset, ok := index.sources[source]
			if !ok || str(asset.Metadata, "decision") == "skip" {
				return jobPermanent("참조 중인 첨부는 제외할 수 없습니다")
			}
			asset, e = scanMigrationItem(s.DB.QueryRow(ctx, "SELECT "+migrationItemSelect+" FROM migration_session_items WHERE id=$1", asset.ID))
			if e != nil {
				return e
			}
			data, e := s.migrationPreparedValue(asset)
			if e != nil {
				return e
			}
			if e = s.stageMigrationReference(ctx, j, v, asset, doc.TargetID, data); e != nil {
				return e
			}
		}
	}
	return nil
}

func (s *Server) commitMigrationReferenceObjectsTx(ctx context.Context, tx pgx.Tx, p *Principal, v migrationSession, provider storageProvider) error {
	rows, e := tx.Query(ctx, `SELECT o.item_id::text,o.document_id::text,o.attachment_id::text,o.object,o.provider_fingerprint,o.status FROM migration_session_objects o JOIN migration_session_items i ON i.id=o.item_id WHERE i.session_id=$1 ORDER BY o.item_id,o.document_id`, v.ID)
	if e != nil {
		return e
	}
	type entry struct {
		Item, Doc, ID       string
		Raw                 []byte
		Fingerprint, Status string
	}
	entries := []entry{}
	for rows.Next() {
		var a entry
		if e = rows.Scan(&a.Item, &a.Doc, &a.ID, &a.Raw, &a.Fingerprint, &a.Status); e != nil {
			rows.Close()
			return e
		}
		entries = append(entries, a)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return e
	}
	for _, a := range entries {
		if a.Status != "ready" || a.Fingerprint != migrationStorageFingerprint(provider) {
			return errMigrationChanged
		}
		item, e := scanMigrationItem(tx.QueryRow(ctx, "SELECT "+migrationItemSelect+" FROM migration_session_items WHERE id=$1", a.Item))
		if e != nil {
			return e
		}
		prepared, e := s.migrationPreparedValue(item)
		if e != nil {
			return e
		}
		var object storedObject
		if e = json.Unmarshal(a.Raw, &object); e != nil {
			return e
		}
		clean, e := s.ProtectAttachmentTx(ctx, tx, p, a.Doc, v.WorkspaceID, prepared.Title, prepared.ContentType, prepared.AttachmentData)
		if e != nil {
			return e
		}
		if clean.Changed {
			return errMigrationChanged
		}
		_, e = tx.Exec(ctx, `INSERT INTO attachments(id,document_id,user_id,name,content_type,size,path,storage_provider_id,object_key,checksum_sha256) VALUES($1,$2,$3,$4,$5,$6,$7,NULLIF($8,'')::uuid,$9,$10)`, a.ID, a.Doc, p.ID, prepared.Title, prepared.ContentType, object.Size, object.Path, object.ProviderID, object.Key, object.Checksum)
		if e != nil {
			return e
		}
	}
	_, e = tx.Exec(ctx, "UPDATE migration_session_objects SET status='claimed' WHERE item_id IN(SELECT id FROM migration_session_items WHERE session_id=$1)", v.ID)
	return e
}
