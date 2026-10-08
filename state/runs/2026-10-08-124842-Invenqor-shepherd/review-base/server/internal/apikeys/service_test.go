package apikeys_test

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/hkjang/invenqor/server/internal/storagetest"

	"github.com/google/uuid"
	"github.com/hkjang/invenqor/server/internal/apikeys"
)

func TestAPIKeyScopeLifecycleRotationAndRevocation(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	userID := uuid.NewString()
	_, err := runtime.DB().Exec(
		`INSERT INTO users(
		 id, username, normalized_username, display_name, active, super_admin
		 ) VALUES ($1,'key.owner','key.owner','Key Owner',TRUE,TRUE)`, userID,
	)
	if err != nil {
		t.Fatal(err)
	}
	service := apikeys.NewService(runtime.DB())
	created, err := service.Create(
		context.Background(), userID, "automation",
		[]string{"mcp.access", "assets.read"}, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if created.Secret == "" || created.Key.Prefix == "" ||
		len(created.Key.Scopes) != 2 {
		t.Fatalf("created = %#v", created)
	}
	credential, err := service.Authenticate(context.Background(), created.Secret)
	if err != nil || credential.KeyID != created.Key.ID {
		t.Fatalf("Authenticate() = %#v, %v", credential, err)
	}
	invalidName := "must-not-be-partially-saved"
	invalidScopes := []string{"root.everything"}
	if _, err := service.Update(
		context.Background(),
		created.Key.ID,
		&invalidName,
		&invalidScopes,
	); !errors.Is(err, apikeys.ErrInvalid) {
		t.Fatalf("Update() invalid scope error = %v", err)
	}
	unchanged, err := service.Get(context.Background(), created.Key.ID)
	if err != nil {
		t.Fatal(err)
	}
	if unchanged.Name != "automation" {
		t.Fatalf("invalid atomic update changed name to %q", unchanged.Name)
	}
	updatedName := "automation-renamed"
	replacementScopes := []string{"assets.read", "mcp.access"}
	updatedTogether, err := service.Update(
		context.Background(),
		created.Key.ID,
		&updatedName,
		&replacementScopes,
	)
	if err != nil || updatedTogether.Name != updatedName ||
		len(updatedTogether.Scopes) != 2 {
		t.Fatalf("atomic Update() = %#v, %v", updatedTogether, err)
	}

	updated, err := service.AddScopes(
		context.Background(), created.Key.ID, []string{"agents.read"},
	)
	if err != nil || len(updated.Scopes) != 3 {
		t.Fatalf("AddScopes() = %#v, %v", updated, err)
	}
	updated, err = service.RemoveScope(
		context.Background(), created.Key.ID, "assets.read",
	)
	if err != nil || len(updated.Scopes) != 2 {
		t.Fatalf("RemoveScope() = %#v, %v", updated, err)
	}

	rotated, err := service.Rotate(
		context.Background(), created.Key.ID, time.Hour,
		apikeys.Access{UserID: userID, SuperAdmin: true},
	)
	if err != nil || rotated.Secret == created.Secret {
		t.Fatalf("Rotate() = %#v, %v", rotated, err)
	}
	if _, err := service.Authenticate(context.Background(), created.Secret); err != nil {
		t.Fatalf("old key rejected during grace: %v", err)
	}
	if _, err := service.Authenticate(context.Background(), rotated.Secret); err != nil {
		t.Fatalf("new key rejected: %v", err)
	}

	if err := service.Revoke(context.Background(), created.Key.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Authenticate(context.Background(), rotated.Secret); !errors.Is(err, apikeys.ErrUnauthorized) {
		t.Fatalf("revoked key error = %v", err)
	}
}

func TestConcurrentAPIKeyScopeAdditionsDoNotLoseUpdates(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	userID := uuid.NewString()
	if _, err := runtime.DB().Exec(
		`INSERT INTO users(
		 id, username, normalized_username, display_name, active, super_admin
		 ) VALUES ($1,'concurrent.owner','concurrent.owner','Concurrent Owner',TRUE,TRUE)`,
		userID,
	); err != nil {
		t.Fatal(err)
	}
	services := []*apikeys.Service{
		apikeys.NewService(runtime.DB()),
		apikeys.NewService(runtime.DB()),
		apikeys.NewService(runtime.DB()),
		apikeys.NewService(runtime.DB()),
	}
	additions := []string{
		"agents.read",
		"assets.read",
		"assets.write",
		"relations.read",
	}
	for iteration := range 25 {
		created, err := services[0].Create(
			context.Background(),
			userID,
			"concurrent-scope-update",
			[]string{"mcp.access"},
			nil,
		)
		if err != nil {
			t.Fatal(err)
		}
		start := make(chan struct{})
		errorsByWorker := make(chan error, len(additions))
		var workers sync.WaitGroup
		for index, scope := range additions {
			workers.Add(1)
			go func(service *apikeys.Service, addition string) {
				defer workers.Done()
				<-start
				_, err := service.AddScopes(
					context.Background(),
					created.Key.ID,
					[]string{addition},
				)
				errorsByWorker <- err
			}(services[index], scope)
		}
		close(start)
		workers.Wait()
		close(errorsByWorker)
		for err := range errorsByWorker {
			if err != nil {
				t.Fatalf("iteration %d concurrent AddScopes() error = %v", iteration, err)
			}
		}
		key, err := services[0].Get(context.Background(), created.Key.ID)
		if err != nil {
			t.Fatal(err)
		}
		if len(key.Scopes) != len(additions)+1 {
			t.Fatalf(
				"iteration %d scopes = %v, want all concurrent additions",
				iteration,
				key.Scopes,
			)
		}
	}
}

// Create refuses a key with no scopes because such a key authenticates and is
// then refused by every scope check. ReplaceScopes and RemoveScope reach the
// same stored state, so they have to refuse it too.
func TestAPIKeyScopeListCannotBeEmptied(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	userID := uuid.NewString()
	if _, err := runtime.DB().Exec(
		`INSERT INTO users(
		 id, username, normalized_username, display_name, active, super_admin
		 ) VALUES ($1,'empty.owner','empty.owner','Empty Owner',TRUE,TRUE)`,
		userID,
	); err != nil {
		t.Fatal(err)
	}
	service := apikeys.NewService(runtime.DB())
	created, err := service.Create(
		context.Background(), userID, "reporting", []string{"assets.read"}, nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ReplaceScopes(
		context.Background(), created.Key.ID, []string{},
	); !errors.Is(err, apikeys.ErrScopesRequired) {
		t.Fatalf("ReplaceScopes() with an empty list error = %v", err)
	}
	if _, err := service.RemoveScope(
		context.Background(), created.Key.ID, "assets.read",
	); !errors.Is(err, apikeys.ErrScopesRequired) {
		t.Fatalf("RemoveScope() of the last scope error = %v", err)
	}
	kept, err := service.Get(context.Background(), created.Key.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(kept.Scopes) != 1 || kept.Scopes[0] != "assets.read" {
		t.Fatalf("scopes after refused removals = %v", kept.Scopes)
	}
	if !errors.Is(apikeys.ErrScopesRequired, apikeys.ErrInvalid) {
		t.Fatal("ErrScopesRequired must stay reportable as invalid input")
	}
}

func TestAPIKeyRejectsUnknownScopesAndExpiredKeys(t *testing.T) {
	runtime := storagetest.Open(t)
	defer runtime.Close()
	userID := uuid.NewString()
	_, _ = runtime.DB().Exec(
		`INSERT INTO users(
		 id, username, normalized_username, display_name, active, super_admin
		 ) VALUES ($1,'key.owner','key.owner','Key Owner',TRUE,TRUE)`, userID,
	)
	service := apikeys.NewService(runtime.DB())
	if _, err := service.Create(context.Background(), userID, "bad",
		[]string{"root.everything"}, nil); !errors.Is(err, apikeys.ErrInvalid) {
		t.Fatalf("unknown scope error = %v", err)
	}
	past := time.Now().Add(-time.Minute)
	if _, err := service.Create(context.Background(), userID, "expired",
		[]string{"assets.read"}, &past); !errors.Is(err, apikeys.ErrInvalid) {
		t.Fatalf("past expiry error = %v", err)
	}
}
