package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/base32"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/hkjang/invenqor/server/internal/bootstrap"
)

const (
	totpDigits = 6
	totpPeriod = 30
)

type TOTPService struct {
	db        *sql.DB
	protector *bootstrap.Store
	now       func() time.Time
}

func NewTOTPService(db *sql.DB, protector *bootstrap.Store) *TOTPService {
	return &TOTPService{
		db:        db,
		protector: protector,
		now:       time.Now,
	}
}

func (service *TOTPService) Enabled(ctx context.Context, userID string) (bool, error) {
	var enabled bool
	if err := service.db.QueryRowContext(
		ctx,
		"SELECT enabled FROM user_totp WHERE user_id = $1",
		userID,
	).Scan(&enabled); errors.Is(err, sql.ErrNoRows) {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("read TOTP status: %w", err)
	}
	return enabled, nil
}

func (service *TOTPService) Setup(
	ctx context.Context,
	user User,
) (TOTPSetup, error) {
	enabled, err := service.Enabled(ctx, user.ID)
	if err != nil {
		return TOTPSetup{}, err
	}
	if enabled {
		return TOTPSetup{}, ErrMFAAlreadyEnabled
	}
	secretBytes := make([]byte, 20)
	if _, err := io.ReadFull(rand.Reader, secretBytes); err != nil {
		return TOTPSetup{}, fmt.Errorf("generate TOTP secret: %w", err)
	}
	secret := base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(secretBytes)
	encrypted, err := service.protector.SealString("totp:"+user.ID, secret)
	if err != nil {
		return TOTPSetup{}, err
	}
	recoveryCodes, recoveryHashes, err := generateRecoveryCodes(10)
	if err != nil {
		return TOTPSetup{}, err
	}
	recoveryJSON, err := json.Marshal(recoveryHashes)
	if err != nil {
		return TOTPSetup{}, fmt.Errorf("encode TOTP recovery hashes: %w", err)
	}
	if _, err := service.db.ExecContext(
		ctx,
		`INSERT INTO user_totp(
			user_id, encrypted_secret, enabled, recovery_codes_json, updated_at
		) VALUES ($1, $2, FALSE, $3, CURRENT_TIMESTAMP)
		ON CONFLICT (user_id) DO UPDATE SET
		  encrypted_secret = excluded.encrypted_secret,
		  enabled = FALSE,
		  recovery_codes_json = excluded.recovery_codes_json,
		  verified_at = NULL,
		  updated_at = CURRENT_TIMESTAMP`,
		user.ID,
		encrypted,
		string(recoveryJSON),
	); err != nil {
		return TOTPSetup{}, fmt.Errorf("store TOTP setup: %w", err)
	}
	label := "Invenqor:" + user.Username
	query := url.Values{
		"secret":    []string{secret},
		"issuer":    []string{"Invenqor"},
		"algorithm": []string{"SHA1"},
		"digits":    []string{strconv.Itoa(totpDigits)},
		"period":    []string{strconv.Itoa(totpPeriod)},
	}
	return TOTPSetup{
		Secret:          secret,
		ProvisioningURI: "otpauth://totp/" + url.PathEscape(label) + "?" + query.Encode(),
		RecoveryCodes:   recoveryCodes,
	}, nil
}

