package auth

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/invenqor/server/internal/audit"
)

type ServiceOptions struct {
	LockoutThreshold int
	LockoutDuration  time.Duration
	IPRateThreshold  int
	IPRateWindow     time.Duration
	SessionIdle      time.Duration
	SessionMaximum   time.Duration
	TOTP             *TOTPService
}

func DefaultServiceOptions() ServiceOptions {
	return ServiceOptions{
		LockoutThreshold: 5,
		LockoutDuration:  15 * time.Minute,
		IPRateThreshold:  20,
		IPRateWindow:     15 * time.Minute,
		SessionIdle:      30 * time.Minute,
		SessionMaximum:   12 * time.Hour,
	}
}

type Service struct {
	db        *sql.DB
	options   ServiceOptions
	policy    PasswordPolicy
	dummyHash string
	audit     audit.Recorder
	totp      *TOTPService
	// locked is told when repeated failures lock an account. The mail
	// notifier hangs here so the auth package stays unaware of mail.
	locked func(ctx context.Context, user User, sourceIP string, until time.Time)
}

// SetAccountLockedHook registers the function called when a login failure
// locks an account. It runs on the login request path and must not block.
func (service *Service) SetAccountLockedHook(
	hook func(ctx context.Context, user User, sourceIP string, until time.Time),
) {
	service.locked = hook
}

func (service *Service) noteLocked(ctx context.Context, user User, sourceIP string, now time.Time) {
	if service.locked != nil {
		service.locked(ctx, user, sourceIP, now.Add(service.options.LockoutDuration))
	}
}

type databaseUser struct {
	User
	PasswordHash     sql.NullString
	Active           bool
	FailedLoginCount int
	LockedUntil      flexibleTime
}

func NewService(db *sql.DB, options ServiceOptions) (*Service, error) {
	if options.LockoutThreshold <= 0 ||
		options.LockoutDuration <= 0 ||
		options.IPRateThreshold <= 0 ||
		options.IPRateWindow <= 0 ||
		options.SessionIdle <= 0 ||
		options.SessionMaximum <= 0 {
		return nil, errors.New("authentication service durations and thresholds must be positive")
	}
	dummyHash, err := HashPassword("Invenqor-Dummy-Password!42")
	if err != nil {
		return nil, err
	}
	return &Service{
		db:        db,
		options:   options,
		policy:    DefaultPasswordPolicy(),
		dummyHash: dummyHash,
		totp:      options.TOTP,
	}, nil
}

