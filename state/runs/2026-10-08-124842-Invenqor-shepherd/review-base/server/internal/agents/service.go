package agents

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
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hkjang/invenqor/server/internal/apitime"
	"github.com/hkjang/invenqor/server/internal/audit"
)

var (
	ErrUnauthorized            = errors.New("agent credential is not valid")
	ErrBlocked                 = errors.New("agent is blocked")
	ErrEnrollmentClaimMismatch = errors.New("agent enrollment claim does not match")
	ErrInvalidEnrollment       = errors.New("agent enrollment request is invalid")
)

type Agent struct {
	ID              string       `json:"id"`
	AgentID         string       `json:"agent_id"`
	Hostname        string       `json:"hostname"`
	Status          string       `json:"status"`
	Version         string       `json:"version"`
	OSName          string       `json:"os_name"`
	Architecture    string       `json:"architecture"`
	AuthMethod      string       `json:"auth_method"`
	PolicyVersion   string       `json:"policy_version"`
	LastSeenAt      apitime.Time `json:"last_seen_at,omitempty"`
	LastInventoryAt apitime.Time `json:"last_inventory_at,omitempty"`
}

type ProvisionResult struct {
	Agent Agent  `json:"agent"`
	Token string `json:"token"`
}

type Service struct {
	database *sql.DB
	audit    audit.Recorder
	now      func() time.Time
	cacheMu  sync.RWMutex
	cache    map[string]cachedAgent
}

type cachedAgent struct {
	agent     Agent
	expiresAt time.Time
}

func NewService(database *sql.DB) *Service {
	return &Service{
		database: database,
		audit:    audit.Recorder{},
		now:      func() time.Time { return time.Now().UTC() },
		cache:    make(map[string]cachedAgent),
	}
}

