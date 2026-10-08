package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
)

func TestOIDCIdentityIsScopedByIssuer(t *testing.T) {
	runtime, _ := setupAuthUser(t)
	defer runtime.Close()
	service := &OIDCService{db: runtime.DB()}

	firstSettings := oidcProvisioningSettings("https://id.example.test/realms/first")
	first, err := service.provisionUser(
		context.Background(),
		firstSettings,
		"shared-subject",
		oidcProvisioningClaims("first.user", firstSettings.EffectiveIssuer()),
	)
	if err != nil {
		t.Fatalf("provision first issuer: %v", err)
	}

	secondSettings := oidcProvisioningSettings("https://id.example.test/realms/second")
	second, err := service.provisionUser(
		context.Background(),
		secondSettings,
		"shared-subject",
		oidcProvisioningClaims("second.user", secondSettings.EffectiveIssuer()),
	)
	if err != nil {
		t.Fatalf("provision second issuer: %v", err)
	}
	if second.ID == first.ID {
		t.Fatal("a new issuer with the same subject took over the first local user")
	}

	thirdSettings := oidcProvisioningSettings("https://id.example.test/realms/third")
	if _, err := service.provisionUser(
		context.Background(),
		thirdSettings,
		"shared-subject",
		oidcProvisioningClaims("first.user", thirdSettings.EffectiveIssuer()),
	); !errors.Is(err, ErrOIDCUsernameTaken) {
		t.Fatalf("same username from a different issuer error = %v, want ErrOIDCUsernameTaken", err)
	}

	var identities int
	if err := runtime.DB().QueryRow(
		`SELECT COUNT(*) FROM external_identities
		 WHERE provider='keycloak' AND subject='shared-subject'`,
	).Scan(&identities); err != nil {
		t.Fatal(err)
	}
	if identities != 2 {
		t.Fatalf("issuer-scoped identity count = %d, want 2", identities)
	}
}

func TestOIDCLegacyIdentityMigrationRequiresPersistedIssuerProof(t *testing.T) {
	runtime, _ := setupAuthUser(t)
	defer runtime.Close()
	service := &OIDCService{db: runtime.DB()}
	firstIssuer := "https://id.example.test/realms/first"
	secondIssuer := "https://id.example.test/realms/second"

	provenUserID := insertLegacyOIDCUser(
		t,
		runtime.DB(),
		"legacy.proven",
		"legacy-proven-subject",
		fmt.Sprintf(`{"iss":%q,"preferred_username":"legacy.proven"}`, firstIssuer+"/"),
	)
	settings := oidcProvisioningSettings(firstIssuer)
	proven, err := service.provisionUser(
		context.Background(),
		settings,
		"legacy-proven-subject",
		oidcProvisioningClaims("legacy.proven", firstIssuer),
	)
	if err != nil {
		t.Fatalf("migrate proven legacy identity: %v", err)
	}
	if proven.ID != provenUserID {
		t.Fatalf("migrated user ID = %q, want %q", proven.ID, provenUserID)
	}
	var migratedIssuer string
	if err := runtime.DB().QueryRow(
		`SELECT issuer FROM external_identities
		 WHERE provider='keycloak' AND subject='legacy-proven-subject'`,
	).Scan(&migratedIssuer); err != nil {
		t.Fatal(err)
	}
	if migratedIssuer != firstIssuer {
		t.Fatalf("migrated issuer = %q, want %q", migratedIssuer, firstIssuer)
	}

	unprovenUserID := insertLegacyOIDCUser(
		t,
		runtime.DB(),
		"legacy.unproven",
		"legacy-unproven-subject",
		fmt.Sprintf(`{"iss":%q,"preferred_username":"legacy.unproven"}`, firstIssuer),
	)
	secondSettings := oidcProvisioningSettings(secondIssuer)
	separate, err := service.provisionUser(
		context.Background(),
		secondSettings,
		"legacy-unproven-subject",
		oidcProvisioningClaims("second.realm.user", secondIssuer),
	)
	if err != nil {
		t.Fatalf("provision different issuer identity: %v", err)
	}
	if separate.ID == unprovenUserID {
		t.Fatal("different issuer took over the legacy local user")
	}
	var legacyIssuer string
	if err := runtime.DB().QueryRow(
		`SELECT issuer FROM external_identities
		 WHERE user_id=$1 AND provider='keycloak'`,
		unprovenUserID,
	).Scan(&legacyIssuer); err != nil {
		t.Fatal(err)
	}
	if legacyIssuer != "" {
		t.Fatalf("unproven legacy identity issuer = %q, want empty", legacyIssuer)
	}
}

func TestSafeReturnToRejectsAuthorityAndBackslashBypasses(t *testing.T) {
	for _, accepted := range []string{
		"/",
		"/assets",
		"/assets?tab=software#details",
	} {
		if !safeReturnTo(accepted) {
			t.Errorf("safeReturnTo(%q) = false, want true", accepted)
		}
	}
	for _, rejected := range []string{
		"",
		"https://attacker.example/",
		"//attacker.example/",
		`/\attacker.example/`,
		`/assets\next`,
		"/%5c%5cattacker.example/",
		"/%255c%255cattacker.example/",
		"/%252525255c%252525255cattacker.example/",
		"/%2f%2fattacker.example/",
		"/%252f%252fattacker.example/",
		"%2f%2fattacker.example/",
		"/assets%0d%0aLocation:%20https://attacker.example/",
		"/invalid%escape",
	} {
		if safeReturnTo(rejected) {
			t.Errorf("safeReturnTo(%q) = true, want false", rejected)
		}
	}
}

func oidcProvisioningSettings(issuer string) OIDCSettings {
	settings := DefaultOIDCSettings()
	settings.IssuerURL = issuer
	settings.AutoCreateUsers = true
	return settings
}

func oidcProvisioningClaims(username string, issuer string) map[string]any {
	return map[string]any{
		"iss":                issuer,
		"preferred_username": username,
		"email":              username + "@example.test",
		"name":               username,
	}
}

func insertLegacyOIDCUser(
	t *testing.T,
	database *sql.DB,
	username string,
	subject string,
	claimsJSON string,
) string {
	t.Helper()
	userID := uuid.NewString()
	if _, err := database.Exec(
		`INSERT INTO users(
			id, username, normalized_username, display_name, email,
			active, super_admin
		) VALUES ($1,$2,$2,$2,$3,TRUE,FALSE)`,
		userID,
		username,
		username+"@example.test",
	); err != nil {
		t.Fatalf("insert legacy OIDC user: %v", err)
	}
	if _, err := database.Exec(
		`INSERT INTO external_identities(
			id, user_id, provider, issuer, subject, claims_json
		) VALUES ($1,$2,'keycloak','',$3,$4)`,
		uuid.NewString(),
		userID,
		subject,
		claimsJSON,
	); err != nil {
		t.Fatalf("insert legacy OIDC identity: %v", err)
	}
	return userID
}
