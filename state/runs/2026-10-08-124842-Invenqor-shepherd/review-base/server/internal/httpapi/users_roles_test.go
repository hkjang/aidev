package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type httptestRecorder = httptest.ResponseRecorder

// The console renders one role chip per grant, so a user list that drops role
// names makes every account look unprivileged. This exercises enough accounts
// to cover the slice growth that used to invalidate the lookup table.
func TestUserListReportsRolesForEveryAccount(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	expected := map[string]string{}
	for index, role := range []string{
		"viewer",
		"operator",
		"auditor",
		"asset_manager",
		"security_admin",
		"viewer",
		"operator",
		"auditor",
		"asset_manager",
	} {
		username := fmt.Sprintf("member.%02d", index)
		expected[username] = role
		created := performAuthenticatedJSON(
			t, server, http.MethodPost, "/api/v1/admin/users",
			map[string]any{
				"username":     username,
				"display_name": username,
				"password":     "CorrectHorse!42",
				"roles":        []string{role},
			},
			cookie, csrf,
		)
		if created.Code != http.StatusCreated {
			t.Fatalf("create %s status = %d body = %s", username, created.Code, created.Body.String())
		}
	}

	listed := performAuthenticatedJSON(
		t, server, http.MethodGet, "/api/v1/admin/users", nil, cookie, csrf,
	)
	if listed.Code != http.StatusOK {
		t.Fatalf("list status = %d body = %s", listed.Code, listed.Body.String())
	}
	var payload struct {
		Users []struct {
			Username   string   `json:"username"`
			Roles      []string `json:"roles"`
			LocalRoles []string `json:"local_roles"`
			SuperAdmin bool     `json:"super_admin"`
		} `json:"users"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	for _, user := range payload.Users {
		if user.Username == "admin.user" {
			if len(user.Roles) == 0 || user.Roles[0] != "super_admin" {
				t.Errorf("initial administrator roles = %v", user.Roles)
			}
			continue
		}
		role, tracked := expected[user.Username]
		if !tracked {
			continue
		}
		seen[user.Username] = true
		if len(user.Roles) != 1 || user.Roles[0] != role {
			t.Errorf("%s roles = %v, want [%s]", user.Username, user.Roles, role)
		}
		if len(user.LocalRoles) != 1 || user.LocalRoles[0] != role {
			t.Errorf("%s local_roles = %v, want [%s]", user.Username, user.LocalRoles, role)
		}
	}
	for username := range expected {
		if !seen[username] {
			t.Errorf("%s is missing from the user list", username)
		}
	}
}

// A role grant is the only thing that gives an account any permission, so an
// update that removes the last one leaves a user who can log in and do nothing.
func TestUserUpdateRefusesToLeaveAnAccountWithoutAnyRole(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	created := performAuthenticatedJSON(
		t, server, http.MethodPost, "/api/v1/admin/users",
		map[string]any{
			"username": "role.holder",
			"password": "CorrectHorse!42",
			"roles":    []string{"viewer"},
		},
		cookie, csrf,
	)
	if created.Code != http.StatusCreated {
		t.Fatalf("create status = %d body = %s", created.Code, created.Body.String())
	}
	var payload struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(created.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}

	stripped := performAuthenticatedJSON(
		t, server, http.MethodPatch, "/api/v1/admin/users/"+payload.User.ID,
		map[string]any{"roles": []string{}}, cookie, csrf,
	)
	if stripped.Code != http.StatusBadRequest {
		t.Fatalf("empty role update status = %d body = %s", stripped.Code, stripped.Body.String())
	}
	if !containsCode(stripped.Body.String(), "ROLE_REQUIRED") {
		t.Fatalf("expected ROLE_REQUIRED, got %s", stripped.Body.String())
	}

	// The rejected update must not have removed the existing grant.
	listed := performAuthenticatedJSON(
		t, server, http.MethodGet, "/api/v1/admin/users", nil, cookie, csrf,
	)
	if !containsCode(listed.Body.String(), "viewer") {
		t.Fatalf("viewer grant disappeared: %s", listed.Body.String())
	}
}

// Deleting a user must release the account name. The unique constraint is on
// normalized_username, so a soft delete that keeps the name makes it impossible
// to ever recreate the account - and the blocking row is invisible in the UI.
func TestDeletedUsernameCanBeReused(t *testing.T) {
	runtime := newRuntime(t)
	server := testServer(t, runtime)
	cookie, csrf := authenticateInitialAdmin(t, server, runtime)

	create := func() *httptestRecorder {
		return performAuthenticatedJSON(
			t, server, http.MethodPost, "/api/v1/admin/users",
			map[string]any{
				"username": "rotating.member",
				"password": "CorrectHorse!42",
				"roles":    []string{"viewer"},
			},
			cookie, csrf,
		)
	}
	first := create()
	if first.Code != http.StatusCreated {
		t.Fatalf("first create status = %d body = %s", first.Code, first.Body.String())
	}
	var payload struct {
		User struct {
			ID string `json:"id"`
		} `json:"user"`
	}
	if err := json.Unmarshal(first.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	deleted := performAuthenticatedJSON(
		t, server, http.MethodDelete, "/api/v1/admin/users/"+payload.User.ID,
		nil, cookie, csrf,
	)
	if deleted.Code != http.StatusNoContent {
		t.Fatalf("delete status = %d body = %s", deleted.Code, deleted.Body.String())
	}
	second := create()
	if second.Code != http.StatusCreated {
		t.Fatalf("recreate status = %d body = %s", second.Code, second.Body.String())
	}
	// The deleted account must stay out of the list while keeping its audit row.
	listed := performAuthenticatedJSON(
		t, server, http.MethodGet, "/api/v1/admin/users", nil, cookie, csrf,
	)
	var listing struct {
		Users []struct {
			ID       string `json:"id"`
			Username string `json:"username"`
		} `json:"users"`
	}
	if err := json.Unmarshal(listed.Body.Bytes(), &listing); err != nil {
		t.Fatal(err)
	}
	matches := 0
	for _, user := range listing.Users {
		if user.Username == "rotating.member" {
			matches++
		}
		if user.ID == payload.User.ID {
			t.Fatal("the deleted account is still listed")
		}
	}
	if matches != 1 {
		t.Fatalf("expected exactly one active rotating.member, found %d", matches)
	}
}

func containsCode(body string, needle string) bool {
	return strings.Contains(body, needle)
}
