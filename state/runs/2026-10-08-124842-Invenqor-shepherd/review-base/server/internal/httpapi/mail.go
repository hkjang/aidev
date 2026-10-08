package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hkjang/invenqor/server/internal/auth"
	"github.com/hkjang/invenqor/server/internal/mail"
)

// mailPasswordPurpose is the seal purpose for mail.password. It matches what
// the generic settings endpoint would use, so the row is readable either way;
// the generic endpoint is still refused the key so a plaintext row can never
// be written by mistake.
const mailPasswordPurpose = "setting:" + mail.KeyPassword

// loadMailConfig reads every mail.* row. A fresh installation has none and
// gets the default, which is off; nothing is written by a read.
func (s *Server) loadMailConfig(ctx context.Context) (mail.Config, error) {
	rows, err := s.database.DB().QueryContext(ctx,
		`SELECT key,value_json,secret FROM settings WHERE key LIKE 'mail.%'`)
	if err != nil {
		return mail.Config{}, fmt.Errorf("load mail settings: %w", err)
	}
	defer rows.Close()
	values := map[string]any{}
	for rows.Next() {
		var key string
		var raw any
		var secret bool
		if err := rows.Scan(&key, &raw, &secret); err != nil {
			return mail.Config{}, fmt.Errorf("load mail settings: %w", err)
		}
		encoded := valueString(raw)
		if key == mail.KeyPassword {
			// A password row that is not sealed is not trusted; it cannot be
			// written by this Server and would be plaintext in the database.
			if !secret || s.bootstrapStore == nil {
				continue
			}
			var envelope struct {
				Ciphertext string `json:"ciphertext"`
			}
			if json.Unmarshal([]byte(encoded), &envelope) != nil || envelope.Ciphertext == "" {
				continue
			}
			password, err := s.bootstrapStore.OpenString(mailPasswordPurpose, envelope.Ciphertext)
			if err != nil {
				return mail.Config{}, fmt.Errorf("open mail password: %w", err)
			}
			values[key] = password
			continue
		}
		var value any
		if json.Unmarshal([]byte(encoded), &value) != nil {
			continue
		}
		values[key] = value
	}
	if err := rows.Err(); err != nil {
		return mail.Config{}, fmt.Errorf("load mail settings: %w", err)
	}
	return mail.FromValues(values).Normalize(), nil
}

// saveMailConfig writes the mail.* rows in one transaction. The password is
// written only when the caller supplies one: nil keeps the stored value, an
// empty string clears it.
func (s *Server) saveMailConfig(
	ctx context.Context, updatedBy string, config mail.Config, password *string,
) error {
	if password != nil && *password != "" && s.bootstrapStore == nil {
		return errors.New("secret storage is unavailable")
	}
	tx, err := s.database.DB().BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	now := time.Now().UTC()
	upsert := func(key string, value string, secret bool) error {
		_, err := tx.ExecContext(ctx,
			`INSERT INTO settings(
			 key,value_json,secret,apply_mode,pending_value_json,version,
			 updated_by,updated_at
			 ) VALUES($1,$2,$3,'immediate',NULL,1,$4,$5)
			 ON CONFLICT(key) DO UPDATE SET
			 value_json=excluded.value_json,secret=excluded.secret,
			 apply_mode='immediate',pending_value_json=NULL,
			 version=settings.version+1,updated_by=excluded.updated_by,
			 updated_at=excluded.updated_at`,
			key, value, secret, updatedBy, now)
		return err
	}
	for key, value := range config.Values() {
		encoded, err := json.Marshal(value)
		if err != nil {
			return err
		}
		if err := upsert(key, string(encoded), false); err != nil {
			return fmt.Errorf("save %s: %w", key, err)
		}
	}
	if password != nil {
		if *password == "" {
			if _, err := tx.ExecContext(ctx,
				`DELETE FROM settings WHERE key=$1`, mail.KeyPassword); err != nil {
				return fmt.Errorf("clear %s: %w", mail.KeyPassword, err)
			}
		} else {
			sealed, err := s.bootstrapStore.SealString(mailPasswordPurpose, *password)
			if err != nil {
				return fmt.Errorf("seal %s: %w", mail.KeyPassword, err)
			}
			envelope, _ := json.Marshal(map[string]string{"ciphertext": sealed})
			if err := upsert(mail.KeyPassword, string(envelope), true); err != nil {
				return fmt.Errorf("save %s: %w", mail.KeyPassword, err)
			}
		}
	}
	return tx.Commit()
}

