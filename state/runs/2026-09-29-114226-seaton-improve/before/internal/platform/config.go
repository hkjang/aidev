package platform

import (
	"errors"
	"os"
	"strings"
)

// Config intentionally exposes only the three deployment-time settings SeatOn
// needs. Every other operational setting lives in the administrator UI/DB.
type Config struct {
	PostgresDSN            string
	BootstrapAdmin         string
	BootstrapAdminPassword string
}

func LoadConfig() (Config, error) {
	c := Config{
		PostgresDSN:            strings.TrimSpace(os.Getenv("POSTGRES_DSN")),
		BootstrapAdmin:         strings.TrimSpace(os.Getenv("BOOTSTRAP_ADMIN")),
		BootstrapAdminPassword: os.Getenv("BOOTSTRAP_ADMIN_PASSWORD"),
	}
	var missing []string
	if c.PostgresDSN == "" {
		missing = append(missing, "POSTGRES_DSN")
	}
	if c.BootstrapAdmin == "" {
		missing = append(missing, "BOOTSTRAP_ADMIN")
	}
	if c.BootstrapAdminPassword == "" {
		missing = append(missing, "BOOTSTRAP_ADMIN_PASSWORD")
	}
	if len(missing) > 0 {
		return Config{}, errors.New("missing required environment variables: " + strings.Join(missing, ", "))
	}
	if len(c.BootstrapAdminPassword) < 12 {
		return Config{}, errors.New("BOOTSTRAP_ADMIN_PASSWORD must be at least 12 characters")
	}
	return c, nil
}
