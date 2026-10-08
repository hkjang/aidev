package auth

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/hkjang/invenqor/server/internal/storagetest"

	"github.com/hkjang/invenqor/server/internal/storage"
)

func TestLocalLoginLocksAccountAndRecoversAfterWindow(t *testing.T) {
	runtime, user := setupAuthUser(t)
	defer runtime.Close()
	options := DefaultServiceOptions()
	options.LockoutThreshold = 2
	options.LockoutDuration = time.Minute
	service, err := NewService(runtime.DB(), options)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	input := LoginInput{Username: user.Username, Password: "wrong-password"}
	if _, err := service.Authenticate(
		context.Background(), input, "192.0.2.10", "test", "request-1",
	); !errors.Is(err, ErrInvalidCredentials) {
		t.Fatalf("first bad login error = %v, want invalid credentials", err)
	}
	if _, err := service.Authenticate(
		context.Background(), input, "192.0.2.10", "test", "request-2",
	); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("second bad login error = %v, want account locked", err)
	}
	if _, err := service.Authenticate(
		context.Background(),
		LoginInput{Username: user.Username, Password: "CorrectHorse!42"},
		"192.0.2.10",
		"test",
		"request-3",
	); !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("locked account login error = %v, want account locked", err)
	}
	if _, err := runtime.DB().Exec(
		"UPDATE users SET locked_until = $1 WHERE id = $2",
		time.Now().Add(-time.Minute),
		user.ID,
	); err != nil {
		t.Fatalf("expire lock error = %v", err)
	}
	session, err := service.Authenticate(
		context.Background(),
		LoginInput{Username: strings.ToUpper(user.Username), Password: "CorrectHorse!42"},
		"192.0.2.10",
		"test",
		"request-4",
	)
	if err != nil {
		t.Fatalf("login after lock expiry error = %v", err)
	}
	if session.Token == "" || session.CSRFToken == "" {
		t.Fatal("successful login omitted secure session tokens")
	}
}

func TestLoginIPRateLimitAlsoCoversUnknownUsers(t *testing.T) {
	runtime, _ := setupAuthUser(t)
	defer runtime.Close()
	options := DefaultServiceOptions()
	options.IPRateThreshold = 2
	service, err := NewService(runtime.DB(), options)
	if err != nil {
		t.Fatalf("NewService() error = %v", err)
	}
	for index := 0; index < 2; index++ {
		_, err := service.Authenticate(
			context.Background(),
			LoginInput{Username: "unknown-user", Password: "WrongPassword!42"},
			"192.0.2.50",
			"test",
			"request",
		)
		if !errors.Is(err, ErrInvalidCredentials) {
			t.Fatalf("unknown user attempt %d error = %v", index, err)
		}
	}
	_, err = service.Authenticate(
		context.Background(),
		LoginInput{Username: "another-user", Password: "WrongPassword!42"},
		"192.0.2.50",
		"test",
		"request",
	)
	if !errors.Is(err, ErrAccountLocked) {
		t.Fatalf("rate-limited login error = %v, want account locked", err)
	}
}

func setupAuthUser(t *testing.T) (*storage.Runtime, User) {
	t.Helper()
	root := t.TempDir()
	runtime := storagetest.Open(t)
	manager := NewBootstrapManager(runtime.DB(), root)
	if _, err := manager.Ensure(context.Background()); err != nil {
		runtime.Close()
		t.Fatalf("BootstrapManager.Ensure() error = %v", err)
	}
	token, err := os.ReadFile(manager.TokenPath())
	if err != nil {
		runtime.Close()
		t.Fatalf("read bootstrap token error = %v", err)
	}
	user, err := manager.CreateInitialAdmin(
		context.Background(),
		strings.TrimSpace(string(token)),
		InitialAdminInput{
			Username:    "local.admin",
			Password:    "CorrectHorse!42",
			DisplayName: "Local Admin",
		},
		"192.0.2.1",
		"test",
		"request",
	)
	if err != nil {
		runtime.Close()
		t.Fatalf("CreateInitialAdmin() error = %v", err)
	}
	return runtime, user
}