func (service *Service) Authenticate(
	ctx context.Context,
	input LoginInput,
	sourceIP string,
	userAgent string,
	requestID string,
) (Session, error) {
	normalized := normalizeUsername(input.Username)
	blocked, err := service.ipRateLimited(ctx, sourceIP, time.Now().UTC())
	if err != nil {
		return Session{}, err
	}
	if blocked {
		service.recordLoginAudit(ctx, nil, input.Username, sourceIP, userAgent, requestID, "rate_limited")
		return Session{}, ErrAccountLocked
	}
	user, err := service.findUserByUsername(ctx, normalized)
	if errors.Is(err, sql.ErrNoRows) {
		_, _ = VerifyPassword(input.Password, service.dummyHash)
		_ = service.recordLoginAttempt(ctx, normalized, sourceIP, false, time.Now().UTC())
		service.recordLoginAudit(ctx, nil, input.Username, sourceIP, userAgent, requestID, "failure")
		return Session{}, ErrInvalidCredentials
	}
	if err != nil {
		return Session{}, err
	}
	now := time.Now().UTC()
	if !user.Active {
		service.recordLoginAudit(ctx, &user, user.Username, sourceIP, userAgent, requestID, "inactive")
		return Session{}, ErrAccountInactive
	}
	if user.LockedUntil.Valid && now.Before(user.LockedUntil.Time) {
		service.recordLoginAudit(ctx, &user, user.Username, sourceIP, userAgent, requestID, "locked")
		return Session{}, ErrAccountLocked
	}
	valid := false
	if user.PasswordHash.Valid {
		valid, err = VerifyPassword(input.Password, user.PasswordHash.String)
		if err != nil {
			return Session{}, fmt.Errorf("verify password hash: %w", err)
		}
	} else {
		_, _ = VerifyPassword(input.Password, service.dummyHash)
	}
	if !valid {
		if err := service.recordLoginAttempt(ctx, normalized, sourceIP, false, now); err != nil {
			return Session{}, err
		}
		if err := service.recordLoginFailure(ctx, user, now); err != nil {
			return Session{}, err
		}
		service.recordLoginAudit(ctx, &user, user.Username, sourceIP, userAgent, requestID, "failure")
		if user.FailedLoginCount+1 >= service.options.LockoutThreshold {
			service.noteLocked(ctx, user.User, sourceIP, now)
			return Session{}, ErrAccountLocked
		}
		return Session{}, ErrInvalidCredentials
	}
	if service.totp != nil {
		enabled, err := service.totp.Enabled(ctx, user.ID)
		if err != nil {
			return Session{}, err
		}
		if enabled {
			if strings.TrimSpace(input.TOTPCode) == "" {
				service.recordLoginAudit(ctx, &user, user.Username, sourceIP, userAgent, requestID, "mfa_required")
				return Session{}, ErrMFARequired
			}
			if err := service.totp.Verify(ctx, user.ID, input.TOTPCode); err != nil {
				// A wrong second factor is a failed login. Without recording it
				// here, neither the lockout counter nor the per-IP rate limit
				// saw the attempt, which left the second factor open to
				// unlimited guessing once a password was known.
				if err := service.recordLoginAttempt(ctx, normalized, sourceIP, false, now); err != nil {
					return Session{}, err
				}
				if err := service.recordLoginFailure(ctx, user, now); err != nil {
					return Session{}, err
				}
				service.recordLoginAudit(ctx, &user, user.Username, sourceIP, userAgent, requestID, "mfa_failure")
				if user.FailedLoginCount+1 >= service.options.LockoutThreshold {
					service.noteLocked(ctx, user.User, sourceIP, now)
					return Session{}, ErrAccountLocked
				}
				return Session{}, ErrMFAInvalid
			}
		}
	}
	if _, err := service.db.ExecContext(
		ctx,
		`UPDATE users
		 SET failed_login_count = 0, locked_until = NULL, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $1`,
		user.ID,
	); err != nil {
		return Session{}, fmt.Errorf("reset login failures: %w", err)
	}
	if err := service.recordLoginAttempt(ctx, normalized, sourceIP, true, now); err != nil {
		return Session{}, err
	}
	user.Roles, user.Permissions, err = service.rolesAndPermissions(ctx, user.ID)
	if err != nil {
		return Session{}, err
	}
	session, err := service.createSession(ctx, user.User, sourceIP, userAgent, now)
	if err != nil {
		return Session{}, err
	}
	service.recordLoginAudit(ctx, &user, user.Username, sourceIP, userAgent, requestID, "success")
	return session, nil
}

