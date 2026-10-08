package auth

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/hkjang/invenqor/server/internal/storagetest"
)

func TestConfiguredBootstrapCreatesInitialAdministratorOnce(t *testing.T) {
	root := t.TempDir()
	runtime := storagetest.Open(t)
	defer runtime.Close()
	manager := NewBootstrapManager(runtime.DB(), root)
	status, err := manager.Ensure(context.Background())
	if err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if !status.Required {
		t.Fatal("Ensure() did not require initial setup")
	}
	user, err := manager.CreateInitialAdminFromConfig(
		context.Background(),
		InitialAdminInput{
			Username:    "bootstrap.admin",
			Password:    "CorrectHorse!42",
			DisplayName: "Bootstrap Admin",
		},
	)
	if err != nil {
		t.Fatalf("CreateInitialAdminFromConfig() error = %v", err)
	}
	if user.Username != "bootstrap.admin" || !user.SuperAdmin {
		t.Fatalf("unexpected bootstrap user: %#v", user)
	}
	if _, err := os.Stat(manager.TokenPath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("bootstrap token file still exists, stat error = %v", err)
	}
	if _, err := manager.CreateInitialAdminFromConfig(
		context.Background(),
		InitialAdminInput{
			Username: "second.admin",
			Password: "AnotherHorse!42",
		},
	); !errors.Is(err, ErrBootstrapComplete) {
		t.Fatalf("second configured bootstrap error = %v, want complete", err)
	}
	service, err := NewService(runtime.DB(), DefaultServiceOptions())
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	if _, err := service.Authenticate(
		context.Background(),
		LoginInput{Username: "bootstrap.admin", Password: "CorrectHorse!42"},
		"192.0.2.20",
		"test",
		"login",
	); err != nil {
		t.Fatalf("Authenticate() error = %v", err)
	}
	var storedHash string
	if err := runtime.DB().QueryRow(
		"SELECT password_hash FROM users WHERE username = $1",
		"bootstrap.admin",
	).Scan(&storedHash); err != nil {
		t.Fatalf("read password hash: %v", err)
	}
	if storedHash == "CorrectHorse!42" {
		t.Fatal("bootstrap password was stored as plaintext")
	}
}

func TestConfiguredBootstrapValidationFailurePreservesTokenClaim(t *testing.T) {
	root := t.TempDir()
	runtime := storagetest.Open(t)
	defer runtime.Close()
	manager := NewBootstrapManager(runtime.DB(), root)
	if _, err := manager.Ensure(context.Background()); err != nil {
		t.Fatalf("Ensure() error = %v", err)
	}
	if _, err := manager.CreateInitialAdminFromConfig(
		context.Background(),
		InitialAdminInput{Username: "bootstrap.admin", Password: "weak"},
	); err == nil {
		t.Fatal("configured bootstrap accepted a weak password")
	}
	if _, err := os.Stat(manager.TokenPath()); err != nil {
		t.Fatalf("bootstrap token file was removed after validation failure: %v", err)
	}
	var claims int
	if err := runtime.DB().QueryRow(
		"SELECT COUNT(*) FROM server_metadata WHERE key = 'bootstrap_token_hash'",
	).Scan(&claims); err != nil {
		t.Fatalf("count bootstrap claims: %v", err)
	}
	if claims != 1 {
		t.Fatalf("bootstrap claim count = %d, want 1", claims)
	}
}
