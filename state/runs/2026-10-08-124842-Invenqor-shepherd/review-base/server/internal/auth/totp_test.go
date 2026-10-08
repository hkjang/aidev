package auth

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/hkjang/invenqor/server/internal/bootstrap"
)

func TestTOTPSetupEnableLoginRecoveryAndDisable(t *testing.T) {
	runtime, user := setupAuthUser(t)
	defer runtime.Close()
	bootstrapStore, err := bootstrap.Open(filepath.Dir(runtime.SQLitePath()))
	if err != nil {
		t.Fatalf("bootstrap.Open() error = %v", err)
	}
	totpService := NewTOTPService(runtime.DB(), bootstrapStore)
	fixedTime := time.Date(2026, 7, 28, 6, 0, 0, 0, time.UTC)
	totpService.now = func() time.Time { return fixedTime }
	setup, err := totpService.Setup(context.Background(), user)
	if err != nil {
		t.Fatalf("Setup() error = %v", err)
	}
	if setup.Secret == "" ||
		!strings.HasPrefix(setup.ProvisioningURI, "otpauth://totp/") ||
		len(setup.RecoveryCodes) != 10 {
		t.Fatalf("Setup() = %#v", setup)
	}
	var encryptedSecret string
	if err := runtime.DB().QueryRow(
		"SELECT encrypted_secret FROM user_totp WHERE user_id = $1",
		user.ID,
	).Scan(&encryptedSecret); err != nil {
		t.Fatalf("query encrypted TOTP secret error = %v", err)
	}
	if strings.Contains(encryptedSecret, setup.Secret) {
		t.Fatal("database contains the plaintext TOTP secret")
	}
	code, err := generateTOTP(setup.Secret, fixedTime.Unix()/totpPeriod)
	if err != nil {
		t.Fatalf("generateTOTP() error = %v", err)
	}
	if err := totpService.Enable(context.Background(), user.ID, code); err != nil {
		t.Fatalf("Enable() error = %v", err)
	}
	enabled, err := totpService.Enabled(context.Background(), user.ID)
	if err != nil || !enabled {
		t.Fatalf("Enabled() = %v, %v", enabled, err)
	}

	options := DefaultServiceOptions()
	options.TOTP = totpService
	localAuth, err := NewService(runtime.DB(), options)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	login := LoginInput{Username: user.Username, Password: "CorrectHorse!42"}
	if _, err := localAuth.Authenticate(
		context.Background(), login, "192.0.2.80", "test", "request",
	); err != ErrMFARequired {
		t.Fatalf("login without TOTP error = %v, want MFA required", err)
	}
	login.TOTPCode = "000000"
	if _, err := localAuth.Authenticate(
		context.Background(), login, "192.0.2.80", "test", "request",
	); err != ErrMFAInvalid {
		t.Fatalf("login with wrong TOTP error = %v, want MFA invalid", err)
	}
	login.TOTPCode = code
	if _, err := localAuth.Authenticate(
		context.Background(), login, "192.0.2.80", "test", "request",
	); err != nil {
		t.Fatalf("login with TOTP error = %v", err)
	}
	if err := totpService.Verify(
		context.Background(),
		user.ID,
		setup.RecoveryCodes[0],
	); err != nil {
		t.Fatalf("Verify(recovery) error = %v", err)
	}
	if err := totpService.Verify(
		context.Background(),
		user.ID,
		setup.RecoveryCodes[0],
	); err != ErrMFAInvalid {
		t.Fatalf("reused recovery code error = %v, want MFA invalid", err)
	}
	if err := totpService.Disable(context.Background(), user.ID, code); err != nil {
		t.Fatalf("Disable() error = %v", err)
	}
	enabled, err = totpService.Enabled(context.Background(), user.ID)
	if err != nil || enabled {
		t.Fatalf("Enabled() after disable = %v, %v", enabled, err)
	}
}