// ProvisionBearer creates an agent if necessary and returns a new bearer token.
// Only the SHA-256 digest is retained; the plaintext token is returned once.
func (s *Service) ProvisionBearer(
	ctx context.Context,
	agentID string,
	hostname string,
	actorID string,
) (ProvisionResult, error) {
	if _, err := uuid.Parse(agentID); err != nil {
		return ProvisionResult{}, fmt.Errorf("agent_id must be a UUID: %w", err)
	}
	token, err := randomToken("ivq_at_", 32)
	if err != nil {
		return ProvisionResult{}, err
	}
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return ProvisionResult{}, fmt.Errorf("begin agent provisioning: %w", err)
	}
	defer tx.Rollback()

	internalID := ""
	err = tx.QueryRowContext(
		ctx,
		`SELECT id FROM agents WHERE agent_id = $1`,
		agentID,
	).Scan(&internalID)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		internalID = uuid.NewString()
		_, err = tx.ExecContext(
			ctx,
			`INSERT INTO agents(
				id, agent_id, hostname, status, auth_method, updated_at
			) VALUES ($1, $2, $3, 'provisioned', 'bearer', $4)`,
			internalID,
			agentID,
			strings.TrimSpace(hostname),
			s.now(),
		)
	case err == nil:
		_, err = tx.ExecContext(
			ctx,
			`UPDATE agents SET hostname = CASE WHEN $1 = '' THEN hostname ELSE $1 END,
				auth_method = 'bearer', status = CASE WHEN blocked_at IS NULL
				THEN 'provisioned' ELSE status END, updated_at = $2 WHERE id = $3`,
			strings.TrimSpace(hostname),
			s.now(),
			internalID,
		)
	}
	if err != nil {
		return ProvisionResult{}, fmt.Errorf("upsert agent: %w", err)
	}
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO agent_credentials(
			id, agent_id, credential_type, secret_hash, not_before
		) VALUES ($1, $2, 'bearer', $3, CURRENT_TIMESTAMP)`,
		uuid.NewString(),
		internalID,
		digest(token),
	)
	if err != nil {
		return ProvisionResult{}, fmt.Errorf("store agent credential: %w", err)
	}
	if err := s.audit.Record(ctx, tx, audit.Entry{
		ActorType:    "user",
		ActorID:      actorID,
		Action:       "agent.provision",
		ResourceType: "agent",
		ResourceID:   internalID,
		Result:       "success",
		After: map[string]any{
			"agent_id": agentID,
			"hostname": strings.TrimSpace(hostname),
			"auth":     "bearer",
		},
	}); err != nil {
		return ProvisionResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return ProvisionResult{}, fmt.Errorf("commit agent provisioning: %w", err)
	}
	agent, err := s.Get(ctx, internalID)
	if err != nil {
		return ProvisionResult{}, err
	}
	return ProvisionResult{Agent: agent, Token: token}, nil
}

// AutoEnroll exchanges a fleet enrollment authorization and a device-local
// claim for a device-specific bearer credential. The claim makes retries safe:
// if the first response is lost, only the same device can replace the bearer.
func (s *Service) AutoEnroll(
	ctx context.Context,
	agentID string,
	hostname string,
	claimToken string,
	sourceIP string,
) (ProvisionResult, error) {
	if _, err := uuid.Parse(agentID); err != nil {
		return ProvisionResult{}, fmt.Errorf("%w: agent_id must be a UUID", ErrInvalidEnrollment)
	}
	if !strings.HasPrefix(claimToken, "ivq_ec_") || len(claimToken) < 39 {
		return ProvisionResult{}, fmt.Errorf("%w: enrollment claim is invalid", ErrInvalidEnrollment)
	}
	bearer, err := randomToken("ivq_at_", 32)
	if err != nil {
		return ProvisionResult{}, err
	}
	now := s.now()
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return ProvisionResult{}, fmt.Errorf("begin automatic enrollment: %w", err)
	}
	defer tx.Rollback()

	var internalID string
	var blocked any
	err = tx.QueryRowContext(
		ctx,
		`SELECT id, blocked_at FROM agents WHERE agent_id = $1`,
		agentID,
	).Scan(&internalID, &blocked)
	switch {
	case errors.Is(err, sql.ErrNoRows):
		internalID = uuid.NewString()
		if _, err = tx.ExecContext(
			ctx,
			`INSERT INTO agents(
				id, agent_id, hostname, status, auth_method, updated_at
			) VALUES ($1, $2, $3, 'provisioned', 'auto_bearer', $4)`,
			internalID,
			agentID,
			strings.TrimSpace(hostname),
			now,
		); err != nil {
			return ProvisionResult{}, fmt.Errorf("create automatically enrolled agent: %w", err)
		}
		if _, err = tx.ExecContext(
			ctx,
			`INSERT INTO agent_credentials(
				id, agent_id, credential_type, secret_hash, not_before
			) VALUES ($1, $2, 'enrollment_claim', $3, CURRENT_TIMESTAMP)`,
			uuid.NewString(),
			internalID,
			digest(claimToken),
		); err != nil {
			return ProvisionResult{}, fmt.Errorf("store enrollment claim: %w", err)
		}
	case err != nil:
		return ProvisionResult{}, fmt.Errorf("find enrollment target: %w", err)
	default:
		if blocked != nil {
			return ProvisionResult{}, ErrBlocked
		}
		var credentialID string
		if err := tx.QueryRowContext(
			ctx,
			`SELECT id FROM agent_credentials
			 WHERE agent_id = $1
			   AND credential_type = 'enrollment_claim'
			   AND secret_hash = $2
			   AND revoked_at IS NULL
			 ORDER BY created_at DESC LIMIT 1`,
			internalID,
			digest(claimToken),
		).Scan(&credentialID); errors.Is(err, sql.ErrNoRows) {
			return ProvisionResult{}, ErrEnrollmentClaimMismatch
		} else if err != nil {
			return ProvisionResult{}, fmt.Errorf("verify enrollment claim: %w", err)
		}
		if _, err = tx.ExecContext(
			ctx,
			`UPDATE agent_credentials
			 SET revoked_at = $1
			 WHERE agent_id = $2
			   AND credential_type = 'bearer'
			   AND revoked_at IS NULL`,
			now,
			internalID,
		); err != nil {
			return ProvisionResult{}, fmt.Errorf("replace lost bearer credential: %w", err)
		}
		if _, err = tx.ExecContext(
			ctx,
			`UPDATE agents SET
				hostname = CASE WHEN $1 = '' THEN hostname ELSE $1 END,
				status = 'provisioned', auth_method = 'auto_bearer',
				updated_at = $2
			 WHERE id = $3`,
			strings.TrimSpace(hostname),
			now,
			internalID,
		); err != nil {
			return ProvisionResult{}, fmt.Errorf("refresh automatically enrolled agent: %w", err)
		}
	}
	if _, err = tx.ExecContext(
		ctx,
		`INSERT INTO agent_credentials(
			id, agent_id, credential_type, secret_hash, not_before
		) VALUES ($1, $2, 'bearer', $3, CURRENT_TIMESTAMP)`,
		uuid.NewString(),
		internalID,
		digest(bearer),
	); err != nil {
		return ProvisionResult{}, fmt.Errorf("store device bearer credential: %w", err)
	}
	if err := s.ensureEnrollmentAsset(
		ctx,
		tx,
		internalID,
		agentID,
		strings.TrimSpace(hostname),
		strings.TrimSpace(sourceIP),
		now,
	); err != nil {
		return ProvisionResult{}, err
	}
	if err := s.audit.Record(ctx, tx, audit.Entry{
		ActorType: "agent", ActorID: internalID, Action: "agent.auto_enroll",
		ResourceType: "agent", ResourceID: internalID, Result: "success",
		After: map[string]any{
			"agent_id":  agentID,
			"hostname":  strings.TrimSpace(hostname),
			"auth":      "auto_bearer",
			"source_ip": strings.TrimSpace(sourceIP),
		},
	}); err != nil {
		return ProvisionResult{}, err
	}
	if err := tx.Commit(); err != nil {
		return ProvisionResult{}, fmt.Errorf("commit automatic enrollment: %w", err)
	}
	s.invalidateCache(internalID)
	agent, err := s.Get(ctx, internalID)
	if err != nil {
		return ProvisionResult{}, err
	}
	return ProvisionResult{Agent: agent, Token: bearer}, nil
}

func (s *Service) ensureEnrollmentAsset(
	ctx context.Context,
	tx *sql.Tx,
	internalAgentID string,
	externalAgentID string,
	hostname string,
	sourceIP string,
	now time.Time,
) error {
	displayName := hostname
	if displayName == "" {
		displayName = sourceIP
	}
	if displayName == "" {
		displayName = "agent-" + externalAgentID[:8]
	}
	payload, err := json.Marshal(map[string]any{
		"agent_id":      externalAgentID,
		"enrollment_ip": sourceIP,
		"hostname":      hostname,
		"state":         "awaiting_inventory",
	})
	if err != nil {
		return fmt.Errorf("encode enrollment asset: %w", err)
	}

	var sourceID, assetID string
	err = tx.QueryRowContext(
		ctx,
		`SELECT id, asset_id FROM asset_sources
		 WHERE agent_id = $1 AND category = 'enrollment'
		   AND source_asset_id = $2`,
		internalAgentID,
		externalAgentID,
	).Scan(&sourceID, &assetID)
	if errors.Is(err, sql.ErrNoRows) {
		err = tx.QueryRowContext(
			ctx,
			`SELECT asset_id FROM asset_sources
			 WHERE agent_id = $1 AND category = 'system'
			   AND deleted_at IS NULL
			 ORDER BY first_seen_at LIMIT 1`,
			internalAgentID,
		).Scan(&assetID)
		if errors.Is(err, sql.ErrNoRows) {
			assetID = uuid.NewString()
			if _, err = tx.ExecContext(
				ctx,
				`INSERT INTO assets(
					id, asset_key, name, type, status, confidence,
					attributes_json, source, first_seen_at, last_seen_at,
					created_at, updated_at
				) VALUES (
					$1, $2, $3, 'host', 'discovered', 0.8,
					$4, 'agent_enrollment', $5, $5, $5, $5
				)`,
				assetID,
				"agent:"+externalAgentID,
				displayName,
				string(payload),
				now,
			); err != nil {
				return fmt.Errorf("create enrollment asset: %w", err)
			}
		} else if err != nil {
			return fmt.Errorf("find inventory asset for enrollment: %w", err)
		}
		sourceID = uuid.NewString()
		if _, err = tx.ExecContext(
			ctx,
			`INSERT INTO asset_sources(
				id, asset_id, agent_id, category, source_asset_id,
				source_name, payload_json, collected_at, first_seen_at,
				last_seen_at
			) VALUES ($1, $2, $3, 'enrollment', $4, $5, $6, $7, $7, $7)`,
			sourceID,
			assetID,
			internalAgentID,
			externalAgentID,
			sourceIP,
			string(payload),
			now,
		); err != nil {
			return fmt.Errorf("create enrollment asset source: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("find enrollment asset: %w", err)
	} else {
		if _, err = tx.ExecContext(
			ctx,
			`UPDATE asset_sources SET source_name=$1, payload_json=$2,
			 collected_at=$3, last_seen_at=$3, deleted_at=NULL
			 WHERE id=$4`,
			sourceIP,
			string(payload),
			now,
			sourceID,
		); err != nil {
			return fmt.Errorf("refresh enrollment asset source: %w", err)
		}
		if _, err = tx.ExecContext(
			ctx,
			`UPDATE assets SET
				name=CASE WHEN status='discovered' THEN $1 ELSE name END,
				last_seen_at=$2, updated_at=$2, deleted_at=NULL
			 WHERE id=$3`,
			displayName,
			now,
			assetID,
		); err != nil {
			return fmt.Errorf("refresh enrollment asset: %w", err)
		}
	}
	if sourceIP != "" {
		if _, err = tx.ExecContext(
			ctx,
			`INSERT INTO asset_identifiers(
				id, asset_id, identifier_type, identifier_value,
				source, confidence
			) VALUES ($1, $2, 'ip', $3, 'agent_enrollment', 0.8)
			ON CONFLICT(asset_id, identifier_type, identifier_value)
			DO UPDATE SET source='agent_enrollment', confidence=0.8`,
			uuid.NewString(),
			assetID,
			sourceIP,
		); err != nil {
			return fmt.Errorf("store enrollment IP identifier: %w", err)
		}
	}
	return nil
}

// RotateBearer issues a replacement and leaves previous credentials valid only
// for the requested grace period.
func (s *Service) RotateBearer(
	ctx context.Context,
	internalID string,
	grace time.Duration,
	actorID string,
) (string, error) {
	if grace < 0 {
		return "", errors.New("grace period cannot be negative")
	}
	token, err := randomToken("ivq_at_", 32)
	if err != nil {
		return "", err
	}
	now := s.now()
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer tx.Rollback()
	result, err := tx.ExecContext(
		ctx,
		`UPDATE agent_credentials SET expires_at = $1, grace_until = $2
		 WHERE agent_id = $3 AND credential_type = 'bearer' AND revoked_at IS NULL`,
		now,
		now.Add(grace),
		internalID,
	)
	if err != nil {
		return "", fmt.Errorf("expire previous credentials: %w", err)
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return "", sql.ErrNoRows
	}
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO agent_credentials(
			id, agent_id, credential_type, secret_hash, not_before
		) VALUES ($1, $2, 'bearer', $3, CURRENT_TIMESTAMP)`,
		uuid.NewString(),
		internalID,
		digest(token),
	)
	if err != nil {
		return "", fmt.Errorf("store replacement credential: %w", err)
	}
	if err := s.audit.Record(ctx, tx, audit.Entry{
		ActorType: "user", ActorID: actorID, Action: "agent.token.rotate",
		ResourceType: "agent", ResourceID: internalID, Result: "success",
		Metadata: map[string]any{"grace_seconds": int64(grace.Seconds())},
	}); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	s.invalidateCache(internalID)
	return token, nil
}

