package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/mail"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/hkjang/invenqor/server/internal/audit"
)

const bootstrapTokenFile = "initial-admin.token"

type BootstrapManager struct {
	db       *sql.DB
	stateDir string
	policy   PasswordPolicy
	audit    audit.Recorder
	mutex    sync.Mutex
}

func NewBootstrapManager(db *sql.DB, stateDir string) *BootstrapManager {
	return &BootstrapManager{
		db:       db,
		stateDir: stateDir,
		policy:   DefaultPasswordPolicy(),
	}
}

func (manager *BootstrapManager) TokenPath() string {
	return filepath.Join(manager.stateDir, bootstrapTokenFile)
}

func (manager *BootstrapManager) Ensure(ctx context.Context) (BootstrapStatus, error) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	count, err := manager.userCount(ctx, manager.db)
	if err != nil {
		return BootstrapStatus{}, err
	}
	if count > 0 {
		_ = os.Remove(manager.TokenPath())
		return BootstrapStatus{Required: false}, nil
	}
	var storedHash string
	err = manager.db.QueryRowContext(
		ctx,
		"SELECT value FROM server_metadata WHERE key = 'bootstrap_token_hash'",
	).Scan(&storedHash)
	switch {
	case err == nil:
		if _, statErr := os.Stat(manager.TokenPath()); statErr == nil {
			return BootstrapStatus{Required: true, TokenFile: manager.TokenPath()}, nil
		}
		// Another server pod owns the one-time token file. The hash is shared
		// through the database, so this pod can still accept that token.
		return BootstrapStatus{Required: true}, nil
	case errors.Is(err, sql.ErrNoRows):
	default:
		return BootstrapStatus{}, fmt.Errorf("read bootstrap token state: %w", err)
	}
	token, hash, err := newSecret()
	if err != nil {
		return BootstrapStatus{}, err
	}
	if err := writeSecretFile(manager.TokenPath(), []byte(token+"\n")); err != nil {
		return BootstrapStatus{}, fmt.Errorf("write initial administrator token: %w", err)
	}
	result, err := manager.db.ExecContext(
		ctx,
		`INSERT INTO server_metadata(key, value, updated_at)
		 VALUES ('bootstrap_token_hash', $1, CURRENT_TIMESTAMP)
		 ON CONFLICT (key) DO NOTHING`,
		hash,
	)
	if err != nil {
		_ = os.Remove(manager.TokenPath())
		return BootstrapStatus{}, fmt.Errorf("store initial administrator token hash: %w", err)
	}
	if rows, err := result.RowsAffected(); err != nil || rows == 0 {
		_ = os.Remove(manager.TokenPath())
		return BootstrapStatus{Required: true}, err
	}
	return BootstrapStatus{Required: true, TokenFile: manager.TokenPath()}, nil
}

func (manager *BootstrapManager) CreateInitialAdmin(
	ctx context.Context,
	token string,
	input InitialAdminInput,
	sourceIP string,
	userAgent string,
	requestID string,
) (User, error) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	var expectedHash string
	if err := manager.db.QueryRowContext(
		ctx,
		"SELECT value FROM server_metadata WHERE key = 'bootstrap_token_hash'",
	).Scan(&expectedHash); errors.Is(err, sql.ErrNoRows) {
		return User{}, ErrBootstrapComplete
	} else if err != nil {
		return User{}, fmt.Errorf("read bootstrap token hash: %w", err)
	}
	actualHash := sha256.Sum256([]byte(strings.TrimSpace(token)))
	expectedBytes, err := hex.DecodeString(expectedHash)
	if err != nil ||
		len(expectedBytes) != len(actualHash) ||
		subtle.ConstantTimeCompare(actualHash[:], expectedBytes) != 1 {
		return User{}, ErrBootstrapToken
	}
	username, passwordHash, err := manager.prepareInitialAdmin(input)
	if err != nil {
		return User{}, err
	}
	return manager.createInitialAdmin(
		ctx,
		input,
		username,
		passwordHash,
		`DELETE FROM server_metadata
		 WHERE key = 'bootstrap_token_hash' AND value = $1`,
		[]any{expectedHash},
		sourceIP,
		userAgent,
		requestID,
		"token",
	)
}

// CreateInitialAdminFromConfig consumes the shared bootstrap claim without
// requiring access to a pod-local token file. It is intended only for
// process-start credentials supplied by the operator. The metadata DELETE is
// the cross-pod claim: exactly one transaction can create the initial user.
func (manager *BootstrapManager) CreateInitialAdminFromConfig(
	ctx context.Context,
	input InitialAdminInput,
) (User, error) {
	manager.mutex.Lock()
	defer manager.mutex.Unlock()
	count, err := manager.userCount(ctx, manager.db)
	if err != nil {
		return User{}, err
	}
	if count != 0 {
		_ = os.Remove(manager.TokenPath())
		return User{}, ErrBootstrapComplete
	}
	username, passwordHash, err := manager.prepareInitialAdmin(input)
	if err != nil {
		return User{}, err
	}
	return manager.createInitialAdmin(
		ctx,
		input,
		username,
		passwordHash,
		`DELETE FROM server_metadata WHERE key = 'bootstrap_token_hash'`,
		nil,
		"",
		"server-startup",
		"startup",
		"environment",
	)
}

