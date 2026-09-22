package server

import (
	"context"
	"encoding/json"
)

// Delete only journaled, unclaimed objects of cancelled sessions. A restored or
// changed provider is deliberately not contacted; operators review it manually.
func (s *Server) cleanupMigrationObjects(ctx context.Context) {
	rows, e := s.DB.Query(ctx, `SELECT item_id,document_id,object,fingerprint,workspace_id FROM (
 SELECT o.item_id::text,o.document_id::text,o.object,o.provider_fingerprint fingerprint,m.workspace_id::text,o.created_at FROM migration_session_objects o JOIN migration_session_items i ON i.id=o.item_id JOIN migration_sessions m ON m.id=i.session_id WHERE o.status IN('planned','ready') AND m.status='cancelled'
 UNION ALL SELECT i.id::text,''::text,i.staged_object,i.storage_fingerprint,m.workspace_id::text,i.updated_at FROM migration_session_items i JOIN migration_sessions m ON m.id=i.session_id WHERE i.staged_object<>'{}' AND m.status='cancelled'
 ) q ORDER BY created_at LIMIT 100`)
	if e != nil {
		return
	}
	type entry struct {
		Item, Doc, Fingerprint, Workspace string
		Raw                               []byte
	}
	entries := []entry{}
	for rows.Next() {
		var a entry
		if rows.Scan(&a.Item, &a.Doc, &a.Raw, &a.Fingerprint, &a.Workspace) != nil {
			rows.Close()
			return
		}
		entries = append(entries, a)
	}
	e = rows.Err()
	rows.Close()
	if e != nil {
		return
	}
	for _, a := range entries {
		func() {
			release, lockErr := s.migrationObjectLock(ctx, a.Item, a.Doc)
			if lockErr != nil {
				return
			}
			defer release()
			var cancelled bool
			if s.DB.QueryRow(ctx, "SELECT m.status='cancelled' FROM migration_sessions m JOIN migration_session_items i ON i.session_id=m.id WHERE i.id=$1", a.Item).Scan(&cancelled) != nil || !cancelled {
				return
			}
			provider, e := s.resolveStorage(ctx, a.Workspace)
			if e != nil || migrationStorageFingerprint(provider) != a.Fingerprint {
				return
			}
			var object storedObject
			if json.Unmarshal(a.Raw, &object) != nil {
				return
			}
			var used bool
			if s.DB.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM attachments WHERE object_key=$1 AND coalesce(storage_provider_id::text,'')=$2)", object.Key, object.ProviderID).Scan(&used) != nil || used {
				return
			}
			if e = s.cleanupStoredObject(ctx, object); e != nil {
				return
			}
			if a.Doc == "" {
				_, _ = s.DB.Exec(ctx, "UPDATE migration_session_items SET staged_object='{}',storage_fingerprint='' WHERE id=$1", a.Item)
			} else {
				_, _ = s.DB.Exec(ctx, "UPDATE migration_session_objects SET status='discarded' WHERE item_id=$1 AND document_id=$2", a.Item, a.Doc)
			}
		}()
	}
}
