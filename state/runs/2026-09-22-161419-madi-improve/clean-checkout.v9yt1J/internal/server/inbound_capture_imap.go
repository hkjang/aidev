package server

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/emersion/go-imap/v2"
	"github.com/emersion/go-imap/v2/imapclient"
	"github.com/jackc/pgx/v5"
)

func (s *Server) registerInboundCaptures() {
	s.admin("GET /api/v1/admin/inbound-capture-settings", func(w http.ResponseWriter, r *http.Request) {
		v, e := s.one(r.Context(), "SELECT to_jsonb(x) FROM inbound_capture_settings x WHERE id=1")
		respond(w, v, e)
	})
	s.admin("PUT /api/v1/admin/inbound-capture-settings", func(w http.ResponseWriter, r *http.Request) {
		var in map[string]any
		if decode(r, &in) != nil {
			apiError(w, 400, "수집 정책을 확인하세요")
			return
		}
		hosts, e := notificationHostList(in["allowed_hosts"])
		if e != nil {
			apiError(w, 400, e.Error())
			return
		}
		_, e = s.DB.Exec(r.Context(), "UPDATE inbound_capture_settings SET hooks_enabled=$1,imap_enabled=$2,allowed_hosts=$3 WHERE id=1", boolean(in, "hooks_enabled"), boolean(in, "imap_enabled"), jsonValue(hosts))
		if e == nil {
			s.audit(r, "INBOUND_POLICY", "", map[string]any{"hooks_enabled": boolean(in, "hooks_enabled"), "imap_enabled": boolean(in, "imap_enabled")})
		}
		respond(w, map[string]bool{"ok": e == nil}, e)
	})
	s.handle("GET /api/v1/capture-channels", s.listInboundChannels)
	s.handle("GET /api/v1/capture-policy", func(w http.ResponseWriter, r *http.Request) {
		if !notificationHuman(w, r) {
			return
		}
		v, e := s.one(r.Context(), "SELECT jsonb_build_object('hooks_enabled',hooks_enabled,'imap_enabled',imap_enabled) FROM inbound_capture_settings WHERE id=1")
		respond(w, v, e)
	})
	s.handle("POST /api/v1/capture-channels", s.saveInboundChannel)
	s.handle("PUT /api/v1/capture-channels/{id}", s.saveInboundChannel)
	s.handle("POST /api/v1/capture-channels/{id}/rotate", s.rotateInboundSecret)
	s.handle("GET /api/v1/capture-channels/{id}/history", s.inboundCaptureHistory)
	s.handle("POST /api/v1/capture-channels/{id}/poll", s.queueInboundPoll)
	s.handle("POST /api/v1/capture-channels/{id}/messages/{message}/cancel", s.cancelInboundMessage)
	s.handle("GET /api/v1/capture-channels/{id}/jobs", func(w http.ResponseWriter, r *http.Request) {
		c, ok := s.requestedInboundChannel(w, r)
		if !ok {
			return
		}
		v, e := s.rows(r.Context(), `SELECT jsonb_build_object('id',id,'status',status,'attempts',attempts,'last_error',last_error,'created_at',created_at,'updated_at',updated_at) FROM automation_jobs WHERE kind='capture.imap' AND owner_id=$1 AND payload->>'channel_id'=$2 ORDER BY created_at DESC LIMIT 20`, c.UserID, c.ID)
		respond(w, v, e)
	})
	s.apiRoutes = append(s.apiRoutes, "POST /api/v1/capture-hooks/{id}")
	s.mux.HandleFunc("POST /api/v1/capture-hooks/{id}", s.receiveCaptureHook)
	s.RegisterJobHandler("capture.ingest", s.ingestInboundCapture)
	s.RegisterJobHandler("capture.imap", s.pollInboundIMAP)
}
func (s *Server) inboundIMAPTLS(ctx context.Context, c inboundChannel) (*tls.Config, error) {
	var enabled bool
	var raw []byte
	if s.DB.QueryRow(ctx, "SELECT imap_enabled,allowed_hosts FROM inbound_capture_settings WHERE id=1").Scan(&enabled, &raw) != nil || !enabled {
		return nil, jobPermanent("관리자가 IMAP 수집을 중지했습니다")
	}
	var hosts []string
	if json.Unmarshal(raw, &hosts) != nil {
		return nil, errors.New("메일 호스트 정책을 읽지 못했습니다")
	}
	host := str(c.Config, "host")
	found := false
	for _, h := range hosts {
		if strings.EqualFold(h, host) {
			found = true
		}
	}
	if !found {
		return nil, jobPermanent("관리자가 허용하지 않은 메일 호스트입니다")
	}
	config := &tls.Config{MinVersion: tls.VersionTLS12, ServerName: host, InsecureSkipVerify: boolean(c.Config, "insecure_tls")}
	if pem := str(c.Config, "ca_pem"); pem != "" {
		pool, _ := x509.SystemCertPool()
		if pool == nil {
			pool = x509.NewCertPool()
		}
		if !pool.AppendCertsFromPEM([]byte(pem)) {
			return nil, jobPermanent("메일 CA 인증서를 확인하세요")
		}
		config.RootCAs = pool
	}
	return config, nil
}
func (s *Server) enqueueInboundPoll(ctx context.Context, tx pgx.Tx, c inboundChannel) (string, error) {
	var revision int
	var enabled bool
	if e := tx.QueryRow(ctx, "SELECT revision,enabled FROM inbound_capture_channels WHERE id=$1 FOR UPDATE", c.ID).Scan(&revision, &enabled); e != nil {
		return "", e
	}
	if !enabled || revision != c.Revision || c.Kind != "imap" {
		return "", errors.New("활성화된 IMAP 채널을 확인하세요")
	}
	var busy bool
	if e := tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM automation_jobs WHERE kind='capture.imap' AND payload->>'channel_id'=$1 AND status IN ('pending','running'))`, c.ID).Scan(&busy); e != nil {
		return "", e
	}
	if busy {
		return "", nil
	}
	id, e := s.EnqueueJob(ctx, tx, "capture.imap", c.UserID, c.WorkspaceID, map[string]any{"channel_id": c.ID, "revision": c.Revision})
	if e == nil {
		_, e = tx.Exec(ctx, "UPDATE automation_jobs SET timeout_seconds=180,max_attempts=3 WHERE id=$1", id)
	}
	return id, e
}
func (s *Server) queueInboundPoll(w http.ResponseWriter, r *http.Request) {
	c, ok := s.requestedInboundChannel(w, r)
	if !ok {
		return
	}
	if _, e := s.inboundOwner(r.Context(), c); e != nil {
		apiError(w, 403, e.Error())
		return
	}
	if _, e := s.inboundIMAPTLS(r.Context(), c); e != nil {
		apiError(w, 400, e.Error())
		return
	}
	tx, e := s.DB.Begin(r.Context())
	if e != nil {
		respond(w, nil, e)
		return
	}
	defer tx.Rollback(r.Context())
	id, e := s.enqueueInboundPoll(r.Context(), tx, c)
	if e == nil {
		e = tx.Commit(r.Context())
	}
	if e != nil {
		apiError(w, 409, e.Error())
		return
	}
	respond(w, map[string]any{"job_id": id, "already_running": id == ""}, nil)
}
func (s *Server) StartInboundCaptures(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()
		for {
			if e := s.scheduleInboundCaptures(ctx); e != nil && ctx.Err() == nil {
				slog.Error("inbound capture scheduler failed")
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
}
func (s *Server) scheduleInboundCaptures(ctx context.Context) error {
	var enabled, paused bool
	if e := s.DB.QueryRow(ctx, "SELECT f.imap_enabled,j.paused FROM inbound_capture_settings f,job_settings j WHERE f.id=1 AND j.id=1").Scan(&enabled, &paused); e != nil {
		return e
	}
	if paused {
		return nil
	}
	if _, e := s.DB.Exec(ctx, "UPDATE inbound_capture_messages SET payload_ciphertext='',payload_size=0 WHERE status<>'pending' AND completed_at<now()-interval '7 days' AND payload_ciphertext<>''"); e != nil {
		return e
	}
	if !enabled {
		return nil
	}
	rows, e := s.rows(ctx, "SELECT jsonb_build_object('id',id) FROM inbound_capture_channels WHERE kind='imap' AND enabled AND next_run<=now() ORDER BY next_run LIMIT 100")
	if e != nil {
		return e
	}
	for _, row := range rows {
		c, e := s.inboundChannel(ctx, str(row, "id"))
		if e != nil {
			continue
		}
		p, e := s.inboundOwner(ctx, c)
		if e != nil {
			continue
		}
		jobCtx := context.WithValue(ctx, principalKey, p)
		tx, e := s.DB.Begin(jobCtx)
		if e != nil {
			return e
		}
		_, e = s.enqueueInboundPoll(jobCtx, tx, c)
		if e == nil {
			_, e = tx.Exec(jobCtx, "UPDATE inbound_capture_channels SET next_run=now()+($2::text||' minutes')::interval WHERE id=$1", c.ID, number(c.Config, "interval_minutes", 5))
		}
		if e == nil {
			e = tx.Commit(jobCtx)
		} else {
			_ = tx.Rollback(jobCtx)
		}
		if e != nil {
			return e
		}
	}
	return nil
}
func (s *Server) pollInboundIMAP(ctx context.Context, j Job) (map[string]any, error) {
	c, e := s.inboundChannel(ctx, str(j.Payload, "channel_id"))
	if e != nil || c.Kind != "imap" || !c.Enabled || c.Revision != number(j.Payload, "revision", 0) || c.UserID != j.OwnerID {
		return nil, jobPermanent("메일 수집 채널 설정이 변경되었습니다")
	}
	if _, e = s.inboundOwner(ctx, c); e != nil {
		return nil, e
	}
	tlsConfig, e := s.inboundIMAPTLS(ctx, c)
	if e != nil {
		return nil, e
	}
	conn, e := webhookDial(ctx, "tcp", net.JoinHostPort(str(c.Config, "host"), strconv.Itoa(number(c.Config, "port", 993))))
	if e != nil {
		return nil, errors.New("IMAP 서버에 연결하지 못했습니다")
	}
	defer conn.Close()
	_ = conn.SetDeadline(time.Now().Add(120 * time.Second))
	stop := context.AfterFunc(ctx, func() { conn.Close() })
	defer stop()
	secure := tls.Client(conn, tlsConfig)
	if e = secure.HandshakeContext(ctx); e != nil {
		return nil, errors.New("IMAP TLS 인증서 검증에 실패했습니다")
	}
	client := imapclient.New(secure, &imapclient.Options{})
	defer client.Close()
	if e = client.Login(str(c.Secrets, "username"), str(c.Secrets, "password")).Wait(); e != nil {
		return nil, jobPermanent("IMAP 인증에 실패했습니다. 비밀번호 또는 앱 암호를 확인하세요")
	}
	selected, e := client.Select(str(c.Config, "mailbox"), &imap.SelectOptions{ReadOnly: true}).Wait()
	if e != nil || selected.UIDValidity == 0 || selected.UIDNext == 0 {
		return nil, jobPermanent("읽기 전용 IMAP 폴더의 UID 정보를 읽지 못했습니다")
	}
	validity := int(selected.UIDValidity)
	last := number(c.Checkpoint, "last_uid", 0)
	oldValidity := number(c.Checkpoint, "uid_validity", 0)
	if oldValidity != validity {
		last = 0
		if oldValidity == 0 && boolean(c.Config, "from_now") {
			last = int(selected.UIDNext) - 1
		}
	}
	advance := func(uid int) error {
		return s.advanceInboundCheckpoint(ctx, c, validity, uid, oldValidity)
	}
	if last >= int(selected.UIDNext)-1 {
		if e = advance(last); e != nil {
			return nil, e
		}
		return map[string]any{"staged": 0, "last_uid": last, "readonly": true}, nil
	}
	upper := min(last+100, int(selected.UIDNext)-1)
	set := imap.UIDSet{}
	set.AddRange(imap.UID(last+1), imap.UID(upper))
	metadata, e := client.Fetch(set, &imap.FetchOptions{UID: true, RFC822Size: true}).Collect()
	if e != nil {
		return nil, errors.New("IMAP 메시지 크기 목록을 읽지 못했습니다")
	}
	sort.Slice(metadata, func(i, k int) bool { return metadata[i].UID < metadata[k].UID })
	staged, rejected, bytesRead := 0, 0, 0
	processed := last
	for index, item := range metadata {
		uid := int(item.UID)
		if uid <= last || uid > upper {
			continue
		}
		if index >= 20 || bytesRead >= 20<<20 {
			break
		}
		var current bool
		if e = s.DB.QueryRow(ctx, `SELECT c.enabled AND c.revision=$2 AND NOT u.disabled AND u.kind='user' AND u.role<>'viewer' AND m.role IN ('owner','admin','editor') AND f.imap_enabled FROM inbound_capture_channels c JOIN users u ON u.id=c.user_id JOIN workspace_members m ON m.user_id=c.user_id AND m.workspace_id=c.workspace_id JOIN inbound_capture_settings f ON f.id=1 WHERE c.id=$1`, c.ID, c.Revision).Scan(&current); e != nil || !current {
			return nil, jobPermanent("메일 검사 중 채널 정책 또는 작성 권한이 변경되었습니다")
		}
		key := fmt.Sprintf("imap:%d:%d", validity, uid)
		reject := func(message string) error {
			_, e := s.DB.Exec(ctx, "INSERT INTO inbound_capture_messages(id,channel_id,message_key,payload_hash,channel_revision,status,message,completed_at) VALUES($1,$2,$3,$4,$5,'rejected',$6,now()) ON CONFLICT(channel_id,message_key) DO NOTHING", newID(), c.ID, key, digest(key), c.Revision, message)
			return e
		}
		if item.RFC822Size <= 0 || item.RFC822Size > inboundCaptureMax {
			if e = reject("메일 원문이 10MB 제한을 초과하거나 크기가 올바르지 않습니다"); e != nil {
				return nil, e
			}
			rejected++
			processed = uid
			continue
		}
		if bytesRead+int(item.RFC822Size) > 20<<20 {
			break
		}
		part := &imap.FetchItemBodySection{Peek: true, Partial: &imap.SectionPartial{Offset: 0, Size: inboundCaptureMax + 1}}
		command := client.Fetch(imap.UIDSetNum(item.UID), &imap.FetchOptions{UID: true, BodySection: []*imap.FetchItemBodySection{part}})
		var raw []byte
		for message := command.Next(); message != nil; message = command.Next() {
			for data := message.Next(); data != nil; data = message.Next() {
				if body, ok := data.(imapclient.FetchItemDataBodySection); ok {
					raw, e = io.ReadAll(io.LimitReader(body.Literal, inboundCaptureMax+1))
					if e != nil || len(raw) > inboundCaptureMax {
						_ = client.Close()
						return nil, jobPermanent("IMAP 서버가 허용된 메일 크기를 초과했습니다")
					}
				}
			}
		}
		if e = command.Close(); e != nil {
			return nil, errors.New("IMAP 메일 본문을 읽지 못했습니다")
		}
		bytesRead += len(raw)
		content, e := parseInboundMIME(raw)
		if e != nil {
			if e = reject(e.Error()); e != nil {
				return nil, e
			}
			rejected++
			processed = uid
			continue
		}
		if content.MessageID == "" {
			content.MessageID = "sha256:" + digest(string(raw))
		}
		_, duplicate, e := s.stageInboundContent(ctx, c, key, content)
		if e != nil {
			return nil, e
		}
		if !duplicate {
			staged++
		}
		processed = uid
	}
	if len(metadata) == 0 {
		processed = upper
	}
	if e = advance(processed); e != nil {
		return nil, e
	}
	_ = client.Logout().Wait()
	return map[string]any{"staged": staged, "rejected": rejected, "last_uid": processed, "readonly": true}, nil
}

func (s *Server) advanceInboundCheckpoint(ctx context.Context, c inboundChannel, validity, uid, oldValidity int) error {
	// Lease recovery can briefly overlap an old worker. Never rewind a UID
	// checkpoint within the same mailbox generation or replace a newer epoch.
	tag, e := s.DB.Exec(ctx, `UPDATE inbound_capture_channels SET checkpoint=jsonb_build_object('uid_validity',$2::bigint,'last_uid',CASE WHEN coalesce((checkpoint->>'uid_validity')::bigint,0)=$2 THEN greatest(coalesce((checkpoint->>'last_uid')::bigint,0),$3::bigint) ELSE $3::bigint END),updated_at=now() WHERE id=$1 AND revision=$4 AND enabled AND coalesce((checkpoint->>'uid_validity')::bigint,0) IN ($2,$5)`, c.ID, validity, uid, c.Revision, oldValidity)
	if e == nil && tag.RowsAffected() != 1 {
		return jobPermanent("메일 검사 중 채널 설정이 변경되었습니다")
	}
	return e
}
