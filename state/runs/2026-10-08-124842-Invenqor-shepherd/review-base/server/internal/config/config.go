package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	defaultListenAddress = "127.0.0.1:7070"
	defaultStateDir      = "/var/lib/invenqor-server"
)

// Config contains only process bootstrap settings. Runtime settings live in the
// metadata database and are edited through the administrative API.
type Config struct {
	ListenAddress          string
	BaseURL                string
	StateDir               string
	SQLitePath             string
	PostgresDSN            string
	DatabaseSchema         string
	DatabaseTimeout        time.Duration
	ShutdownTimeout        time.Duration
	MasterKeyPath          string
	UpdateDir              string
	EventSpoolDir          string
	PostgresDSNFromEnv     bool
	BootstrapAdmin         string
	BootstrapAdminPassword string
	AgentAutoEnrollment    bool
	AgentEnrollmentToken   string
	// UpdateSigningPublicKey lets the server verify an update signature when it
	// is published instead of discovering the mistake on every agent later.
	UpdateSigningPublicKey string
}

func Load() (Config, error) {
	stateDir := envOrDefault("INVENQOR_STATE_DIR", defaultStateDir)
	postgresDSN, postgresDSNFromEnv := firstTrimmedEnvWithPresence(
		"INVENQOR_POSTGRES_DSN",
		"POSTGRES_DSN",
		"postgres_dsn",
	)
	bootstrapAdmin, bootstrapPassword, err := bootstrapCredentials()
	if err != nil {
		return Config{}, err
	}
	agentEnrollmentToken, err := secretFromEnvironment(
		[]string{
			"INVENQOR_AGENT_ENROLLMENT_TOKEN",
			"AGENT_ENROLLMENT_TOKEN",
			"agent_enrollment_token",
		},
		[]string{
			"INVENQOR_AGENT_ENROLLMENT_TOKEN_FILE",
			"AGENT_ENROLLMENT_TOKEN_FILE",
			"agent_enrollment_token_file",
		},
	)
	if err != nil {
		return Config{}, fmt.Errorf("agent enrollment token: %w", err)
	}
	agentAutoEnrollment, err := boolEnvironment(
		true,
		"INVENQOR_AGENT_AUTO_ENROLLMENT",
		"AGENT_AUTO_ENROLLMENT",
		"agent_auto_enrollment",
	)
	if err != nil {
		return Config{}, err
	}
	updateSigningKey, err := secretFromEnvironment(
		[]string{
			"INVENQOR_UPDATE_PUBLIC_KEY",
			"UPDATE_PUBLIC_KEY",
			"update_public_key",
		},
		[]string{
			"INVENQOR_UPDATE_PUBLIC_KEY_FILE",
			"UPDATE_PUBLIC_KEY_FILE",
			"update_public_key_file",
		},
	)
	if err != nil {
		return Config{}, fmt.Errorf("update signing public key: %w", err)
	}
	config := Config{
		ListenAddress:          envOrDefault("INVENQOR_LISTEN_ADDRESS", defaultListenAddress),
		BaseURL:                strings.TrimRight(os.Getenv("INVENQOR_BASE_URL"), "/"),
		StateDir:               stateDir,
		SQLitePath:             envOrDefault("INVENQOR_SQLITE_PATH", filepath.Join(stateDir, "invenqor.db")),
		PostgresDSN:            postgresDSN,
		DatabaseSchema:         envOrDefault("INVENQOR_POSTGRES_SCHEMA", "public"),
		DatabaseTimeout:        durationEnv("INVENQOR_DATABASE_TIMEOUT", 5*time.Second),
		ShutdownTimeout:        durationEnv("INVENQOR_SHUTDOWN_TIMEOUT", 15*time.Second),
		MasterKeyPath:          strings.TrimSpace(os.Getenv("INVENQOR_MASTER_KEY_FILE")),
		UpdateDir:              envOrDefault("INVENQOR_UPDATE_DIR", filepath.Join(stateDir, "updates")),
		EventSpoolDir:          envOrDefault("INVENQOR_EVENT_SPOOL_DIR", filepath.Join(stateDir, "spool")),
		PostgresDSNFromEnv:     postgresDSNFromEnv,
		BootstrapAdmin:         bootstrapAdmin,
		BootstrapAdminPassword: bootstrapPassword,
		AgentAutoEnrollment:    agentAutoEnrollment,
		AgentEnrollmentToken:   agentEnrollmentToken,
		UpdateSigningPublicKey: updateSigningKey,
	}
	if err := config.Validate(); err != nil {
		return Config{}, err
	}
	return config, nil
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.ListenAddress) == "" {
		return errors.New("listen address is required")
	}
	if _, _, err := net.SplitHostPort(c.ListenAddress); err != nil {
		return fmt.Errorf("invalid listen address: %w", err)
	}
	if !filepath.IsAbs(c.StateDir) {
		return errors.New("state directory must be absolute")
	}
	if !filepath.IsAbs(c.SQLitePath) {
		return errors.New("SQLite path must be absolute")
	}
	if c.DatabaseTimeout <= 0 {
		return errors.New("database timeout must be positive")
	}
	if c.ShutdownTimeout <= 0 {
		return errors.New("shutdown timeout must be positive")
	}
	if c.MasterKeyPath != "" && !filepath.IsAbs(c.MasterKeyPath) {
		return errors.New("master key file path must be absolute")
	}
	if c.UpdateDir != "" && !filepath.IsAbs(c.UpdateDir) {
		return errors.New("update directory must be absolute")
	}
	if c.EventSpoolDir != "" && !filepath.IsAbs(c.EventSpoolDir) {
		return errors.New("event spool directory must be absolute")
	}
	if (c.BootstrapAdmin == "") != (c.BootstrapAdminPassword == "") {
		return errors.New("bootstrap administrator and password must be configured together")
	}
	if c.AgentEnrollmentToken != "" && len(c.AgentEnrollmentToken) < 32 {
		return errors.New("agent enrollment token must contain at least 32 characters")
	}
	if c.BaseURL != "" {
		parsed, err := url.Parse(c.BaseURL)
		if err != nil || parsed.Host == "" {
			return errors.New("base URL is invalid")
		}
		if parsed.Scheme != "https" && parsed.Scheme != "http" {
			return errors.New("base URL must use HTTP or HTTPS")
		}
	}
	if !validSchemaName(c.DatabaseSchema) {
		return errors.New("PostgreSQL schema contains invalid characters")
	}
	return nil
}