func (manager *BootstrapManager) prepareInitialAdmin(
	input InitialAdminInput,
) (string, string, error) {
	username, err := validateUsername(input.Username)
	if err != nil {
		return "", "", err
	}
	if err := manager.policy.Validate(input.Password); err != nil {
		return "", "", err
	}
	if input.Email != "" {
		address, err := mail.ParseAddress(input.Email)
		if err != nil || !strings.EqualFold(address.Address, strings.TrimSpace(input.Email)) {
			return "", "", errors.New("email address is invalid")
		}
	}
	passwordHash, err := HashPassword(input.Password)
	if err != nil {
		return "", "", err
	}
	return username, passwordHash, nil
}

func (manager *BootstrapManager) createInitialAdmin(
	ctx context.Context,
	input InitialAdminInput,
	username string,
	passwordHash string,
	claimQuery string,
	claimArguments []any,
	sourceIP string,
	userAgent string,
	requestID string,
	method string,
) (User, error) {
	transaction, err := manager.db.BeginTx(ctx, nil)
	if err != nil {
		return User{}, fmt.Errorf("begin initial administrator transaction: %w", err)
	}
	defer transaction.Rollback()
	claim, err := transaction.ExecContext(ctx, claimQuery, claimArguments...)
	if err != nil {
		return User{}, fmt.Errorf("claim initial administrator bootstrap: %w", err)
	}
	claimed, err := claim.RowsAffected()
	if err != nil {
		return User{}, fmt.Errorf("read initial administrator claim: %w", err)
	}
	if claimed != 1 {
		return User{}, ErrBootstrapComplete
	}
	count, err := manager.userCount(ctx, transaction)
	if err != nil {
		return User{}, err
	}
	if count != 0 {
		return User{}, ErrBootstrapComplete
	}
	user := User{
		ID:          uuid.NewString(),
		Username:    username,
		DisplayName: strings.TrimSpace(input.DisplayName),
		Email:       strings.TrimSpace(input.Email),
		SuperAdmin:  true,
		Roles:       []string{"super_admin"},
	}
	if _, err := transaction.ExecContext(
		ctx,
		`INSERT INTO users(
			id, username, normalized_username, display_name, email,
			password_hash, active, super_admin, password_changed_at
		) VALUES ($1, $2, $3, $4, $5, $6, TRUE, TRUE, CURRENT_TIMESTAMP)`,
		user.ID,
		user.Username,
		normalizeUsername(user.Username),
		user.DisplayName,
		user.Email,
		passwordHash,
	); err != nil {
		return User{}, fmt.Errorf("create initial administrator: %w", err)
	}
	if _, err := transaction.ExecContext(
		ctx,
		`INSERT INTO user_roles(user_id, role_id, source)
		 VALUES ($1, $2, 'local')`,
		user.ID,
		superAdminRoleID,
	); err != nil {
		return User{}, fmt.Errorf("grant initial administrator role: %w", err)
	}
	if err := manager.audit.Record(ctx, transaction, audit.Entry{
		ActorType:    "bootstrap",
		ActorID:      user.ID,
		ActorName:    user.Username,
		Action:       "user.initial_admin.create",
		ResourceType: "user",
		ResourceID:   user.ID,
		RequestID:    requestID,
		SourceIP:     sourceIP,
		UserAgent:    userAgent,
		Result:       "success",
		After: map[string]any{
			"username":     user.Username,
			"display_name": user.DisplayName,
			"email":        user.Email,
			"super_admin":  true,
		},
		Metadata: map[string]string{"method": method},
	}); err != nil {
		return User{}, err
	}
	if err := transaction.Commit(); err != nil {
		return User{}, fmt.Errorf("commit initial administrator: %w", err)
	}
	if err := os.Remove(manager.TokenPath()); err != nil && !errors.Is(err, os.ErrNotExist) {
		return User{}, fmt.Errorf("remove consumed bootstrap token: %w", err)
	}
	return user, nil
}

type countQuerier interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

func (manager *BootstrapManager) userCount(
	ctx context.Context,
	database countQuerier,
) (int, error) {
	var count int
	if err := database.QueryRowContext(
		ctx,
		"SELECT COUNT(*) FROM users WHERE deleted_at IS NULL",
	).Scan(&count); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

func validateUsername(value string) (string, error) {
	value = strings.TrimSpace(value)
	if len(value) < 3 || len(value) > 64 {
		return "", errors.New("username must contain 3 to 64 characters")
	}
	for _, char := range value {
		if char >= 'a' && char <= 'z' ||
			char >= 'A' && char <= 'Z' ||
			char >= '0' && char <= '9' ||
			char == '.' || char == '_' || char == '-' {
			continue
		}
		return "", errors.New("username contains unsupported characters")
	}
	return value, nil
}

// ValidateUsername applies the same account-name policy used by initial setup.
func ValidateUsername(value string) (string, error) {
	return validateUsername(value)
}

func normalizeUsername(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

// NormalizeUsername returns the case-insensitive lookup form stored for users.
func NormalizeUsername(value string) string {
	return normalizeUsername(value)
}

func newSecret() (string, string, error) {
	bytes := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", "", fmt.Errorf("generate secure token: %w", err)
	}
	token := base64.RawURLEncoding.EncodeToString(bytes)
	hash := sha256.Sum256([]byte(token))
	return token, hex.EncodeToString(hash[:]), nil
}

func writeSecretFile(path string, bytes []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(path), ".secret-*")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer os.Remove(temporaryPath)
	if err := temporary.Chmod(0o600); err != nil {
		temporary.Close()
		return err
	}
	if _, err := temporary.Write(bytes); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Sync(); err != nil {
		temporary.Close()
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporaryPath, path); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}