func (service *Service) PrincipalByToken(
	ctx context.Context,
	token string,
) (Principal, error) {
	if token == "" {
		return Principal{}, ErrUnauthorized
	}
	tokenHash := hashSecret(token)
	var principal Principal
	var active bool
	var revoked flexibleTime
	var storedIdleExpires flexibleTime
	var storedAbsoluteExpires flexibleTime
	err := service.db.QueryRowContext(
		ctx,
		`SELECT
			s.id, s.csrf_hash, s.idle_expires_at, s.absolute_expires_at, s.revoked_at,
			u.id, u.username, u.display_name, u.email, u.super_admin, u.active
		 FROM sessions s
		 JOIN users u ON u.id = s.user_id
		 WHERE s.token_hash = $1 AND u.deleted_at IS NULL`,
		tokenHash,
	).Scan(
		&principal.SessionID,
		&principal.CSRFHash,
		&storedIdleExpires,
		&storedAbsoluteExpires,
		&revoked,
		&principal.User.ID,
		&principal.User.Username,
		&principal.User.DisplayName,
		&principal.User.Email,
		&principal.User.SuperAdmin,
		&active,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Principal{}, ErrUnauthorized
	}
	if err != nil {
		return Principal{}, fmt.Errorf("load session: %w", err)
	}
	if !storedIdleExpires.Valid || !storedAbsoluteExpires.Valid {
		return Principal{}, ErrUnauthorized
	}
	principal.IdleExpiresAt = storedIdleExpires.Time
	principal.AbsoluteExpiresAt = storedAbsoluteExpires.Time
	now := time.Now().UTC()
	if !active || revoked.Valid ||
		!now.Before(principal.IdleExpiresAt) ||
		!now.Before(principal.AbsoluteExpiresAt) {
		return Principal{}, ErrUnauthorized
	}
	principal.User.Roles, principal.User.Permissions, err =
		service.rolesAndPermissions(ctx, principal.User.ID)
	if err != nil {
		return Principal{}, err
	}
	idleExpires := now.Add(service.options.SessionIdle)
	if idleExpires.After(principal.AbsoluteExpiresAt) {
		idleExpires = principal.AbsoluteExpiresAt
	}
	if _, err := service.db.ExecContext(
		ctx,
		`UPDATE sessions
		 SET last_seen_at = $1, idle_expires_at = $2
		 WHERE id = $3`,
		now,
		idleExpires,
		principal.SessionID,
	); err != nil {
		return Principal{}, fmt.Errorf("refresh session: %w", err)
	}
	principal.IdleExpiresAt = idleExpires
	return principal, nil
}

func (service *Service) VerifyCSRF(principal Principal, token string) error {
	if token == "" {
		return ErrCSRF
	}
	actual := hashSecret(token)
	if subtle.ConstantTimeCompare([]byte(actual), []byte(principal.CSRFHash)) != 1 {
		return ErrCSRF
	}
	return nil
}

func (service *Service) RevokeSession(
	ctx context.Context,
	principal Principal,
	sourceIP string,
	userAgent string,
	requestID string,
) error {
	if _, err := service.db.ExecContext(
		ctx,
		"UPDATE sessions SET revoked_at = CURRENT_TIMESTAMP WHERE id = $1",
		principal.SessionID,
	); err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return service.audit.Record(ctx, service.db, audit.Entry{
		ActorType:    "user",
		ActorID:      principal.User.ID,
		ActorName:    principal.User.Username,
		Action:       "auth.logout",
		ResourceType: "session",
		ResourceID:   principal.SessionID,
		RequestID:    requestID,
		SourceIP:     sourceIP,
		UserAgent:    userAgent,
		Result:       "success",
	})
}

func (service *Service) ChangePassword(
	ctx context.Context,
	principal Principal,
	input ChangePasswordInput,
	sourceIP string,
	userAgent string,
	requestID string,
) error {
	if input.CurrentPassword == input.NewPassword {
		return ErrPasswordUnchanged
	}
	if err := service.policy.Validate(input.NewPassword); err != nil {
		return err
	}
	// An account provisioned through Keycloak holds no local password, so the
	// column is NULL. Reading it into a plain string turned an ordinary
	// situation into a scan failure and a 500.
	var currentHash sql.NullString
	if err := service.db.QueryRowContext(
		ctx,
		"SELECT password_hash FROM users WHERE id = $1 AND active = TRUE AND deleted_at IS NULL",
		principal.User.ID,
	).Scan(&currentHash); errors.Is(err, sql.ErrNoRows) {
		return ErrUnauthorized
	} else if err != nil {
		return fmt.Errorf("load current password: %w", err)
	}
	if !currentHash.Valid || currentHash.String == "" {
		return ErrPasswordUnavailable
	}
	valid, err := VerifyPassword(input.CurrentPassword, currentHash.String)
	if err != nil {
		return fmt.Errorf("verify current password: %w", err)
	}
	if !valid {
		return ErrInvalidCredentials
	}
	newHash, err := HashPassword(input.NewPassword)
	if err != nil {
		return err
	}
	transaction, err := service.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin password change: %w", err)
	}
	defer transaction.Rollback()
	if _, err := transaction.ExecContext(
		ctx,
		`UPDATE users
		 SET password_hash = $1, password_changed_at = CURRENT_TIMESTAMP,
		     updated_at = CURRENT_TIMESTAMP
		 WHERE id = $2`,
		newHash,
		principal.User.ID,
	); err != nil {
		return fmt.Errorf("change password: %w", err)
	}
	// Every session but the caller's own. The exclusion is only added when the
	// caller actually has one: sessions.id is a UUID column, and a principal
	// without a session carries an empty SessionID while an API-key principal
	// carries "api_key:<id>" - neither is a UUID, and PostgreSQL rejects the
	// comparison outright (SQLSTATE 22P02) where SQLite silently matches nothing.
	// With no session of its own to keep, revoking all of the user's sessions is
	// also the right answer.
	revoke := `UPDATE sessions SET revoked_at = CURRENT_TIMESTAMP
		 WHERE user_id = $1 AND revoked_at IS NULL`
	arguments := []any{principal.User.ID}
	if _, err := uuid.Parse(principal.SessionID); err == nil {
		revoke += ` AND id <> $2`
		arguments = append(arguments, principal.SessionID)
	}
	if _, err := transaction.ExecContext(ctx, revoke, arguments...); err != nil {
		return fmt.Errorf("revoke other sessions: %w", err)
	}
	if err := service.audit.Record(ctx, transaction, audit.Entry{
		ActorType:    "user",
		ActorID:      principal.User.ID,
		ActorName:    principal.User.Username,
		Action:       "auth.password.change",
		ResourceType: "user",
		ResourceID:   principal.User.ID,
		RequestID:    requestID,
		SourceIP:     sourceIP,
		UserAgent:    userAgent,
		Result:       "success",
	}); err != nil {
		return err
	}
	if err := transaction.Commit(); err != nil {
		return fmt.Errorf("commit password change: %w", err)
	}
	return nil
}