func envOrDefault(name, fallback string) string {
	if value := strings.TrimSpace(os.Getenv(name)); value != "" {
		return value
	}
	return fallback
}

func bootstrapCredentials() (string, string, error) {
	admin := firstTrimmedEnv(
		"INVENQOR_BOOTSTRAP_ADMIN",
		"BOOTSTRAP_ADMIN",
		"bootstrap_admin",
	)
	password := firstEnv(
		"INVENQOR_BOOTSTRAP_ADMIN_PASSWORD",
		"BOOTSTRAP_ADMIN_PASSWORD",
		"bootstrap_admin_password",
	)
	passwordFile := firstTrimmedEnv(
		"INVENQOR_BOOTSTRAP_ADMIN_PASSWORD_FILE",
		"BOOTSTRAP_ADMIN_PASSWORD_FILE",
		"bootstrap_admin_password_file",
	)
	if password != "" && passwordFile != "" {
		return "", "", errors.New(
			"bootstrap administrator password and password file cannot both be configured",
		)
	}
	if passwordFile != "" {
		if !filepath.IsAbs(passwordFile) {
			return "", "", errors.New("bootstrap administrator password file must be absolute")
		}
		bytes, err := os.ReadFile(passwordFile)
		if err != nil {
			return "", "", fmt.Errorf("read bootstrap administrator password file: %w", err)
		}
		password = strings.TrimRight(string(bytes), "\r\n")
	}
	return admin, password, nil
}

func secretFromEnvironment(valueNames, fileNames []string) (string, error) {
	value := firstEnv(valueNames...)
	file := firstTrimmedEnv(fileNames...)
	if value != "" && file != "" {
		return "", errors.New("direct value and secret file cannot both be configured")
	}
	if file == "" {
		return strings.TrimSpace(value), nil
	}
	if !filepath.IsAbs(file) {
		return "", errors.New("secret file path must be absolute")
	}
	bytes, err := os.ReadFile(file)
	if err != nil {
		return "", fmt.Errorf("read secret file: %w", err)
	}
	return strings.TrimSpace(string(bytes)), nil
}

func firstTrimmedEnv(names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value
		}
	}
	return ""
}

func firstTrimmedEnvWithPresence(names ...string) (string, bool) {
	for _, name := range names {
		if value := strings.TrimSpace(os.Getenv(name)); value != "" {
			return value, true
		}
	}
	return "", false
}

func firstEnv(names ...string) string {
	for _, name := range names {
		if value := os.Getenv(name); value != "" {
			return value
		}
	}
	return ""
}

func durationEnv(name string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	if parsed, err := time.ParseDuration(value); err == nil {
		return parsed
	}
	if seconds, err := strconv.ParseUint(value, 10, 64); err == nil {
		return time.Duration(seconds) * time.Second
	}
	return fallback
}

func boolEnvironment(fallback bool, names ...string) (bool, error) {
	for _, name := range names {
		value, present := os.LookupEnv(name)
		if !present || strings.TrimSpace(value) == "" {
			continue
		}
		parsed, err := strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return false, fmt.Errorf("%s must be true or false", name)
		}
		return parsed, nil
	}
	return fallback, nil
}

func validSchemaName(value string) bool {
	if value == "" {
		return false
	}
	for index, char := range value {
		if (char >= 'a' && char <= 'z') ||
			(char >= 'A' && char <= 'Z') ||
			char == '_' ||
			(index > 0 && char >= '0' && char <= '9') {
			continue
		}
		return false
	}
	return true
}
