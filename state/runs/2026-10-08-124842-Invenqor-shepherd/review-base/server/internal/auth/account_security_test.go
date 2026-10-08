package auth

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/hkjang/invenqor/server/internal/bootstrap"
	"github.com/hkjang/invenqor/server/internal/storage"
)

// A Keycloak-provisioned account has no local password_hash. Reading it into a
// plain string failed the scan, so the console received a 500 and the server
// logged an internal error for what is an ordinary, expected situation.
func TestChangePasswordRefusesAnAccountWithoutALocalPassword(t *testing.T) {
	runtime, user := setupAuthUser(t)
	defer runtime.Close()
	service, err := NewService(runtime.DB(), DefaultServiceOptions())
	if err != nil {
		t.Fatal(err)
	}
	federated := insertFederatedUser(t, runtime)

	err = service.ChangePassword(
		context.Background(),
		Principal{User: federated},
		ChangePasswordInput{
			CurrentPassword: "anything",
			NewPassword:     "BrandNewSecret!42",
		},
		"192.0.2.7", "test", "request-federated",
	)
	if !errors.Is(err, ErrPasswordUnavailable) {
		t.Fatalf("ChangePassword() error = %v, want ErrPasswordUnavailable", err)
	}

	// The local account must still be able to change its password.
	if err := service.ChangePassword(
		context.Background(),
		Principal{User: user},
		ChangePasswordInput{
			CurrentPassword: "CorrectHorse!42",
			NewPassword:     "BrandNewSecret!42",
		},
		"192.0.2.7", "test", "request-local",
	); err != nil {
		t.Fatalf("ChangePassword() for a local account error = %v", err)
	}
}

// A wrong TOTP code left the lockout counter and the IP rate limiter untouched,
// so an attacker holding a valid password could guess the second factor without
// limit - and recovery codes are accepted at the same prompt.
func TestFailedTOTPCodeCountsTowardTheLockout(t *testing.T) {
	runtime, user := setupAuthUser(t)
	defer runtime.Close()
	totpService := enabledTOTP(t, runtime, user)
	options := DefaultServiceOptions()
	options.TOTP = totpService
	options.LockoutThreshold = 3
	options.LockoutDuration = time.Minute
	service, err := NewService(runtime.DB(), options)
	if err != nil {
		t.Fatal(err)
	}
	attempt := LoginInput{
		Username: user.Username,
		Password: "CorrectHorse!42",
		TOTPCode: "000000",
	}
	for index := 1; index <= 2; index++ {
		if _, err := service.Authenticate(
			context.Background(), attempt, "192.0.2.60", "test", "request",
		); !errors.Is(err, ErrMFAInvalid) {
			t.Fatalf("attempt %d error = %v, want ErrMFAInvalid", index, err)
		}
	}
	if _, err := service.Authenticate(
		context.Background(), attempt, "192.0.2.60", "test", "request",
	); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("third wrong code error = %v, want ErrAccountLocked", err)
	}
	// The lock must hold even when the correct second factor arrives next.
	var count int
	if err := runtime.DB().QueryRow(
		"SELECT COUNT(*) FROM login_attempts WHERE source_ip = $1 AND succeeded = FALSE",
		"192.0.2.60",
	).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 3 {
		t.Fatalf("recorded %d failed attempts, want 3 - the IP rate limiter never sees MFA failures", count)
	}
}

// Enrolment must prove the authenticator works. Accepting a recovery code as
// proof enabled MFA for a user whose authenticator was never registered, and
// left the code unconsumed, so the user locked themselves out of the console.
func TestEnableRejectsARecoveryCodeAsEnrolmentProof(t *testing.T) {
	runtime, user := setupAuthUser(t)
	defer runtime.Close()
	bootstrapStore, err := bootstrap.Open(filepath.Dir(runtime.SQLitePath()))
	if err != nil {
		t.Fatal(err)
	}
	totpService := NewTOTPService(runtime.DB(), bootstrapStore)
	setup, err := totpService.Setup(context.Background(), user)
	if err != nil {
		t.Fatal(err)
	}
	if err := totpService.Enable(
		context.Background(), user.ID, setup.RecoveryCodes[0],
	); !errors.Is(err, ErrMFAInvalid) {
		t.Fatalf("Enable() with a recovery code error = %v, want ErrMFAInvalid", err)
	}
	enabled, err := totpService.Enabled(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if enabled {
		t.Fatal("a recovery code enabled MFA without a working authenticator")
	}
	code, err := generateTOTP(setup.Secret, time.Now().UTC().Unix()/totpPeriod)
	if err != nil {
		t.Fatal(err)
	}
	if err := totpService.Enable(context.Background(), user.ID, code); err != nil {
		t.Fatalf("Enable() with an authenticator code error = %v", err)
	}
}

// The console cannot show what it cannot read: without this the account page
// offered "disable MFA" to users who had never enabled it.
func TestSecurityStatusReportsWhatTheConsoleMustShow(t *testing.T) {
	runtime, user := setupAuthUser(t)
	defer runtime.Close()
	totpService := enabledTOTP(t, runtime, user)
	status, err := totpService.Status(context.Background(), user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if !status.Enabled || status.RecoveryCodesRemaining != 10 {
		t.Fatalf("Status() = %+v", status)
	}
	federated := insertFederatedUser(t, runtime)
	federatedStatus, err := totpService.Status(context.Background(), federated.ID)
	if err != nil {
		t.Fatal(err)
	}
	if federatedStatus.Enabled || federatedStatus.RecoveryCodesRemaining != 0 {
		t.Fatalf("Status() for an account without MFA = %+v", federatedStatus)
	}

	service, err := NewService(runtime.DB(), DefaultServiceOptions())
	if err != nil {
		t.Fatal(err)
	}
	local, err := service.PasswordConfigured(context.Background(), user.ID)
	if err != nil || !local {
		t.Fatalf("PasswordConfigured() for a local account = %v, %v", local, err)
	}
	remote, err := service.PasswordConfigured(context.Background(), federated.ID)
	if err != nil || remote {
		t.Fatalf("PasswordConfigured() for a federated account = %v, %v", remote, err)
	}
}

func insertFederatedUser(t *testing.T, runtime *storage.Runtime) User {
	t.Helper()
	user := User{
		ID:          "6f1f2f9e-7d55-4c5b-9c1e-4a5a71c1f001",
		Username:    "keycloak.user",
		DisplayName: "Keycloak User",
		Email:       "keycloak.user@example.com",
	}
	if _, err := runtime.DB().Exec(
		`INSERT INTO users(id, username, normalized_username, display_name,
		 email, active, super_admin)
		 VALUES ($1, $2, $3, $4, $5, TRUE, FALSE)`,
		user.ID, user.Username, user.Username, user.DisplayName, user.Email,
	); err != nil {
		t.Fatalf("insert federated user error = %v", err)
	}
	return user
}

func enabledTOTP(t *testing.T, runtime *storage.Runtime, user User) *TOTPService {
	t.Helper()
	bootstrapStore, err := bootstrap.Open(filepath.Dir(runtime.SQLitePath()))
	if err != nil {
		t.Fatal(err)
	}
	service := NewTOTPService(runtime.DB(), bootstrapStore)
	setup, err := service.Setup(context.Background(), user)
	if err != nil {
		t.Fatal(err)
	}
	code, err := generateTOTP(setup.Secret, time.Now().UTC().Unix()/totpPeriod)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Enable(context.Background(), user.ID, code); err != nil {
		t.Fatal(err)
	}
	return service
}