func (s *Service) RegisterCertificate(
	ctx context.Context,
	internalID string,
	fingerprint string,
	expiresAt *time.Time,
	actorID string,
) error {
	fingerprint = normalizeFingerprint(fingerprint)
	if len(fingerprint) != sha256.Size*2 {
		return errors.New("certificate fingerprint must be SHA-256")
	}
	_, err := hex.DecodeString(fingerprint)
	if err != nil {
		return errors.New("certificate fingerprint must be hexadecimal")
	}
	tx, err := s.database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	_, err = tx.ExecContext(
		ctx,
		`INSERT INTO agent_credentials(
			id, agent_id, credential_type, certificate_fingerprint,
			certificate_expires_at, not_before
		) VALUES ($1, $2, 'mtls', $3, $4, CURRENT_TIMESTAMP)`,
		uuid.NewString(), internalID, fingerprint, expiresAt,
	)
	if err != nil {
		return fmt.Errorf("store certificate credential: %w", err)
	}
	_, err = tx.ExecContext(
		ctx,
		`UPDATE agents SET auth_method = 'mtls', updated_at = $1 WHERE id = $2`,
		s.now(), internalID,
	)
	if err != nil {
		return err
	}
	if err := s.audit.Record(ctx, tx, audit.Entry{
		ActorType: "user", ActorID: actorID, Action: "agent.certificate.register",
		ResourceType: "agent", ResourceID: internalID, Result: "success",
	}); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Service) AuthenticateBearer(
	ctx context.Context,
	token string,
) (Agent, error) {
	if !strings.HasPrefix(token, "ivq_at_") {
		return Agent{}, ErrUnauthorized
	}
	return s.authenticate(ctx, "c.secret_hash", digest(token))
}

