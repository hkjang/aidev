package apikeys

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/invenqor/server/internal/apitime"
)

var (
	ErrInvalid      = errors.New("API key input is invalid")
	ErrUnauthorized = errors.New("API key is invalid or inactive")
	ErrNotFound     = errors.New("API key does not exist")
	ErrConflict     = errors.New("API key changed concurrently")
	ErrForbidden    = errors.New("API key operation is not allowed")
	// ErrScopesRequired keeps the rule Create has always applied on the paths
	// that change an existing key. A key with no scopes still authenticates,
	// so it is not disabled: it is refused by every scope check instead, which
	// reads as a broken endpoint rather than a retired credential.
	ErrScopesRequired = fmt.Errorf("%w: at least one scope is required", ErrInvalid)
)

const scopeMutationAttempts = 8

type Scope struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

var scopeCatalog = []Scope{
	{Name: "agents.read", Description: "Read agent status"},
	{Name: "assets.delete", Description: "Logically delete and restore assets"},
	{Name: "assets.read", Description: "Read and search assets"},
	{Name: "assets.write", Description: "Create and update assets"},
	{Name: "mcp.access", Description: "Connect to the Invenqor MCP endpoint"},
	{Name: "queries.execute", Description: "Execute the validated asset Query DSL"},
	{Name: "relations.read", Description: "Read asset relationships"},
	{Name: "relations.write", Description: "Create and remove asset relationships"},
}

type Key struct {
	ID         string       `json:"id"`
	UserID     string       `json:"user_id,omitempty"`
	Name       string       `json:"name"`
	Prefix     string       `json:"prefix"`
	Scopes     []string     `json:"scopes"`
	ExpiresAt  apitime.Time `json:"expires_at,omitempty"`
	LastUsedAt apitime.Time `json:"last_used_at,omitempty"`
	CreatedAt  apitime.Time `json:"created_at"`
	UpdatedAt  apitime.Time `json:"updated_at"`
	RevokedAt  apitime.Time `json:"revoked_at,omitempty"`
}

type Credential struct {
	KeyID  string
	UserID string
	Name   string
	Scopes []string
}

type Created struct {
	Key    Key    `json:"api_key"`
	Secret string `json:"secret"`
}

// Access describes the authenticated user at the service boundary. Regular
// administrators can manage only their own keys, and plaintext credentials
// can be issued only for scopes they already hold. Super administrators are
// the explicit exception to both rules.
type Access struct {
	UserID          string
	SuperAdmin      bool
	GrantableScopes []string
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func Scopes() []Scope {
	result := make([]Scope, len(scopeCatalog))
	copy(result, scopeCatalog)
	return result
}

func ValidScopes(scopes []string) ([]string, error) {
	allowed := make(map[string]struct{}, len(scopeCatalog))
	for _, scope := range scopeCatalog {
		allowed[scope.Name] = struct{}{}
	}
	unique := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if _, ok := allowed[scope]; !ok {
			return nil, fmt.Errorf("%w: unknown scope %q", ErrInvalid, scope)
		}
		unique[scope] = struct{}{}
	}
	result := make([]string, 0, len(unique))
	for scope := range unique {
		result = append(result, scope)
	}
	sort.Strings(result)
	return result, nil
}

// RequireScopes validates a key's complete scope list. Callers that replace
// the whole list use this instead of ValidScopes, which accepts an empty list
// because AddScopes and RemoveScope pass it a fragment rather than the result.
func RequireScopes(scopes []string) ([]string, error) {
	validated, err := ValidScopes(scopes)
	if err != nil {
		return nil, err
	}
	if len(validated) == 0 {
		return nil, ErrScopesRequired
	}
	return validated, nil
}

func (s *Service) Create(
	ctx context.Context,
	userID string,
	name string,
	scopes []string,
	expiresAt *time.Time,
) (Created, error) {
	name = strings.TrimSpace(name)
	if name == "" || len(name) > 120 || userID == "" {
		return Created{}, ErrInvalid
	}
	scopes, err := RequireScopes(scopes)
	if err != nil {
		return Created{}, err
	}
	if expiresAt != nil && !expiresAt.After(time.Now().UTC()) {
		return Created{}, fmt.Errorf("%w: expiry must be in the future", ErrInvalid)
	}
	secret, prefix, hash, err := newSecret()
	if err != nil {
		return Created{}, err
	}
	id := uuid.NewString()
	scopeJSON, _ := json.Marshal(scopes)
	now := time.Now().UTC()
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO api_keys(
			id, user_id, name, key_hash, key_prefix, scopes_json,
			expires_at, created_at, updated_at
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$8)`,
		id, userID, name, hash, prefix, string(scopeJSON), expiresAt, now,
	)
	if err != nil {
		return Created{}, fmt.Errorf("create API key: %w", err)
	}
	key, err := s.Get(ctx, id)
	return Created{Key: key, Secret: secret}, err
}

func (s *Service) List(ctx context.Context) ([]Key, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, name, key_prefix, scopes_json, expires_at,
		 last_used_at, created_at, updated_at, revoked_at
		 FROM api_keys ORDER BY created_at DESC, id`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Key, 0)
	for rows.Next() {
		key, err := scanKey(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, key)
	}
	return result, rows.Err()
}