// PasswordConfigured reports whether the account can change a local password at
// all. The console asks so it can explain a federated account rather than
// offering a form that must fail.
func (service *Service) PasswordConfigured(
	ctx context.Context,
	userID string,
) (bool, error) {
	var hash sql.NullString
	if err := service.db.QueryRowContext(
		ctx,
		"SELECT password_hash FROM users WHERE id = $1 AND deleted_at IS NULL",
		userID,
	).Scan(&hash); errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("read password state: %w", err)
	}
	return hash.Valid && hash.String != "", nil
}

func (service *Service) findUserByUsername(
	ctx context.Context,
	normalized string,
) (databaseUser, error) {
	var user databaseUser
	err := service.db.QueryRowContext(
		ctx,
		`SELECT
			id, username, display_name, email, password_hash, active,
			super_admin, failed_login_count, locked_until
		 FROM users
		 WHERE normalized_username = $1 AND deleted_at IS NULL`,
		normalized,
	).Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&user.Email,
		&user.PasswordHash,
		&user.Active,
		&user.SuperAdmin,
		&user.FailedLoginCount,
		&user.LockedUntil,
	)
	return user, err
}

func (service *Service) recordLoginFailure(
	ctx context.Context,
	user databaseUser,
	now time.Time,
) error {
	failures := user.FailedLoginCount + 1
	var lockedUntil any
	if failures >= service.options.LockoutThreshold {
		lockedUntil = now.Add(service.options.LockoutDuration)
	}
	if _, err := service.db.ExecContext(
		ctx,
		`UPDATE users
		 SET failed_login_count = $1, locked_until = $2, updated_at = CURRENT_TIMESTAMP
		 WHERE id = $3`,
		failures,
		lockedUntil,
		user.ID,
	); err != nil {
		return fmt.Errorf("record login failure: %w", err)
	}
	return nil
}

func (service *Service) ipRateLimited(
	ctx context.Context,
	sourceIP string,
	now time.Time,
) (bool, error) {
	var failures int
	if err := service.db.QueryRowContext(
		ctx,
		`SELECT COUNT(*)
		 FROM login_attempts
		 WHERE source_ip = $1 AND succeeded = FALSE AND occurred_at >= $2`,
		sourceIP,
		now.Add(-service.options.IPRateWindow),
	).Scan(&failures); err != nil {
		return false, fmt.Errorf("query login IP rate: %w", err)
	}
	return failures >= service.options.IPRateThreshold, nil
}