// userDirectory is the one lookup mail borrows from the users table: account
// id to address. Inactive and deleted accounts resolve to nothing.
type userDirectory struct{ db *sql.DB }

func (d userDirectory) LookupEmails(ctx context.Context, ids []string) (map[string]string, error) {
	addresses := make(map[string]string, len(ids))
	for _, id := range ids {
		var email string
		err := d.db.QueryRowContext(ctx,
			`SELECT email FROM users
			  WHERE id=$1 AND active=TRUE AND deleted_at IS NULL`, id).Scan(&email)
		if errors.Is(err, sql.ErrNoRows) {
			continue
		}
		if err != nil {
			return nil, err
		}
		if email = strings.TrimSpace(email); email != "" {
			addresses[id] = email
		}
	}
	return addresses, nil
}

// superAdminIDs is the audience for events that need an administrator's
// hand. Addresses are resolved later, so an administrator without one is
// simply skipped.
func (s *Server) superAdminIDs(ctx context.Context) []string {
	rows, err := s.database.DB().QueryContext(ctx,
		`SELECT id FROM users
		  WHERE super_admin=TRUE AND active=TRUE AND deleted_at IS NULL
		  ORDER BY username`)
	if err != nil {
		s.logger.Warn("mail_audience_unavailable", "error", err.Error())
		return nil
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return ids
		}
		ids = append(ids, id)
	}
	return ids
}

// notifyMail sends one event mail when the mailer is configured. Every
// caller is on a request path, so nothing here can fail the request.
func (s *Server) notifyMail(
	ctx context.Context, notification mail.Notification, actorID string, recipients []string,
) {
	if s.mail == nil || len(recipients) == 0 {
		return
	}
	s.mail.Notify(ctx, notification, actorID, recipients)
}

// actorLabel is the name people recognise in a mail body.
func actorLabel(request *http.Request) string {
	user := principalFromContext(request.Context()).User
	if name := strings.TrimSpace(user.DisplayName); name != "" {
		return name
	}
	if user.Username != "" {
		return user.Username
	}
	return "알 수 없는 사용자"
}

// onAccountLocked is the auth service hook. It runs on a failed login, so
// there is no principal and no actor to exclude; the owner and every
// administrator are told.
func (s *Server) onAccountLocked(ctx context.Context, user auth.User, sourceIP string, until time.Time) {
	recipients := append([]string{user.ID}, s.superAdminIDs(ctx)...)
	s.notifyMail(ctx, mail.AccountLocked(user.Username, user.DisplayName, sourceIP, until), "", recipients)
}

func (s *Server) getMailSettings(response http.ResponseWriter, request *http.Request) {
	config, err := s.loadMailConfig(request.Context())
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, config.Public())
}