func (s *Service) AuthenticateCertificate(
	ctx context.Context,
	fingerprint string,
) (Agent, error) {
	return s.authenticate(
		ctx,
		"c.certificate_fingerprint",
		normalizeFingerprint(fingerprint),
	)
}

func (s *Service) authenticate(
	ctx context.Context,
	credentialColumn string,
	value string,
) (Agent, error) {
	if value == "" {
		return Agent{}, ErrUnauthorized
	}
	var agent Agent
	var blocked any
	query := `SELECT a.id, a.agent_id, a.hostname, a.status, a.version,
		a.os_name, a.architecture, a.auth_method, a.policy_version, a.blocked_at
		FROM agent_credentials c
		JOIN agents a ON a.id = c.agent_id
		WHERE ` + credentialColumn + ` = $1
		  AND c.revoked_at IS NULL
		  AND c.not_before <= CURRENT_TIMESTAMP
		  AND (
		    c.expires_at IS NULL OR c.expires_at > CURRENT_TIMESTAMP
		    OR c.grace_until > CURRENT_TIMESTAMP
		  )
		  AND (
		    c.certificate_expires_at IS NULL
		    OR c.certificate_expires_at > CURRENT_TIMESTAMP
		  )
		ORDER BY c.created_at DESC LIMIT 1`
	err := s.database.QueryRowContext(ctx, query, value).Scan(
		&agent.ID,
		&agent.AgentID,
		&agent.Hostname,
		&agent.Status,
		&agent.Version,
		&agent.OSName,
		&agent.Architecture,
		&agent.AuthMethod,
		&agent.PolicyVersion,
		&blocked,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Agent{}, ErrUnauthorized
	}
	if err != nil {
		s.cacheMu.RLock()
		cached, found := s.cache[credentialColumn+":"+value]
		s.cacheMu.RUnlock()
		if found && s.now().Before(cached.expiresAt) {
			return cached.agent, nil
		}
		return Agent{}, fmt.Errorf("authenticate agent: %w", err)
	}
	if blocked != nil {
		return Agent{}, ErrBlocked
	}
	s.cacheMu.Lock()
	s.cache[credentialColumn+":"+value] = cachedAgent{
		agent:     agent,
		expiresAt: s.now().Add(5 * time.Minute),
	}
	s.cacheMu.Unlock()
	return agent, nil
}