func (service *Service) recordLoginAttempt(
	ctx context.Context,
	normalizedUsername string,
	sourceIP string,
	succeeded bool,
	now time.Time,
) error {
	if _, err := service.db.ExecContext(
		ctx,
		`INSERT INTO login_attempts(
			id, normalized_username, source_ip, succeeded, occurred_at
		) VALUES ($1, $2, $3, $4, $5)`,
		uuid.NewString(),
		normalizedUsername,
		sourceIP,
		succeeded,
		now,
	); err != nil {
		return fmt.Errorf("record login attempt: %w", err)
	}
	return nil
}

func (service *Service) createSession(
	ctx context.Context,
	user User,
	sourceIP string,
	userAgent string,
	now time.Time,
) (Session, error) {
	token, _, err := newSecret()
	if err != nil {
		return Session{}, err
	}
	csrfToken, _, err := newSecret()
	if err != nil {
		return Session{}, err
	}
	absoluteExpires := now.Add(service.options.SessionMaximum)
	idleExpires := now.Add(service.options.SessionIdle)
	session := Session{
		ID:                uuid.NewString(),
		Token:             token,
		CSRFToken:         csrfToken,
		User:              user,
		IdleExpiresAt:     idleExpires,
		AbsoluteExpiresAt: absoluteExpires,
	}
	if _, err := service.db.ExecContext(
		ctx,
		`INSERT INTO sessions(
			id, user_id, token_hash, csrf_hash, source_ip, user_agent,
			last_seen_at, idle_expires_at, absolute_expires_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		session.ID,
		user.ID,
		hashSecret(token),
		hashSecret(csrfToken),
		sourceIP,
		userAgent,
		now,
		idleExpires,
		absoluteExpires,
	); err != nil {
		return Session{}, fmt.Errorf("create session: %w", err)
	}
	return session, nil
}

func (service *Service) rolesAndPermissions(
	ctx context.Context,
	userID string,
) ([]string, []string, error) {
	rows, err := service.db.QueryContext(
		ctx,
		`SELECT DISTINCT r.name
		 FROM roles r
		 JOIN user_roles ur ON ur.role_id = r.id
		 WHERE ur.user_id = $1
		 ORDER BY r.name`,
		userID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("query user roles: %w", err)
	}
	defer rows.Close()
	var roles []string
	for rows.Next() {
		var role string
		if err := rows.Scan(&role); err != nil {
			return nil, nil, fmt.Errorf("scan user role: %w", err)
		}
		roles = append(roles, role)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate user roles: %w", err)
	}
	permissionRows, err := service.db.QueryContext(
		ctx,
		`SELECT DISTINCT rp.permission_name
		 FROM role_permissions rp
		 JOIN user_roles ur ON ur.role_id = rp.role_id
		 WHERE ur.user_id = $1
		 ORDER BY rp.permission_name`,
		userID,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("query user permissions: %w", err)
	}
	defer permissionRows.Close()
	var permissions []string
	for permissionRows.Next() {
		var permission string
		if err := permissionRows.Scan(&permission); err != nil {
			return nil, nil, fmt.Errorf("scan user permission: %w", err)
		}
		permissions = append(permissions, permission)
	}
	if err := permissionRows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate user permissions: %w", err)
	}
	sort.Strings(roles)
	sort.Strings(permissions)
	return roles, permissions, nil
}

func (service *Service) recordLoginAudit(
	ctx context.Context,
	user *databaseUser,
	username string,
	sourceIP string,
	userAgent string,
	requestID string,
	result string,
) {
	entry := audit.Entry{
		ActorType:    "user",
		ActorName:    strings.TrimSpace(username),
		Action:       "auth.local.login",
		ResourceType: "session",
		RequestID:    requestID,
		SourceIP:     sourceIP,
		UserAgent:    userAgent,
		Result:       result,
	}
	if user != nil {
		entry.ActorID = user.ID
		entry.ActorName = user.Username
	}
	_ = service.audit.Record(ctx, service.db, entry)
}

func hashSecret(secret string) string {
	hash := sha256.Sum256([]byte(secret))
	return hex.EncodeToString(hash[:])
}