func (s *Server) updateMailSettings(response http.ResponseWriter, request *http.Request) {
	var input struct {
		Enabled        *bool            `json:"enabled"`
		SMTPHost       *string          `json:"smtp_host"`
		SMTPPort       *int             `json:"smtp_port"`
		Security       *string          `json:"security"`
		SkipTLSVerify  *bool            `json:"skip_tls_verify"`
		Username       *string          `json:"username"`
		Password       *string          `json:"password"`
		FromAddress    *string          `json:"from_address"`
		FromName       *string          `json:"from_name"`
		BaseURL        *string          `json:"base_url"`
		TimeoutSeconds *int             `json:"timeout_seconds"`
		Events         *map[string]bool `json:"events"`
		Reason         string           `json:"reason"`
	}
	if decodeJSON(request, &input) != nil {
		writeAPIError(response, request, http.StatusBadRequest,
			"INVALID_MAIL_SETTINGS", "The request body is invalid.")
		return
	}
	before, err := s.loadMailConfig(request.Context())
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	after := before
	after.Events = map[string]bool{}
	for event, enabled := range before.Events {
		after.Events[event] = enabled
	}
	assign := func(target *string, value *string) {
		if value != nil {
			*target = *value
		}
	}
	if input.Enabled != nil {
		after.Enabled = *input.Enabled
	}
	assign(&after.Host, input.SMTPHost)
	if input.SMTPPort != nil {
		after.Port = *input.SMTPPort
	}
	assign(&after.Security, input.Security)
	if input.SkipTLSVerify != nil {
		after.SkipVerify = *input.SkipTLSVerify
	}
	assign(&after.Username, input.Username)
	assign(&after.FromAddress, input.FromAddress)
	assign(&after.FromName, input.FromName)
	assign(&after.BaseURL, input.BaseURL)
	if input.TimeoutSeconds != nil {
		after.Timeout = time.Duration(*input.TimeoutSeconds) * time.Second
	}
	if input.Events != nil {
		for event, enabled := range *input.Events {
			if _, known := mail.EventSettings[event]; !known {
				writeAPIError(response, request, http.StatusBadRequest,
					"INVALID_MAIL_SETTINGS", fmt.Sprintf("unknown event %q", event))
				return
			}
			after.Events[event] = enabled
		}
	}
	if input.Password != nil {
		after.Password = *input.Password
	}
	after = after.Normalize()
	if err := after.Validate(); err != nil {
		writeAPIError(response, request, http.StatusBadRequest,
			"INVALID_MAIL_SETTINGS", err.Error())
		return
	}
	principal := principalFromContext(request.Context())
	if err := s.saveMailConfig(request.Context(), principal.User.ID, after, input.Password); err != nil {
		s.internalError(response, request, err)
		return
	}
	s.recordAdminAudit(
		request, "settings.mail.update", "setting", mail.KeyPrefix+"*",
		before.Public(), after.Public(), input.Reason,
	)
	writeJSON(response, http.StatusOK, after.Public())
}

// sendTestMail proves the saved relay settings work before anything depends
// on them. The recipient defaults to the administrator's own address.
func (s *Server) sendTestMail(response http.ResponseWriter, request *http.Request) {
	var input struct {
		Recipient string `json:"recipient"`
	}
	if decodeJSON(request, &input) != nil {
		writeAPIError(response, request, http.StatusBadRequest,
			"INVALID_MAIL_TEST", "The request body is invalid.")
		return
	}
	principal := principalFromContext(request.Context())
	recipient := strings.TrimSpace(input.Recipient)
	if recipient == "" {
		recipient = strings.TrimSpace(principal.User.Email)
	}
	if !mail.IsAddress(recipient) {
		writeAPIError(response, request, http.StatusBadRequest,
			"INVALID_MAIL_TEST", "A recipient address is required.")
		return
	}
	if s.mail == nil {
		writeAPIError(response, request, http.StatusServiceUnavailable,
			"MAIL_UNAVAILABLE", "The mail service is not configured.")
		return
	}
	err := s.mail.SendNow(request.Context(), mail.TestMessage(), principal.User.ID, recipient)
	s.recordAdminAudit(
		request, "settings.mail.test", "setting", mail.KeyPrefix+"*", nil,
		map[string]any{"recipient": recipient, "sent": err == nil}, "",
	)
	if err != nil {
		status := http.StatusBadGateway
		if errors.Is(err, mail.ErrDisabled) || errors.Is(err, mail.ErrInvalid) {
			status = http.StatusBadRequest
		}
		writeJSON(response, status, map[string]any{
			"sent": false, "recipient": recipient, "error": err.Error(),
		})
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{"sent": true, "recipient": recipient})
}

func (s *Server) listMailDeliveries(response http.ResponseWriter, request *http.Request) {
	if s.mail == nil {
		writeJSON(response, http.StatusOK, mail.Page{
			Items: []mail.Delivery{}, Summary: mail.Summary{Status: map[string]int{}},
		})
		return
	}
	page, err := s.mail.Deliveries(
		request.Context(), request.URL.Query().Get("status"),
		queryInt(request, "limit", 50, 1, 200),
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	writeJSON(response, http.StatusOK, page)
}
