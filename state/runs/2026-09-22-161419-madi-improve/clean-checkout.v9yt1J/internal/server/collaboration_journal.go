package server

import (
	"context"
	_ "embed"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/reearth/ygo/crdt"
)

//go:embed collaboration_journal.sql
var collaborationJournalSchema string

const (
	collaborationCheckpointUpdates = 64
	collaborationCheckpointBytes   = 1 << 20
	collaborationCompactBytes      = 12 << 20
)

// Replays only the bounded tail after the last durable snapshot. The caller
// holds the canonical document row lock; no concurrent writer can alter its head.
func collaborationReplay(ctx context.Context, tx pgx.Tx, id, epoch string, snapshot []byte, snapshotSequence, sequence int64) ([]byte, error) {
	if snapshotSequence > sequence {
		return nil, collaborationError("journal_gap", "편집 저장 순서를 확인할 수 없습니다. 관리자에게 문의하세요")
	}
	if snapshotSequence == sequence {
		return snapshot, nil
	}
	rows, err := tx.Query(ctx, `SELECT sequence,update_data FROM collaboration_updates WHERE document_id=$1 AND epoch=$2 AND sequence>$3 AND sequence<=$4 ORDER BY sequence`, id, epoch, snapshotSequence, sequence)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	updates := [][]byte{}
	if len(snapshot) > 0 {
		updates = append(updates, snapshot)
	}
	next := snapshotSequence + 1
	size := len(snapshot)
	for rows.Next() {
		var seq int64
		var update []byte
		if err = rows.Scan(&seq, &update); err != nil {
			return nil, err
		}
		if seq != next || len(updates) > collaborationCheckpointUpdates+1 {
			return nil, collaborationError("journal_gap", "편집 증분 이력이 불완전합니다. 원문을 보존하고 관리자에게 문의하세요")
		}
		next++
		size += len(update)
		if size > collaborationMaxState+collaborationMaxUpdate {
			return nil, collaborationError("state_limit", "편집 증분 이력이 크기 제한을 초과했습니다")
		}
		updates = append(updates, update)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	if next != sequence+1 {
		return nil, collaborationError("journal_gap", "편집 증분 이력에 누락이 있습니다. 원문은 변경되지 않았습니다")
	}
	return crdt.MergeUpdatesV1(updates...)
}

func collaborationResetTx(ctx context.Context, tx pgx.Tx, id, epoch, markdown string, version int, reason string) error {
	if _, err := tx.Exec(ctx, `UPDATE document_collaboration SET epoch=$2,sequence=0,state='',snapshot_sequence=0,snapshot_at=now(),projected_markdown=$3,document_version=$4,reset_reason=$5,head_hash='',updated_at=now() WHERE document_id=$1`, id, epoch, markdown, version, reason); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `DELETE FROM collaboration_updates WHERE document_id=$1`, id)
	return err
}

// A restored database is a new collaboration branch, even when restored IDs,
// sequence numbers and Markdown happen to equal an active pre-restore client.
// Canonical content and immutable history are deliberately left untouched.
func invalidateCollaborationRestoreTx(ctx context.Context, tx pgx.Tx) error {
	if _, err := tx.Exec(ctx, `UPDATE document_collaboration c SET epoch=gen_random_uuid(),sequence=0,state='',snapshot_sequence=0,snapshot_at=now(),projected_markdown=d.markdown,document_version=d.version,reset_reason='backup_restored',head_hash='',updated_at=now() FROM documents d WHERE d.id=c.document_id`); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `DELETE FROM collaboration_updates`)
	return err
}

func collaborationPersistTx(ctx context.Context, tx pgx.Tx, id, epoch string, sequence, snapshotSequence int64, snapshotAt time.Time, version int, actor string, delta, state []byte, markdown string) error {
	if _, err := tx.Exec(ctx, `INSERT INTO collaboration_updates(document_id,epoch,sequence,update_data,document_version,actor_id) VALUES($1,$2,$3,$4,$5,$6)`, id, epoch, sequence, delta, version, actor); err != nil {
		return err
	}
	var tailBytes int64
	if err := tx.QueryRow(ctx, `SELECT coalesce(sum(octet_length(update_data)),0) FROM collaboration_updates WHERE document_id=$1 AND epoch=$2 AND sequence>$3`, id, epoch, snapshotSequence).Scan(&tailBytes); err != nil {
		return err
	}
	checkpoint := snapshotSequence == 0 || sequence-snapshotSequence >= collaborationCheckpointUpdates || tailBytes >= collaborationCheckpointBytes || time.Since(snapshotAt) >= time.Minute
	if checkpoint {
		if _, err := tx.Exec(ctx, `UPDATE document_collaboration SET state=$2,snapshot_sequence=$3,snapshot_at=now(),sequence=$3,projected_markdown=$4,document_version=$5,head_hash=$6,updated_at=now() WHERE document_id=$1`, id, state, sequence, markdown, version, digest(string(state))); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `DELETE FROM collaboration_updates WHERE document_id=$1 AND epoch=$2 AND sequence<=$3`, id, epoch, sequence)
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE document_collaboration SET sequence=$2,projected_markdown=$3,document_version=$4,head_hash=$5,updated_at=now() WHERE document_id=$1`, id, sequence, markdown, version, digest(string(state)))
	return err
}

// User-facing activity groups never replace immutable individual snapshots or
// reduce documents.version (which remains the compare-and-swap content version).
func collaborationGroupHistoryTx(ctx context.Context, tx pgx.Tx, id, epoch, actor string, version int) error {
	var group string
	err := tx.QueryRow(ctx, `SELECT id::text FROM collaboration_history_groups WHERE document_id=$1 ORDER BY last_version DESC LIMIT 1`, id).Scan(&group)
	if err != nil && err != pgx.ErrNoRows {
		return err
	}
	if group != "" {
		tag, e := tx.Exec(ctx, `UPDATE collaboration_history_groups SET last_version=$2,updated_at=now() WHERE id=$1 AND actor_id=$3 AND epoch=$4 AND last_version=$2-1 AND updated_at>now()-interval '30 seconds' AND started_at>now()-interval '5 minutes'`, group, version, actor, epoch)
		if e != nil {
			return e
		}
		if tag.RowsAffected() == 1 {
			return nil
		}
	}
	_, err = tx.Exec(ctx, `INSERT INTO collaboration_history_groups(id,document_id,actor_id,epoch,first_version,last_version) VALUES($1,$2,$3,$4,$5,$5)`, newID(), id, actor, epoch, version)
	return err
}