func (s *Service) ListFor(ctx context.Context, access Access) ([]Key, error) {
	if access.UserID == "" {
		return nil, ErrForbidden
	}
	if access.SuperAdmin {
		return s.List(ctx)
	}
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, name, key_prefix, scopes_json, expires_at,
		 last_used_at, created_at, updated_at, revoked_at
		 FROM api_keys WHERE user_id=$1 ORDER BY created_at DESC, id`,
		access.UserID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := make([]Key, 0)
	for rows.Next() {
		key, err := scanKey(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, key)
	}
	return result, rows.Err()
}

func (s *Service) Get(ctx context.Context, id string) (Key, error) {
	key, err := scanKey(s.db.QueryRowContext(ctx,
		`SELECT id, user_id, name, key_prefix, scopes_json, expires_at,
		 last_used_at, created_at, updated_at, revoked_at
		 FROM api_keys WHERE id=$1`, id,
	))
	if errors.Is(err, sql.ErrNoRows) {
		return Key{}, ErrNotFound
	}
	return key, err
}

func (s *Service) GetFor(ctx context.Context, id string, access Access) (Key, error) {
	if access.UserID == "" {
		return Key{}, ErrForbidden
	}
	key, err := s.Get(ctx, id)
	if err != nil {
		return Key{}, err
	}
	if !access.SuperAdmin && key.UserID != access.UserID {
		// Do not disclose another user's key identifier to a regular caller.
		return Key{}, ErrNotFound
	}
	return key, nil
}

func (s *Service) ReplaceScopes(
	ctx context.Context,
	id string,
	scopes []string,
) (Key, error) {
	return s.Update(ctx, id, nil, &scopes)
}

func (s *Service) AddScopes(
	ctx context.Context,
	id string,
	additions []string,
) (Key, error) {
	additions, err := ValidScopes(additions)
	if err != nil {
		return Key{}, err
	}
	return s.mutateScopes(ctx, id, func(current []string) ([]string, error) {
		return append(current, additions...), nil
	})
}

func (s *Service) RemoveScope(
	ctx context.Context,
	id string,
	remove string,
) (Key, error) {
	remove = strings.TrimSpace(remove)
	if _, err := ValidScopes([]string{remove}); err != nil {
		return Key{}, err
	}
	return s.mutateScopes(ctx, id, func(current []string) ([]string, error) {
		next := make([]string, 0, len(current))
		for _, scope := range current {
			if scope != remove {
				next = append(next, scope)
			}
		}
		if len(next) == len(current) {
			return nil, fmt.Errorf("%w: scope is not assigned", ErrInvalid)
		}
		return next, nil
	})
}

// mutateScopes uses the stored JSON value as an optimistic version. This is
// deliberately database-backed instead of guarded by a process mutex: two
// Server Pods can otherwise read the same scope list and silently overwrite
// one another's additions or removals.
func (s *Service) mutateScopes(
	ctx context.Context,
	id string,
	mutate func([]string) ([]string, error),
) (Key, error) {
	for range scopeMutationAttempts {
		var currentJSON string
		err := s.db.QueryRowContext(
			ctx,
			`SELECT scopes_json FROM api_keys
			 WHERE id=$1 AND revoked_at IS NULL`,
			id,
		).Scan(&currentJSON)
		if errors.Is(err, sql.ErrNoRows) {
			return Key{}, ErrNotFound
		}
		if err != nil {
			return Key{}, err
		}
		var current []string
		if err := json.Unmarshal([]byte(currentJSON), &current); err != nil {
			return Key{}, fmt.Errorf("decode API key scopes: %w", err)
		}
		next, err := mutate(append([]string(nil), current...))
		if err != nil {
			return Key{}, err
		}
		next, err = RequireScopes(next)
		if err != nil {
			return Key{}, err
		}
		nextJSON, _ := json.Marshal(next)
		result, err := s.db.ExecContext(
			ctx,
			`UPDATE api_keys
			 SET scopes_json=$1, updated_at=CURRENT_TIMESTAMP
			 WHERE id=$2 AND revoked_at IS NULL AND scopes_json=$3`,
			string(nextJSON),
			id,
			currentJSON,
		)
		if err != nil {
			return Key{}, err
		}
		if changed, _ := result.RowsAffected(); changed == 1 {
			return s.Get(ctx, id)
		}
	}
	return Key{}, ErrConflict
}

func (s *Service) UpdateName(
	ctx context.Context,
	id string,
	name string,
) (Key, error) {
	return s.Update(ctx, id, &name, nil)
}

// Update applies a name/scopes PATCH in one database statement so validation,
// revocation, or a database failure cannot leave a partially updated key.
func (s *Service) Update(
	ctx context.Context,
	id string,
	name *string,
	scopes *[]string,
) (Key, error) {
	if name == nil && scopes == nil {
		return Key{}, ErrInvalid
	}
	nextName := ""
	if name != nil {
		nextName = strings.TrimSpace(*name)
		if nextName == "" || len(nextName) > 120 {
			return Key{}, ErrInvalid
		}
	}
	nextScopes := "[]"
	expectedScopes := "[]"
	if scopes != nil {
		validated, err := RequireScopes(*scopes)
		if err != nil {
			return Key{}, err
		}
		encoded, _ := json.Marshal(validated)
		nextScopes = string(encoded)
		if err := s.db.QueryRowContext(
			ctx,
			`SELECT scopes_json FROM api_keys
			 WHERE id=$1 AND revoked_at IS NULL`,
			id,
		).Scan(&expectedScopes); errors.Is(err, sql.ErrNoRows) {
			return Key{}, ErrNotFound
		} else if err != nil {
			return Key{}, err
		}
	}
	return s.update(ctx, id, name, nextName, scopes, nextScopes, expectedScopes)
}

func (s *Service) update(
	ctx context.Context,
	id string,
	name *string,
	nextName string,
	scopes *[]string,
	nextScopes string,
	expectedScopes string,
) (Key, error) {
	result, err := s.db.ExecContext(
		ctx,
		`UPDATE api_keys SET
		 name=CASE WHEN $1 THEN $2 ELSE name END,
		 scopes_json=CASE WHEN $3 THEN $4 ELSE scopes_json END,
		 updated_at=CURRENT_TIMESTAMP
		 WHERE id=$5 AND revoked_at IS NULL
		   AND ($3=FALSE OR scopes_json=$6)`,
		name != nil,
		nextName,
		scopes != nil,
		nextScopes,
		id,
		expectedScopes,
	)
	if err != nil {
		return Key{}, err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		var active int
		if err := s.db.QueryRowContext(
			ctx,
			`SELECT COUNT(*) FROM api_keys
			 WHERE id=$1 AND revoked_at IS NULL`,
			id,
		).Scan(&active); err != nil {
			return Key{}, err
		}
		if active == 0 {
			return Key{}, ErrNotFound
		}
		return Key{}, ErrConflict
	}
	return s.Get(ctx, id)
}

func (s *Service) Rotate(
	ctx context.Context,
	id string,
	grace time.Duration,
	access Access,
) (Created, error) {
	if grace < 0 || grace > 7*24*time.Hour {
		return Created{}, ErrInvalid
	}
	if access.UserID == "" {
		return Created{}, ErrForbidden
	}
	var expectedHash, rawScopes string
	var ownerID sql.NullString
	if err := s.db.QueryRowContext(
		ctx,
		`SELECT key_hash, user_id, scopes_json FROM api_keys
		 WHERE id=$1 AND revoked_at IS NULL
		   AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)`,
		id,
	).Scan(&expectedHash, &ownerID, &rawScopes); errors.Is(err, sql.ErrNoRows) {
		return Created{}, ErrNotFound
	} else if err != nil {
		return Created{}, err
	}
	if !ownerID.Valid {
		return Created{}, ErrForbidden
	}
	if !access.SuperAdmin && ownerID.String != access.UserID {
		return Created{}, ErrNotFound
	}
	var scopes []string
	if err := json.Unmarshal([]byte(rawScopes), &scopes); err != nil {
		return Created{}, fmt.Errorf("decode API key scopes: %w", err)
	}
	if !access.SuperAdmin && !canGrantAll(access.GrantableScopes, scopes) {
		return Created{}, ErrForbidden
	}
	return s.rotate(ctx, id, grace, expectedHash)
}

func canGrantAll(grantable []string, requested []string) bool {
	allowed := make(map[string]struct{}, len(grantable))
	for _, scope := range grantable {
		allowed[scope] = struct{}{}
	}
	for _, scope := range requested {
		if _, ok := allowed[scope]; !ok {
			return false
		}
	}
	return true
}

func (s *Service) rotate(
	ctx context.Context,
	id string,
	grace time.Duration,
	expectedHash string,
) (Created, error) {
	secret, prefix, hash, err := newSecret()
	if err != nil {
		return Created{}, err
	}
	now := time.Now().UTC()
	result, err := s.db.ExecContext(ctx,
		`UPDATE api_keys SET
		 previous_key_hash=key_hash, previous_valid_until=$1,
		 key_hash=$2, key_prefix=$3, updated_at=$4
		 WHERE id=$5 AND revoked_at IS NULL
		 AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)
		 AND key_hash=$6`,
		now.Add(grace), hash, prefix, now, id, expectedHash,
	)
	if err != nil {
		return Created{}, err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		var active int
		if err := s.db.QueryRowContext(
			ctx,
			`SELECT COUNT(*) FROM api_keys
			 WHERE id=$1 AND revoked_at IS NULL
			   AND (expires_at IS NULL OR expires_at > CURRENT_TIMESTAMP)`,
			id,
		).Scan(&active); err != nil {
			return Created{}, err
		}
		if active == 0 {
			return Created{}, ErrNotFound
		}
		return Created{}, ErrConflict
	}
	key, err := s.Get(ctx, id)
	return Created{Key: key, Secret: secret}, err
}

func (s *Service) Revoke(ctx context.Context, id string) error {
	result, err := s.db.ExecContext(ctx,
		`UPDATE api_keys SET revoked_at=CURRENT_TIMESTAMP,
		 previous_key_hash=NULL, previous_valid_until=NULL,
		 updated_at=CURRENT_TIMESTAMP
		 WHERE id=$1 AND revoked_at IS NULL`, id,
	)
	if err != nil {
		return err
	}
	if rows, _ := result.RowsAffected(); rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) Authenticate(ctx context.Context, secret string) (Credential, error) {
	if !strings.HasPrefix(secret, "ivq_sk_") || len(secret) < 40 {
		return Credential{}, ErrUnauthorized
	}
	hash := sha256.Sum256([]byte(secret))
	var credential Credential
	var userID sql.NullString
	var scopesJSON string
	err := s.db.QueryRowContext(ctx,
		`SELECT k.id, k.user_id, k.name, k.scopes_json
		 FROM api_keys k JOIN users u ON u.id=k.user_id
		 WHERE k.revoked_at IS NULL AND u.active=TRUE AND u.deleted_at IS NULL
		 AND (k.expires_at IS NULL OR k.expires_at > CURRENT_TIMESTAMP)
		 AND (k.key_hash=$1 OR (
		   k.previous_key_hash=$1 AND k.previous_valid_until > CURRENT_TIMESTAMP
		 ))`, hex.EncodeToString(hash[:]),
	).Scan(&credential.KeyID, &userID, &credential.Name, &scopesJSON)
	if errors.Is(err, sql.ErrNoRows) {
		return Credential{}, ErrUnauthorized
	}
	if err != nil {
		return Credential{}, err
	}
	if err := json.Unmarshal([]byte(scopesJSON), &credential.Scopes); err != nil {
		return Credential{}, ErrUnauthorized
	}
	credential.UserID = userID.String
	_, _ = s.db.ExecContext(ctx,
		`UPDATE api_keys SET last_used_at=CURRENT_TIMESTAMP WHERE id=$1`,
		credential.KeyID,
	)
	return credential, nil
}

func newSecret() (secret string, prefix string, hash string, err error) {
	random := make([]byte, 32)
	if _, err = rand.Read(random); err != nil {
		return "", "", "", err
	}
	encoded := base64.RawURLEncoding.EncodeToString(random)
	prefix = encoded[:8]
	secret = "ivq_sk_" + prefix + "_" + encoded
	sum := sha256.Sum256([]byte(secret))
	hash = hex.EncodeToString(sum[:])
	return
}

type rowScanner interface {
	Scan(...any) error
}

func scanKey(scanner rowScanner) (Key, error) {
	var key Key
	var userID sql.NullString
	var scopesJSON string
	err := scanner.Scan(
		&key.ID, &userID, &key.Name, &key.Prefix, &scopesJSON,
		&key.ExpiresAt, &key.LastUsedAt, &key.CreatedAt, &key.UpdatedAt,
		&key.RevokedAt,
	)
	if err != nil {
		return Key{}, err
	}
	key.UserID = userID.String
	if err := json.Unmarshal([]byte(scopesJSON), &key.Scopes); err != nil {
		return Key{}, err
	}
	return key, nil
}
