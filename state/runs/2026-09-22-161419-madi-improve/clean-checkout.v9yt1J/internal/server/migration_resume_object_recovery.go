package server

import (
	"bytes"
	"context"
	"errors"
	"io"

	"github.com/jackc/pgx/v5"
)

// Every writer and collector of one journaled object takes this lock. It is
// process-independent: a dead process releases it, while a delayed old lease
// cannot delete a newer worker's in-flight upload.
func (s *Server) migrationObjectLock(ctx context.Context, itemID, documentID string) (func(), error) {
	tx, err := s.DB.Begin(ctx)
	if err != nil {
		return nil, err
	}
	if _, err = tx.Exec(ctx, "SET LOCAL lock_timeout='5s'"); err != nil {
		_ = tx.Rollback(context.WithoutCancel(ctx))
		return nil, err
	}
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,9327))", itemID+":"+documentID); err != nil {
		_ = tx.Rollback(context.WithoutCancel(ctx))
		return nil, err
	}
	return func() { _ = tx.Rollback(context.WithoutCancel(ctx)) }, nil
}

// Call only while holding migrationObjectLock and after loading the immutable
// journal entry. Only an unclaimed object named by that entry may be repaired.
func (s *Server) recoverMigrationObject(ctx context.Context, j Job, v migrationSession, itemID, docID string, provider storageProvider, object storedObject, prepared migrationPrepared) error {
	if _, _, err := s.migrationSessionJob(ctx, j); err != nil {
		return err
	}
	var leaseOK bool
	if s.DB.QueryRow(ctx, `SELECT status='running' AND NOT cancel_requested AND lease_id=$2::uuid AND lease_until>now() FROM automation_jobs WHERE id=$1`, j.ID, j.LeaseID).Scan(&leaseOK) != nil || !leaseOK {
		return errMigrationChanged
	}
	if object.Checksum != digest(string(prepared.AttachmentData)) || object.Size != int64(len(prepared.AttachmentData)) || !validObjectKey(object.Key) {
		return errMigrationChanged
	}
	var owned, used bool
	err := s.DB.QueryRow(ctx, `SELECT EXISTS(
 SELECT 1 FROM migration_session_items i JOIN migration_sessions m ON m.id=i.session_id WHERE i.id=$1 AND m.id=$3 AND m.status='committing' AND $2='' AND i.staged_object=$4::jsonb
 UNION ALL SELECT 1 FROM migration_session_objects o JOIN migration_session_items i ON i.id=o.item_id JOIN migration_sessions m ON m.id=i.session_id WHERE o.item_id=$1 AND o.document_id::text=$2 AND m.id=$3 AND m.status='committing' AND o.status IN('planned','ready') AND o.object=$4::jsonb),
 EXISTS(SELECT 1 FROM attachments WHERE object_key=$5 AND coalesce(storage_provider_id::text,'')=$6)`, itemID, docID, v.ID, jsonValue(object), object.Key, object.ProviderID).Scan(&owned, &used)
	if err != nil {
		return err
	}
	if !owned || used {
		return errMigrationChanged
	}
	reader, openErr := s.openStoredObject(ctx, object)
	if openErr == nil {
		data, readErr := io.ReadAll(io.LimitReader(reader, object.Size+1))
		_ = reader.Close()
		if readErr == nil && int64(len(data)) == object.Size && digest(string(data)) == object.Checksum {
			return nil
		}
		// A checksum mismatch is repairable only for this private, unclaimed
		// staging key. Public attachments and another migration's key are excluded.
		if readErr != nil {
			return errors.New("준비 첨부를 읽지 못했습니다. 저장소 연결을 확인하세요")
		}
		if err = s.cleanupStoredObject(ctx, object); err != nil {
			return err
		}
	}
	if _, _, err = s.migrationSessionJob(ctx, j); err != nil {
		return err
	}
	written, err := s.putStoredObject(ctx, provider, object.Key, bytes.NewReader(prepared.AttachmentData), 50<<20, prepared.ContentType)
	if err != nil {
		return err
	}
	if written.Checksum != object.Checksum || written.Size != object.Size {
		return errMigrationChanged
	}
	return nil
}

// Source bytes alone do not bound storage: one attachment can be referenced by
// many private documents. Count every isolated copy before contacting storage.
func migrationStoragePlanTx(ctx context.Context, tx pgx.Tx, id string) (int64, int64, error) {
	var size, objects, limit int64
	err := tx.QueryRow(ctx, `WITH copies AS (
 SELECT source_bytes n FROM migration_session_items WHERE session_id=$1 AND kind='attachment' AND disposition<>'unchanged' AND coalesce(metadata->>'decision','apply')<>'skip'
 UNION ALL
 SELECT a.source_bytes FROM migration_session_items d CROSS JOIN LATERAL jsonb_array_elements_text(coalesce(d.compatibility->'attachment_sources','[]')) ref(source_id)
 JOIN migration_session_items a ON a.session_id=d.session_id AND a.source_id=ref.source_id
 WHERE d.session_id=$1 AND d.kind IN('document','folder') AND d.disposition<>'unchanged' AND coalesce(d.metadata->>'decision','apply')<>'skip')
 SELECT coalesce(sum(n),0),count(*),(SELECT max_session_bytes FROM migration_settings WHERE id=1) FROM copies`, id).Scan(&size, &objects, &limit)
	if err != nil {
		return 0, 0, err
	}
	if size > limit || objects > 100000 {
		return size, objects, jobPermanent("문서별 첨부 복제량이 이관 저장 한도를 초과합니다. 묶음을 나누거나 참조를 줄인 뒤 다시 준비하세요")
	}
	return size, objects, nil
}

func migrationDependencyIssuesTx(ctx context.Context, tx pgx.Tx, id string) (int, error) {
	var count int
	err := tx.QueryRow(ctx, `SELECT count(*) FROM migration_session_items d CROSS JOIN LATERAL jsonb_array_elements_text(coalesce(d.compatibility->'attachment_sources','[]')||coalesce(d.compatibility->'links'->'reference_sources','[]')) ref(source_id) LEFT JOIN migration_session_items a ON a.session_id=d.session_id AND a.source_id=ref.source_id WHERE d.session_id=$1 AND coalesce(d.metadata->>'decision','apply')<>'skip' AND (a.id IS NULL OR a.metadata->>'decision'='skip')`, id).Scan(&count)
	return count, err
}