func (service *TOTPService) Enable(
	ctx context.Context,
	userID string,
	code string,
) error {
	// Enrolment must prove the authenticator itself works, so a recovery code is
	// not acceptable here: accepting one enabled MFA for a user whose
	// authenticator was never registered, and left that code unconsumed.
	valid, err := service.verifyAuthenticator(ctx, userID, code, false)
	if err != nil {
		return err
	}
	if !valid {
		return ErrMFAInvalid
	}
	result, err := service.db.ExecContext(
		ctx,
		`UPDATE user_totp
		 SET enabled = TRUE, verified_at = CURRENT_TIMESTAMP,
		     updated_at = CURRENT_TIMESTAMP
		 WHERE user_id = $1 AND enabled = FALSE`,
		userID,
	)
	if err != nil {
		return fmt.Errorf("enable TOTP: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return ErrMFASetupRequired
	}
	return nil
}

func (service *TOTPService) Verify(
	ctx context.Context,
	userID string,
	code string,
) error {
	valid, recoveryHashes, err := service.verifyStored(ctx, userID, code, true, true)
	if err != nil {
		return err
	}
	if !valid {
		return ErrMFAInvalid
	}
	if recoveryHashes != nil {
		bytes, err := json.Marshal(recoveryHashes)
		if err != nil {
			return fmt.Errorf("encode consumed recovery codes: %w", err)
		}
		if _, err := service.db.ExecContext(
			ctx,
			`UPDATE user_totp
			 SET recovery_codes_json = $1, updated_at = CURRENT_TIMESTAMP
			 WHERE user_id = $2 AND enabled = TRUE`,
			string(bytes),
			userID,
		); err != nil {
			return fmt.Errorf("consume recovery code: %w", err)
		}
	}
	return nil
}

// RegenerateRecoveryCodes issues a fresh set and invalidates the old ones. A
// user who has spent their codes previously had to disable MFA and enrol again,
// which meant a window with no second factor at all.
func (service *TOTPService) RegenerateRecoveryCodes(
	ctx context.Context,
	userID string,
	code string,
) ([]string, error) {
	// The authenticator itself must answer: a recovery code cannot mint more
	// recovery codes, or one leaked code would be enough to keep an intruder in
	// indefinitely.
	valid, err := service.verifyAuthenticator(ctx, userID, code, true)
	if err != nil {
		return nil, err
	}
	if !valid {
		return nil, ErrMFAInvalid
	}
	codes, hashes, err := generateRecoveryCodes(10)
	if err != nil {
		return nil, err
	}
	encoded, err := json.Marshal(hashes)
	if err != nil {
		return nil, fmt.Errorf("encode recovery hashes: %w", err)
	}
	result, err := service.db.ExecContext(
		ctx,
		`UPDATE user_totp
		 SET recovery_codes_json = $1, updated_at = CURRENT_TIMESTAMP
		 WHERE user_id = $2 AND enabled = TRUE`,
		string(encoded),
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("store new recovery codes: %w", err)
	}
	if affected, _ := result.RowsAffected(); affected != 1 {
		return nil, ErrMFASetupRequired
	}
	return codes, nil
}

func (service *TOTPService) Disable(
	ctx context.Context,
	userID string,
	code string,
) error {
	if err := service.Verify(ctx, userID, code); err != nil {
		return err
	}
	if _, err := service.db.ExecContext(
		ctx,
		"DELETE FROM user_totp WHERE user_id = $1",
		userID,
	); err != nil {
		return fmt.Errorf("disable TOTP: %w", err)
	}
	return nil
}

// verifyAuthenticator accepts only a code the authenticator app can produce.
func (service *TOTPService) verifyAuthenticator(
	ctx context.Context,
	userID string,
	code string,
	requireEnabled bool,
) (bool, error) {
	valid, _, err := service.verifyStored(ctx, userID, code, requireEnabled, false)
	return valid, err
}

// Status reports what the console needs to describe the account's second
// factor without guessing: whether it is on, since when, and how many recovery
// codes are still usable.
func (service *TOTPService) Status(
	ctx context.Context,
	userID string,
) (TOTPStatus, error) {
	var status TOTPStatus
	// The SQLite start-up fallback hands timestamps back as text, so the shared
	// flexible scanner rather than sql.NullTime.
	var verifiedAt flexibleTime
	var rawRecovery any
	if err := service.db.QueryRowContext(
		ctx,
		`SELECT enabled, verified_at, recovery_codes_json
		 FROM user_totp WHERE user_id = $1`,
		userID,
	).Scan(&status.Enabled, &verifiedAt, &rawRecovery); errors.Is(err, sql.ErrNoRows) {
		return TOTPStatus{}, nil
	} else if err != nil {
		return TOTPStatus{}, fmt.Errorf("read TOTP status: %w", err)
	}
	if verifiedAt.Valid {
		verified := verifiedAt.Time.UTC()
		status.VerifiedAt = &verified
	}
	recoveryBytes, err := jsonBytes(rawRecovery)
	if err != nil {
		return TOTPStatus{}, err
	}
	var recoveryHashes []string
	if err := json.Unmarshal(recoveryBytes, &recoveryHashes); err != nil {
		return TOTPStatus{}, fmt.Errorf("decode TOTP recovery hashes: %w", err)
	}
	// Only an enabled second factor has usable recovery codes; an abandoned
	// setup row must not report ten codes the user cannot use.
	if status.Enabled {
		status.RecoveryCodesRemaining = len(recoveryHashes)
	}
	return status, nil
}

func (service *TOTPService) verifyStored(
	ctx context.Context,
	userID string,
	code string,
	requireEnabled bool,
	allowRecovery bool,
) (bool, []string, error) {
	var encryptedSecret string
	var enabled bool
	var rawRecovery any
	if err := service.db.QueryRowContext(
		ctx,
		`SELECT encrypted_secret, enabled, recovery_codes_json
		 FROM user_totp WHERE user_id = $1`,
		userID,
	).Scan(&encryptedSecret, &enabled, &rawRecovery); errors.Is(err, sql.ErrNoRows) {
		return false, nil, ErrMFASetupRequired
	} else if err != nil {
		return false, nil, fmt.Errorf("load TOTP secret: %w", err)
	}
	if requireEnabled && !enabled {
		return false, nil, ErrMFASetupRequired
	}
	secret, err := service.protector.OpenString("totp:"+userID, encryptedSecret)
	if err != nil {
		return false, nil, err
	}
	code = strings.TrimSpace(strings.ToUpper(code))
	if len(code) == totpDigits {
		for offset := int64(-1); offset <= 1; offset++ {
			expected, err := generateTOTP(secret, service.now().UTC().Unix()/totpPeriod+offset)
			if err != nil {
				return false, nil, err
			}
			if subtle.ConstantTimeCompare([]byte(code), []byte(expected)) == 1 {
				return true, nil, nil
			}
		}
	}
	if !allowRecovery {
		return false, nil, nil
	}
	recoveryBytes, err := jsonBytes(rawRecovery)
	if err != nil {
		return false, nil, err
	}
	var recoveryHashes []string
	if err := json.Unmarshal(recoveryBytes, &recoveryHashes); err != nil {
		return false, nil, fmt.Errorf("decode TOTP recovery hashes: %w", err)
	}
	actualHash := sha256.Sum256([]byte(code))
	actual := hex.EncodeToString(actualHash[:])
	for index, expected := range recoveryHashes {
		if subtle.ConstantTimeCompare([]byte(actual), []byte(expected)) == 1 {
			return true, append(recoveryHashes[:index], recoveryHashes[index+1:]...), nil
		}
	}
	return false, nil, nil
}

func generateTOTP(secret string, counter int64) (string, error) {
	decoded, err := base32.StdEncoding.WithPadding(base32.NoPadding).DecodeString(
		strings.ToUpper(secret),
	)
	if err != nil {
		return "", errors.New("decode TOTP secret")
	}
	counterBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(counterBytes, uint64(counter))
	mac := hmac.New(sha1.New, decoded)
	_, _ = mac.Write(counterBytes)
	sum := mac.Sum(nil)
	offset := sum[len(sum)-1] & 0x0f
	value := binary.BigEndian.Uint32(sum[offset:offset+4]) & 0x7fffffff
	code := value % 1_000_000
	return fmt.Sprintf("%06d", code), nil
}

func generateRecoveryCodes(count int) ([]string, []string, error) {
	codes := make([]string, 0, count)
	hashes := make([]string, 0, count)
	for index := 0; index < count; index++ {
		bytes := make([]byte, 10)
		if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
			return nil, nil, fmt.Errorf("generate recovery code: %w", err)
		}
		encoded := strings.ToUpper(base32.StdEncoding.WithPadding(base32.NoPadding).EncodeToString(bytes))
		code := encoded[:4] + "-" + encoded[4:8] + "-" + encoded[8:12] + "-" + encoded[12:]
		hash := sha256.Sum256([]byte(code))
		codes = append(codes, code)
		hashes = append(hashes, hex.EncodeToString(hash[:]))
	}
	return codes, hashes, nil
}
