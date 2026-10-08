package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hkjang/invenqor/server/internal/agents"
	"github.com/hkjang/invenqor/server/internal/apikeys"
	"github.com/hkjang/invenqor/server/internal/auth"
	"github.com/hkjang/invenqor/server/internal/bootstrap"
	"github.com/hkjang/invenqor/server/internal/config"
	"github.com/hkjang/invenqor/server/internal/httpapi"
	"github.com/hkjang/invenqor/server/internal/ingest"
	"github.com/hkjang/invenqor/server/internal/spool"
	"github.com/hkjang/invenqor/server/internal/storage"
	"github.com/hkjang/invenqor/server/internal/updates"
	"github.com/hkjang/invenqor/server/internal/version"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	if err := run(logger); err != nil {
		logger.Error("server_stopped", "error", err)
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	processConfig, err := config.Load()
	if err != nil {
		return err
	}
	bootstrapStore, err := bootstrap.OpenWithKey(
		processConfig.StateDir,
		processConfig.MasterKeyPath,
	)
	if err != nil {
		return err
	}
	bootstrapValues, err := bootstrapStore.Load()
	if err != nil {
		return err
	}
	if processConfig.PostgresDSN == "" {
		processConfig.PostgresDSN = bootstrapValues.PostgresDSN
	}
	if bootstrapValues.SQLitePath != "" &&
		processConfig.SQLitePath == processConfig.StateDir+"/invenqor.db" {
		processConfig.SQLitePath = bootstrapValues.SQLitePath
	}
	if !bootstrapStore.Exists() {
		if err := bootstrapStore.Save(bootstrap.Values{
			PostgresDSN: processConfig.PostgresDSN,
			SQLitePath:  processConfig.SQLitePath,
		}); err != nil {
			return err
		}
	}

	rootContext, stop := signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
	)
	defer stop()

	database, err := storage.Open(rootContext, storage.Options{
		PostgresDSN: processConfig.PostgresDSN,
		SQLitePath:  processConfig.SQLitePath,
		Schema:      processConfig.DatabaseSchema,
		Timeout:     processConfig.DatabaseTimeout,
	})
	if err != nil {
		return err
	}
	defer database.Close()

	totpService := auth.NewTOTPService(database.DB(), bootstrapStore)
	authOptions := auth.DefaultServiceOptions()
	authOptions.TOTP = totpService
	authService, err := auth.NewService(database.DB(), authOptions)
	if err != nil {
		return err
	}
	bootstrapManager := auth.NewBootstrapManager(database.DB(), processConfig.StateDir)
	oidcService := auth.NewOIDCService(database.DB(), bootstrapStore, authService)
	// v0.2.14 kept the Keycloak client secret in each Pod's bootstrap.enc.
	// Reading it once at startup migrates the value into the encrypted shared
	// database setting, so every Pod can complete an OIDC flow.
	if _, err := oidcService.ClientSecretConfigured(rootContext); err != nil {
		return fmt.Errorf("initialize Keycloak client secret: %w", err)
	}
	agentService := agents.NewService(database.DB())
	ingestService := ingest.NewService(database.DB())
	eventSpool, err := spool.OpenDirectory(processConfig.EventSpoolDir)
	if err != nil {
		return err
	}
	updateStore, err := updates.Open(processConfig.UpdateDir)
	if err != nil {
		return err
	}
	// With the key configured, a bad signature is rejected at publish time
	// instead of failing silently on every agent in the fleet.
	signingKey, err := updates.ParsePublicKey(processConfig.UpdateSigningPublicKey)
	if err != nil {
		return err
	}
	if signingKey != nil {
		updateStore.SetSigningKey(signingKey)
		logger.Info("agent_update_signature_verification_enabled")
	} else {
		logger.Warn(
			"agent_update_signature_not_verified",
			"guidance", "set INVENQOR_UPDATE_PUBLIC_KEY so a mistyped update signature is rejected when it is published",
		)
	}
	apiKeyService := apikeys.NewService(database.DB())
	bootstrapStatus, err := bootstrapManager.Ensure(rootContext)
	if err != nil {
		return err
	}
	if bootstrapStatus.Required && processConfig.BootstrapAdmin != "" {
		user, createErr := bootstrapManager.CreateInitialAdminFromConfig(
			rootContext,
			auth.InitialAdminInput{
				Username:    processConfig.BootstrapAdmin,
				Password:    processConfig.BootstrapAdminPassword,
				DisplayName: processConfig.BootstrapAdmin,
			},
		)
		switch {
		case createErr == nil:
			logger.Info(
				"initial_admin_created_from_environment",
				"username", user.Username,
			)
		case errors.Is(createErr, auth.ErrBootstrapComplete):
			logger.Info("initial_admin_already_configured")
		default:
			return fmt.Errorf(
				"create initial administrator from environment: %w",
				createErr,
			)
		}
		bootstrapStatus, err = bootstrapManager.Ensure(rootContext)
		if err != nil {
			return err
		}
	}
	// Do not retain the plaintext startup secret after the one-time decision.
	processConfig.BootstrapAdminPassword = ""
	if bootstrapStatus.Required {
		logger.Warn(
			"initial_setup_required",
			"bootstrap_token_file", bootstrapStatus.TokenFile,
		)
	}

	api := httpapi.New(httpapi.Options{
		Database:                    database,
		AuthService:                 authService,
		OIDCService:                 oidcService,
		TOTPService:                 totpService,
		BootstrapManager:            bootstrapManager,
		AgentService:                agentService,
		IngestService:               ingestService,
		Spool:                       eventSpool,
		BootstrapStore:              bootstrapStore,
		UpdateStore:                 updateStore,
		APIKeyService:               apiKeyService,
		Logger:                      logger,
		CurrentPostgresDSN:          processConfig.PostgresDSN,
		PostgresEnvironmentOverride: processConfig.PostgresDSNFromEnv,
		DatabaseSchema:              processConfig.DatabaseSchema,
		DatabaseTimeout:             processConfig.DatabaseTimeout,
		AgentAutoEnrollment:         processConfig.AgentAutoEnrollment,
		AgentEnrollmentToken:        processConfig.AgentEnrollmentToken,
		ListenAddress:               processConfig.ListenAddress,
	})
	// The API keeps only a SHA-256 comparison value.
	processConfig.AgentEnrollmentToken = ""
	enrollmentMode, enrollmentErr := api.AgentEnrollmentMode(rootContext)
	if enrollmentErr != nil {
		return fmt.Errorf(
			"initialize agent enrollment policy: %w",
			enrollmentErr,
		)
	}
	switch enrollmentMode {
	case "open":
		logger.Warn(
			"agent_auto_enrollment_open",
			"mode", enrollmentMode,
			"guidance", "configure an enrollment token when port 7070 is reachable from untrusted networks",
		)
	case "token":
		logger.Info("agent_auto_enrollment_enabled", "mode", enrollmentMode)
	default:
		logger.Info("agent_auto_enrollment_disabled")
	}
	go api.RunSpoolReplay(rootContext, 5*time.Second)
	httpServer := &http.Server{
		Addr:              processConfig.ListenAddress,
		Handler:           api.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       2 * time.Minute,
	}
	serverErrors := make(chan error, 1)
	go func() {
		logger.Info(
			"server_started",
			"version", version.Version,
			"listen_address", processConfig.ListenAddress,
			"database_mode", database.Mode(),
		)
		if failure := database.PostgresFailure(); failure != nil {
			logger.Warn(
				"postgres_startup_failed",
				"code", failure.Code,
				"summary", failure.Summary,
				"host", failure.Host,
			)
		}
		serverErrors <- httpServer.ListenAndServe()
	}()

	select {
	case <-rootContext.Done():
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	}
	shutdownContext, cancel := context.WithTimeout(
		context.Background(),
		processConfig.ShutdownTimeout,
	)
	defer cancel()
	if err := httpServer.Shutdown(shutdownContext); err != nil {
		return err
	}
	logger.Info("server_shutdown_complete")
	return nil
}