func (s *Service) SetBlocked(
	ctx context.Context,
	internalID string,
	blocked bool,
	actorID string,
) error {
	now := s.now()
	var value any
	status := "provisioned"
	action := "agent.unblock"
	if blocked {
		value = now
		status = "blocked"
		action = "agent.block"
	}
	result, err := s.database.ExecContext(
		ctx,
		`UPDATE agents SET blocked_at = $1, status = $2, updated_at = $3
		 WHERE id = $4`,
		value, status, now, internalID,
	)
	if err != nil {
		return err
	}
	if count, _ := result.RowsAffected(); count == 0 {
		return sql.ErrNoRows
	}
	s.invalidateCache(internalID)
	return s.audit.Record(ctx, s.database, audit.Entry{
		ActorType: "user", ActorID: actorID, Action: action,
		ResourceType: "agent", ResourceID: internalID, Result: "success",
	})
}

func (s *Service) invalidateCache(internalID string) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	for key, cached := range s.cache {
		if cached.agent.ID == internalID {
			delete(s.cache, key)
		}
	}
}

func (s *Service) GetByExternalID(
	ctx context.Context,
	externalID string,
) (Agent, error) {
	var internalID string
	if err := s.database.QueryRowContext(
		ctx,
		`SELECT id FROM agents WHERE agent_id = $1`,
		externalID,
	).Scan(&internalID); err != nil {
		return Agent{}, err
	}
	return s.Get(ctx, internalID)
}

func (s *Service) Get(ctx context.Context, internalID string) (Agent, error) {
	var agent Agent
	err := s.database.QueryRowContext(
		ctx,
		`SELECT id, agent_id, hostname, status, version, os_name,
		 architecture, auth_method, policy_version
		 FROM agents WHERE id = $1`,
		internalID,
	).Scan(
		&agent.ID, &agent.AgentID, &agent.Hostname, &agent.Status,
		&agent.Version, &agent.OSName, &agent.Architecture,
		&agent.AuthMethod, &agent.PolicyVersion,
	)
	return agent, err
}

func randomToken(prefix string, size int) (string, error) {
	bytes := make([]byte, size)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("generate credential: %w", err)
	}
	return prefix + base64.RawURLEncoding.EncodeToString(bytes), nil
}

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

func normalizeFingerprint(value string) string {
	return strings.ToLower(strings.ReplaceAll(strings.TrimSpace(value), ":", ""))
}
