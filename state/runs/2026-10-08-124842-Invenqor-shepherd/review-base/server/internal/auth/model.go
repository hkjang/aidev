package auth

import (
	"errors"
	"time"
)

var (
	ErrBootstrapComplete  = errors.New("initial setup is already complete")
	ErrBootstrapToken     = errors.New("invalid bootstrap token")
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrAccountLocked      = errors.New("account is temporarily locked")
	ErrAccountInactive    = errors.New("account is inactive")
	ErrUnauthorized       = errors.New("authentication is required")
	ErrCSRF               = errors.New("CSRF validation failed")
	ErrPasswordUnchanged  = errors.New("new password must differ from current password")
	// ErrPasswordUnavailable covers accounts that authenticate through an
	// identity provider and therefore hold no local password to change.
	ErrPasswordUnavailable = errors.New("account has no local password")
	ErrMFARequired         = errors.New("TOTP code is required")
	ErrMFAInvalid          = errors.New("TOTP code is invalid")
	ErrMFAAlreadyEnabled   = errors.New("TOTP is already enabled")
	ErrMFASetupRequired    = errors.New("TOTP setup has not started")
)

const (
	superAdminRoleID = "00000000-0000-0000-0000-000000000001"
	SessionCookie    = "invenqor_session"
	CSRFCookie       = "invenqor_csrf"
)

type User struct {
	ID          string   `json:"id"`
	Username    string   `json:"username"`
	DisplayName string   `json:"display_name"`
	Email       string   `json:"email"`
	SuperAdmin  bool     `json:"super_admin"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

type Session struct {
	ID                string
	Token             string
	CSRFToken         string
	User              User
	IdleExpiresAt     time.Time
	AbsoluteExpiresAt time.Time
}

type Principal struct {
	SessionID         string
	User              User
	CSRFHash          string
	IdleExpiresAt     time.Time
	AbsoluteExpiresAt time.Time
}

func (principal Principal) HasPermission(permission string) bool {
	if principal.User.SuperAdmin {
		return true
	}
	for _, candidate := range principal.User.Permissions {
		if candidate == permission {
			return true
		}
	}
	return false
}

type InitialAdminInput struct {
	Username    string `json:"username"`
	Password    string `json:"password"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
}

type LoginInput struct {
	Username string `json:"username"`
	Password string `json:"password"`
	TOTPCode string `json:"totp_code,omitempty"`
}

type ChangePasswordInput struct {
	CurrentPassword string `json:"current_password"`
	NewPassword     string `json:"new_password"`
}

type BootstrapStatus struct {
	Required  bool   `json:"required"`
	TokenFile string `json:"token_file,omitempty"`
}

type TOTPSetup struct {
	Secret          string   `json:"secret"`
	ProvisioningURI string   `json:"provisioning_uri"`
	RecoveryCodes   []string `json:"recovery_codes"`
}

// TOTPStatus is what the console needs to describe an account's second factor.
type TOTPStatus struct {
	Enabled                bool       `json:"enabled"`
	VerifiedAt             *time.Time `json:"verified_at,omitempty"`
	RecoveryCodesRemaining int        `json:"recovery_codes_remaining"`
}
