package httpapi

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/mail"
	"sort"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/hkjang/invenqor/server/internal/apitime"
	"github.com/hkjang/invenqor/server/internal/auth"
	notify "github.com/hkjang/invenqor/server/internal/mail"
)

type userCreateInput struct {
	Username    string   `json:"username"`
	DisplayName string   `json:"display_name"`
	Email       string   `json:"email"`
	Password    string   `json:"password"`
	Roles       []string `json:"roles"`
	Reason      string   `json:"reason"`
}

type userUpdateInput struct {
	DisplayName *string   `json:"display_name"`
	Email       *string   `json:"email"`
	Active      *bool     `json:"active"`
	Roles       *[]string `json:"roles"`
	Reason      string    `json:"reason"`
}

type roleRecord struct {
	ID          string
	Name        string
	Description string
}

type managedUserRecord struct {
	ID          string
	Username    string
	DisplayName string
	Email       string
	Active      bool
	SuperAdmin  bool
	Locked      bool
	Provider    string
	CreatedAt   apitime.Time
	UpdatedAt   apitime.Time
	Roles       []string
	LocalRoles  []string
	OIDCRoles   []string
}

func (s *Server) listUsers(response http.ResponseWriter, request *http.Request) {
	rows, err := s.database.DB().QueryContext(
		request.Context(),
		`SELECT u.id,u.username,u.display_name,u.email,u.active,u.super_admin,
		        CASE WHEN u.locked_until IS NOT NULL
		                  AND u.locked_until > CURRENT_TIMESTAMP
		             THEN TRUE ELSE FALSE END,
		        CASE WHEN EXISTS(
		                 SELECT 1 FROM external_identities e
		                  WHERE e.user_id=u.id AND e.provider='keycloak'
		             ) THEN 'keycloak' ELSE 'local' END,
		        u.created_at,u.updated_at
		   FROM users u
		  WHERE u.deleted_at IS NULL
		  ORDER BY u.normalized_username`,
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	defer rows.Close()
	users := make([]managedUserRecord, 0)
	for rows.Next() {
		user := managedUserRecord{
			Roles:      make([]string, 0),
			LocalRoles: make([]string, 0),
			OIDCRoles:  make([]string, 0),
		}
		if err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.DisplayName,
			&user.Email,
			&user.Active,
			&user.SuperAdmin,
			&user.Locked,
			&user.Provider,
			&user.CreatedAt,
			&user.UpdatedAt,
		); err != nil {
			s.internalError(response, request, err)
			return
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		s.internalError(response, request, err)
		return
	}
	if err := rows.Close(); err != nil {
		s.internalError(response, request, err)
		return
	}
	roleRows, err := s.database.DB().QueryContext(
		request.Context(),
		`SELECT ur.user_id,r.name,ur.source
		   FROM user_roles ur JOIN roles r ON r.id=ur.role_id
		  ORDER BY r.name,ur.source`,
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	defer roleRows.Close()
	// Collect grants by user identifier rather than by pointer: `users` is still
	// growing while these rows are read, and a slice that reallocates leaves any
	// retained element pointer addressing a discarded array.
	type grants struct {
		all      []string
		local    []string
		keycloak []string
	}
	byID := make(map[string]*grants, len(users))
	for roleRows.Next() {
		var userID, role, source string
		if err := roleRows.Scan(&userID, &role, &source); err != nil {
			s.internalError(response, request, err)
			return
		}
		entry := byID[userID]
		if entry == nil {
			entry = &grants{}
			byID[userID] = entry
		}
		entry.all = appendUniqueRole(entry.all, role)
		if source == "keycloak" {
			entry.keycloak = appendUniqueRole(entry.keycloak, role)
		} else {
			entry.local = appendUniqueRole(entry.local, role)
		}
	}
	if err := roleRows.Err(); err != nil {
		s.internalError(response, request, err)
		return
	}
	items := make([]map[string]any, 0, len(users))
	for _, user := range users {
		if entry := byID[user.ID]; entry != nil {
			user.Roles = appendUniqueRoles(user.Roles, entry.all...)
			user.LocalRoles = appendUniqueRoles(user.LocalRoles, entry.local...)
			user.OIDCRoles = appendUniqueRoles(user.OIDCRoles, entry.keycloak...)
		}
		items = append(items, managedUserJSON(user))
	}
	writeJSON(response, http.StatusOK, map[string]any{"users": items})
}

func (s *Server) listRoles(response http.ResponseWriter, request *http.Request) {
	rows, err := s.database.DB().QueryContext(
		request.Context(),
		`SELECT r.id,r.name,r.description,r.system_role,
		        COALESCE(rp.permission_name,'')
		   FROM roles r
		   LEFT JOIN role_permissions rp ON rp.role_id=r.id
		  ORDER BY r.name,rp.permission_name`,
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	defer rows.Close()
	items := make([]map[string]any, 0)
	var currentID string
	var current map[string]any
	var permissions []string
	for rows.Next() {
		var id, name, description, permission string
		var systemRole bool
		if err := rows.Scan(
			&id,
			&name,
			&description,
			&systemRole,
			&permission,
		); err != nil {
			s.internalError(response, request, err)
			return
		}
		if id != currentID {
			if current != nil {
				current["permissions"] = permissions
				items = append(items, current)
			}
			currentID = id
			permissions = make([]string, 0)
			current = map[string]any{
				"id":          id,
				"name":        name,
				"description": description,
				"system_role": systemRole,
			}
		}
		if permission != "" {
			permissions = append(permissions, permission)
		}
	}
	if err := rows.Err(); err != nil {
		s.internalError(response, request, err)
		return
	}
	if current != nil {
		current["permissions"] = permissions
		items = append(items, current)
	}
	writeJSON(response, http.StatusOK, map[string]any{"roles": items})
}

func (s *Server) createUser(response http.ResponseWriter, request *http.Request) {
	var input userCreateInput
	if err := decodeJSON(request, &input); err != nil {
		writeAPIError(response, request, http.StatusBadRequest, "INVALID_REQUEST", "The request body is invalid.")
		return
	}
	username, err := auth.ValidateUsername(input.Username)
	if err != nil || !validManagedEmail(input.Email) {
		writeAPIError(response, request, http.StatusBadRequest, "INVALID_USER", "The user profile is invalid.")
		return
	}
	if err := auth.DefaultPasswordPolicy().Validate(input.Password); err != nil {
		writeAPIError(response, request, http.StatusBadRequest, "INVALID_PASSWORD", "The password does not satisfy policy.")
		return
	}
	passwordHash, err := auth.HashPassword(input.Password)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	transaction, err := s.database.DB().BeginTx(request.Context(), nil)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	defer transaction.Rollback()
	roles, err := resolveManagedRoles(request.Context(), transaction, input.Roles)
	if err != nil {
		writeAPIError(response, request, http.StatusBadRequest, "INVALID_ROLES", "One or more roles are invalid.")
		return
	}
	if len(roles) == 0 {
		writeAPIError(response, request, http.StatusBadRequest, "ROLE_REQUIRED", "At least one role is required.")
		return
	}
	var exists int
	if err := transaction.QueryRowContext(
		request.Context(),
		"SELECT COUNT(*) FROM users WHERE normalized_username=$1",
		auth.NormalizeUsername(username),
	).Scan(&exists); err != nil {
		s.internalError(response, request, err)
		return
	}
	if exists != 0 {
		writeAPIError(response, request, http.StatusConflict, "USERNAME_EXISTS", "The username already exists.")
		return
	}
	userID := uuid.NewString()
	superAdmin := containsManagedRole(roles, "super_admin")
	if _, err := transaction.ExecContext(
		request.Context(),
		`INSERT INTO users(
		    id,username,normalized_username,display_name,email,password_hash,
		    active,super_admin,password_changed_at
		 ) VALUES($1,$2,$3,$4,$5,$6,TRUE,$7,CURRENT_TIMESTAMP)`,
		userID,
		username,
		auth.NormalizeUsername(username),
		strings.TrimSpace(input.DisplayName),
		strings.TrimSpace(input.Email),
		passwordHash,
		superAdmin,
	); err != nil {
		s.internalError(response, request, err)
		return
	}
	if err := replaceManagedRoles(
		request.Context(),
		transaction,
		userID,
		roles,
	); err != nil {
		s.internalError(response, request, err)
		return
	}
	if err := transaction.Commit(); err != nil {
		s.internalError(response, request, err)
		return
	}
	s.recordAdminAudit(
		request,
		"user.create",
		"user",
		userID,
		nil,
		map[string]any{
			"username": username,
			"roles":    roleNames(roles),
			"active":   true,
		},
		input.Reason,
	)
	s.notifyMail(
		request.Context(),
		notify.AccountCreated(
			username, strings.TrimSpace(input.DisplayName), actorLabel(request),
			roleNames(roles),
		),
		principalFromContext(request.Context()).User.ID,
		[]string{userID},
	)
	writeJSON(response, http.StatusCreated, map[string]any{
		"user": map[string]any{
			"id":           userID,
			"username":     username,
			"display_name": strings.TrimSpace(input.DisplayName),
			"email":        strings.TrimSpace(input.Email),
			"active":       true,
			"super_admin":  superAdmin,
			"locked":       false,
			"provider":     "local",
			"roles":        roleNames(roles),
			"local_roles":  roleNames(roles),
			"oidc_roles":   []string{},
		},
	})
}

func (s *Server) updateUser(response http.ResponseWriter, request *http.Request) {
	userID := chi.URLParam(request, "userID")
	var input userUpdateInput
	if userID == "" || decodeJSON(request, &input) != nil {
		writeAPIError(response, request, http.StatusBadRequest, "INVALID_REQUEST", "The request body is invalid.")
		return
	}
	transaction, err := s.database.DB().BeginTx(request.Context(), nil)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	defer transaction.Rollback()
	current, err := loadManagedUser(request.Context(), transaction, userID)
	if errors.Is(err, sql.ErrNoRows) {
		writeAPIError(response, request, http.StatusNotFound, "USER_NOT_FOUND", "The user does not exist.")
		return
	}
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	displayName := current.DisplayName
	email := current.Email
	active := current.Active
	superAdmin := current.SuperAdmin
	var roles []roleRecord
	if input.DisplayName != nil {
		displayName = strings.TrimSpace(*input.DisplayName)
	}
	if input.Email != nil {
		email = strings.TrimSpace(*input.Email)
		if !validManagedEmail(email) {
			writeAPIError(response, request, http.StatusBadRequest, "INVALID_EMAIL", "The email address is invalid.")
			return
		}
	}
	if input.Active != nil {
		active = *input.Active
	}
	if input.Roles != nil {
		roles, err = resolveManagedRoles(request.Context(), transaction, *input.Roles)
		if err != nil {
			writeAPIError(response, request, http.StatusBadRequest, "INVALID_ROLES", "One or more roles are invalid.")
			return
		}
		// Permissions come only from role grants, so an account left with none
		// can sign in and reach nothing. Creation already refuses this; the
		// update path has to refuse it too, unless Keycloak still grants a role.
		if len(roles) == 0 && len(current.OIDCRoles) == 0 {
			writeAPIError(response, request, http.StatusBadRequest, "ROLE_REQUIRED", "At least one role is required.")
			return
		}
		externalSuperAdmin, err := managedRoleFromSource(
			request.Context(),
			transaction,
			userID,
			"keycloak",
			"super_admin",
		)
		if err != nil {
			s.internalError(response, request, err)
			return
		}
		superAdmin = containsManagedRole(roles, "super_admin") || externalSuperAdmin
	}
	actorID := principalFromContext(request.Context()).User.ID
	if actorID == userID && (!active || (current.SuperAdmin && !superAdmin)) {
		writeAPIError(response, request, http.StatusConflict, "SELF_LOCKOUT", "You cannot deactivate or demote your own account.")
		return
	}
	if current.Active && current.SuperAdmin && (!active || !superAdmin) {
		if err := requireAnotherActiveSuperAdmin(request.Context(), transaction, userID); err != nil {
			if errors.Is(err, errLastSuperAdmin) {
				writeAPIError(response, request, http.StatusConflict, "LAST_SUPER_ADMIN", "At least one active Super Admin is required.")
				return
			}
			s.internalError(response, request, err)
			return
		}
	}
	if _, err := transaction.ExecContext(
		request.Context(),
		`UPDATE users SET display_name=$1,email=$2,active=$3,super_admin=$4,
		        failed_login_count=CASE WHEN $3 THEN failed_login_count ELSE 0 END,
		        locked_until=CASE WHEN $3 THEN locked_until ELSE NULL END,
		        updated_at=CURRENT_TIMESTAMP
		  WHERE id=$5 AND deleted_at IS NULL`,
		displayName,
		email,
		active,
		superAdmin,
		userID,
	); err != nil {
		s.internalError(response, request, err)
		return
	}
	if input.Roles != nil {
		if err := replaceManagedRoles(
			request.Context(),
			transaction,
			userID,
			roles,
		); err != nil {
			s.internalError(response, request, err)
			return
		}
	}
	if !active {
		if err := revokeManagedUserAccess(
			request.Context(),
			transaction,
			userID,
		); err != nil {
			s.internalError(response, request, err)
			return
		}
	}
	if err := transaction.Commit(); err != nil {
		s.internalError(response, request, err)
		return
	}
	afterRoles := current.Roles
	if input.Roles != nil {
		afterRoles = appendUniqueRoles(roleNames(roles), current.OIDCRoles...)
	}
	s.recordAdminAudit(
		request,
		"user.update",
		"user",
		userID,
		managedUserJSON(current),
		map[string]any{
			"id":           userID,
			"username":     current.Username,
			"display_name": displayName,
			"email":        email,
			"active":       active,
			"super_admin":  superAdmin,
			"roles":        afterRoles,
		},
		input.Reason,
	)
	writeJSON(response, http.StatusOK, map[string]any{"updated": true})
}

func (s *Server) resetUserPassword(
	response http.ResponseWriter,
	request *http.Request,
) {
	userID := chi.URLParam(request, "userID")
	var input struct {
		Password string `json:"password"`
		Reason   string `json:"reason"`
	}
	if userID == "" || decodeJSON(request, &input) != nil {
		writeAPIError(response, request, http.StatusBadRequest, "INVALID_REQUEST", "The request body is invalid.")
		return
	}
	if err := auth.DefaultPasswordPolicy().Validate(input.Password); err != nil {
		writeAPIError(response, request, http.StatusBadRequest, "INVALID_PASSWORD", "The password does not satisfy policy.")
		return
	}
	passwordHash, err := auth.HashPassword(input.Password)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	transaction, err := s.database.DB().BeginTx(request.Context(), nil)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	defer transaction.Rollback()
	var provider string
	err = transaction.QueryRowContext(
		request.Context(),
		`SELECT CASE WHEN EXISTS(
		            SELECT 1 FROM external_identities e
		             WHERE e.user_id=u.id AND e.provider='keycloak'
		        ) THEN 'keycloak' ELSE 'local' END
		   FROM users u WHERE u.id=$1 AND u.deleted_at IS NULL`,
		userID,
	).Scan(&provider)
	if errors.Is(err, sql.ErrNoRows) {
		writeAPIError(response, request, http.StatusNotFound, "USER_NOT_FOUND", "The user does not exist.")
		return
	}
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	if provider != "local" {
		writeAPIError(response, request, http.StatusConflict, "EXTERNAL_USER", "Keycloak user passwords are managed by the identity provider.")
		return
	}
	if _, err := transaction.ExecContext(
		request.Context(),
		`UPDATE users SET password_hash=$1,password_changed_at=CURRENT_TIMESTAMP,
		        failed_login_count=0,locked_until=NULL,updated_at=CURRENT_TIMESTAMP
		  WHERE id=$2`,
		passwordHash,
		userID,
	); err != nil {
		s.internalError(response, request, err)
		return
	}
	if err := revokeManagedUserSessions(
		request.Context(),
		transaction,
		userID,
	); err != nil {
		s.internalError(response, request, err)
		return
	}
	if err := transaction.Commit(); err != nil {
		s.internalError(response, request, err)
		return
	}
	s.recordAdminAudit(
		request,
		"user.password.reset",
		"user",
		userID,
		nil,
		map[string]any{"sessions_revoked": true},
		input.Reason,
	)
	writeJSON(response, http.StatusOK, map[string]any{
		"reset":            true,
		"sessions_revoked": true,
	})
}

func (s *Server) unlockUser(response http.ResponseWriter, request *http.Request) {
	userID := chi.URLParam(request, "userID")
	result, err := s.database.DB().ExecContext(
		request.Context(),
		`UPDATE users SET failed_login_count=0,locked_until=NULL,
		        updated_at=CURRENT_TIMESTAMP
		  WHERE id=$1 AND deleted_at IS NULL`,
		userID,
	)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	if rows, _ := result.RowsAffected(); rows != 1 {
		writeAPIError(response, request, http.StatusNotFound, "USER_NOT_FOUND", "The user does not exist.")
		return
	}
	s.recordAdminAudit(
		request,
		"user.unlock",
		"user",
		userID,
		nil,
		map[string]any{"locked": false},
		"",
	)
	var username, displayName string
	if err := s.database.DB().QueryRowContext(
		request.Context(),
		`SELECT username,display_name FROM users WHERE id=$1`, userID,
	).Scan(&username, &displayName); err == nil {
		s.notifyMail(
			request.Context(),
			notify.AccountUnlocked(username, displayName, actorLabel(request)),
			principalFromContext(request.Context()).User.ID,
			[]string{userID},
		)
	}
	writeJSON(response, http.StatusOK, map[string]any{"unlocked": true})
}

func (s *Server) deleteUser(response http.ResponseWriter, request *http.Request) {
	userID := chi.URLParam(request, "userID")
	actorID := principalFromContext(request.Context()).User.ID
	if userID == actorID {
		writeAPIError(response, request, http.StatusConflict, "SELF_DELETE", "You cannot delete your own account.")
		return
	}
	transaction, err := s.database.DB().BeginTx(request.Context(), nil)
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	defer transaction.Rollback()
	current, err := loadManagedUser(request.Context(), transaction, userID)
	if errors.Is(err, sql.ErrNoRows) {
		writeAPIError(response, request, http.StatusNotFound, "USER_NOT_FOUND", "The user does not exist.")
		return
	}
	if err != nil {
		s.internalError(response, request, err)
		return
	}
	if current.Active && current.SuperAdmin {
		if err := requireAnotherActiveSuperAdmin(request.Context(), transaction, userID); err != nil {
			if errors.Is(err, errLastSuperAdmin) {
				writeAPIError(response, request, http.StatusConflict, "LAST_SUPER_ADMIN", "At least one active Super Admin is required.")
				return
			}
			s.internalError(response, request, err)
			return
		}
	}
	// normalized_username is globally unique, so a soft delete that keeps the
	// name makes the account impossible to recreate while the blocking row stays
	// invisible in the console. Release the name and keep the original in the
	// audit record and in the retired form.
	retired := retiredUsername(current.Username, userID)
	if _, err := transaction.ExecContext(
		request.Context(),
		`UPDATE users SET active=FALSE,deleted_at=CURRENT_TIMESTAMP,
		        username=$2,normalized_username=$3,
		        updated_at=CURRENT_TIMESTAMP
		  WHERE id=$1`,
		userID,
		retired,
		auth.NormalizeUsername(retired),
	); err != nil {
		s.internalError(response, request, err)
		return
	}
	if err := revokeManagedUserAccess(
		request.Context(),
		transaction,
		userID,
	); err != nil {
		s.internalError(response, request, err)
		return
	}
	if err := transaction.Commit(); err != nil {
		s.internalError(response, request, err)
		return
	}
	s.recordAdminAudit(
		request,
		"user.delete",
		"user",
		userID,
		managedUserJSON(current),
		map[string]any{
			"deleted":           true,
			"username_released": current.Username,
			"retired_username":  retired,
		},
		"",
	)
	response.WriteHeader(http.StatusNoContent)
}

// retiredUsername keeps the original name readable while freeing it for reuse.
// The '#' cannot appear in a valid account name, so a retired row can never
// collide with a live one.
func retiredUsername(username string, userID string) string {
	suffix := "#deleted-" + strings.ReplaceAll(userID, "-", "")
	// normalized_username has no length limit in the schema, but keeping the
	// visible part bounded avoids surprising audit output.
	if len(username) > 64 {
		username = username[:64]
	}
	return username + suffix
}

func resolveManagedRoles(
	ctx context.Context,
	transaction *sql.Tx,
	names []string,
) ([]roleRecord, error) {
	unique := make(map[string]struct{}, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		unique[name] = struct{}{}
	}
	roles := make([]roleRecord, 0, len(unique))
	for name := range unique {
		var role roleRecord
		err := transaction.QueryRowContext(
			ctx,
			"SELECT id,name,description FROM roles WHERE name=$1",
			name,
		).Scan(&role.ID, &role.Name, &role.Description)
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("role does not exist")
		}
		if err != nil {
			return nil, err
		}
		roles = append(roles, role)
	}
	sort.Slice(roles, func(i, j int) bool {
		return roles[i].Name < roles[j].Name
	})
	return roles, nil
}

func replaceManagedRoles(
	ctx context.Context,
	transaction *sql.Tx,
	userID string,
	roles []roleRecord,
) error {
	if _, err := transaction.ExecContext(
		ctx,
		"DELETE FROM user_roles WHERE user_id=$1 AND source='local'",
		userID,
	); err != nil {
		return err
	}
	for _, role := range roles {
		if _, err := transaction.ExecContext(
			ctx,
			`INSERT INTO user_roles(user_id,role_id,source)
			 VALUES($1,$2,'local')
			 ON CONFLICT(user_id,role_id,source) DO NOTHING`,
			userID,
			role.ID,
		); err != nil {
			return err
		}
	}
	return nil
}

var errLastSuperAdmin = errors.New("last active super administrator")

func requireAnotherActiveSuperAdmin(
	ctx context.Context,
	transaction *sql.Tx,
	excludedUserID string,
) error {
	var count int
	if err := transaction.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM users
		  WHERE active=TRUE AND super_admin=TRUE AND deleted_at IS NULL
		    AND id<>$1`,
		excludedUserID,
	).Scan(&count); err != nil {
		return err
	}
	if count == 0 {
		return errLastSuperAdmin
	}
	return nil
}

func loadManagedUser(
	ctx context.Context,
	transaction *sql.Tx,
	userID string,
) (managedUserRecord, error) {
	var user managedUserRecord
	err := transaction.QueryRowContext(
		ctx,
		`SELECT u.id,u.username,u.display_name,u.email,u.active,u.super_admin,
		        CASE WHEN u.locked_until IS NOT NULL
		                  AND u.locked_until > CURRENT_TIMESTAMP
		             THEN TRUE ELSE FALSE END,
		        CASE WHEN EXISTS(
		                 SELECT 1 FROM external_identities e
		                  WHERE e.user_id=u.id AND e.provider='keycloak'
		             ) THEN 'keycloak' ELSE 'local' END,
		        u.created_at,u.updated_at
		   FROM users u WHERE u.id=$1 AND u.deleted_at IS NULL`,
		userID,
	).Scan(
		&user.ID,
		&user.Username,
		&user.DisplayName,
		&user.Email,
		&user.Active,
		&user.SuperAdmin,
		&user.Locked,
		&user.Provider,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		return user, err
	}
	rows, err := transaction.QueryContext(
		ctx,
		`SELECT r.name,ur.source FROM user_roles ur JOIN roles r ON r.id=ur.role_id
		  WHERE ur.user_id=$1 ORDER BY r.name`,
		userID,
	)
	if err != nil {
		return user, err
	}
	defer rows.Close()
	for rows.Next() {
		var role, source string
		if err := rows.Scan(&role, &source); err != nil {
			return user, err
		}
		user.Roles = appendUniqueRole(user.Roles, role)
		if source == "keycloak" {
			user.OIDCRoles = appendUniqueRole(user.OIDCRoles, role)
		} else {
			user.LocalRoles = appendUniqueRole(user.LocalRoles, role)
		}
	}
	return user, rows.Err()
}

func revokeManagedUserSessions(
	ctx context.Context,
	transaction *sql.Tx,
	userID string,
) error {
	_, err := transaction.ExecContext(
		ctx,
		`UPDATE sessions SET revoked_at=CURRENT_TIMESTAMP
		  WHERE user_id=$1 AND revoked_at IS NULL`,
		userID,
	)
	return err
}

func revokeManagedUserAccess(
	ctx context.Context,
	transaction *sql.Tx,
	userID string,
) error {
	if err := revokeManagedUserSessions(ctx, transaction, userID); err != nil {
		return err
	}
	_, err := transaction.ExecContext(
		ctx,
		`UPDATE api_keys SET revoked_at=CURRENT_TIMESTAMP
		  WHERE user_id=$1 AND revoked_at IS NULL`,
		userID,
	)
	return err
}

func validManagedEmail(value string) bool {
	value = strings.TrimSpace(value)
	if value == "" {
		return true
	}
	address, err := mail.ParseAddress(value)
	return err == nil && strings.EqualFold(address.Address, value)
}

func containsManagedRole(roles []roleRecord, target string) bool {
	for _, role := range roles {
		if role.Name == target {
			return true
		}
	}
	return false
}

func managedRoleFromSource(
	ctx context.Context,
	transaction *sql.Tx,
	userID string,
	source string,
	role string,
) (bool, error) {
	var count int
	if err := transaction.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM user_roles ur JOIN roles r ON r.id=ur.role_id
		  WHERE ur.user_id=$1 AND ur.source=$2 AND r.name=$3`,
		userID,
		source,
		role,
	).Scan(&count); err != nil {
		return false, err
	}
	return count > 0, nil
}

func appendUniqueRole(roles []string, role string) []string {
	for _, existing := range roles {
		if existing == role {
			return roles
		}
	}
	return append(roles, role)
}

func appendUniqueRoles(roles []string, values ...string) []string {
	for _, role := range values {
		roles = appendUniqueRole(roles, role)
	}
	sort.Strings(roles)
	return roles
}

func roleNames(roles []roleRecord) []string {
	names := make([]string, 0, len(roles))
	for _, role := range roles {
		names = append(names, role.Name)
	}
	return names
}

func managedUserJSON(user managedUserRecord) map[string]any {
	return map[string]any{
		"id":           user.ID,
		"username":     user.Username,
		"display_name": user.DisplayName,
		"email":        user.Email,
		"active":       user.Active,
		"super_admin":  user.SuperAdmin,
		"locked":       user.Locked,
		"provider":     user.Provider,
		"roles":        user.Roles,
		"local_roles":  user.LocalRoles,
		"oidc_roles":   user.OIDCRoles,
		"created_at":   user.CreatedAt,
		"updated_at":   user.UpdatedAt,
	}
}
