// Package cmd provides command-line interface functionality for the CLI Proxy API server.
// It includes authentication flows for various AI service providers, service startup,
// and other command-line operations.
package cmd

import (
	"context"
	"errors"
	"os/signal"
	"syscall"
	"time"

	"github.com/router-for-me/CLIProxyAPI/v6/internal/api"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/config"
	"github.com/router-for-me/CLIProxyAPI/v6/internal/storage/postgres"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy"
	"github.com/router-for-me/CLIProxyAPI/v6/sdk/cliproxy/usage"
	log "github.com/sirupsen/logrus"
)

// StartService builds and runs the proxy service using the exported SDK.
// It creates a new proxy service instance, sets up signal handling for graceful shutdown,
// and starts the service with the provided configuration.
//
// Parameters:
//   - cfg: The application configuration
//   - configPath: The path to the configuration file
//   - localPassword: Optional password accepted for local management requests
func StartService(cfg *config.Config, configPath string, localPassword string) {
	builder := cliproxy.NewBuilder().
		WithConfig(cfg).
		WithConfigPath(configPath).
		WithLocalManagementPassword(localPassword)

	ctxSignal, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	runCtx := ctxSignal
	if localPassword != "" {
		var keepAliveCancel context.CancelFunc
		runCtx, keepAliveCancel = context.WithCancel(ctxSignal)
		builder = builder.WithServerOptions(api.WithKeepAliveEndpoint(10*time.Second, func() {
			log.Warn("keep-alive endpoint idle for 10s, shutting down")
			keepAliveCancel()
		}))
	}

	// Initialize PostgreSQL storage if configured
	var pgPlugin *postgres.Plugin
	var pgClose func()
	if cfg.PostgresStorage.Enable {
		var err error
		pgPlugin, err = postgres.InitFromConfig(
			cfg.PostgresStorage.Enable,
			cfg.PostgresStorage.DSN,
			cfg.PostgresStorage.MaxConns,
			cfg.PostgresStorage.MinConns,
			cfg.PostgresStorage.MaxConnLifetime,
			cfg.PostgresStorage.MaxConnIdleTime,
		)
		if err != nil {
			log.Errorf("failed to initialize PostgreSQL storage: %v", err)
		}
		if pgPlugin != nil {
			// Register the plugin with the global usage manager
			usage.RegisterPlugin(pgPlugin)
			log.Info("PostgreSQL storage plugin registered with usage manager")

			builder = builder.WithPostgresPlugin(pgPlugin)
			pgClose = func() {
				pgPlugin.Close()
				if pgPlugin.Pool() != nil {
					pgPlugin.Pool().Close()
				}
			}
		}
	}

	service, err := builder.Build()
	if err != nil {
		log.Errorf("failed to build proxy service: %v", err)
		if pgClose != nil {
			pgClose()
		}
		return
	}

	err = service.Run(runCtx)
	if err != nil && !errors.Is(err, context.Canceled) {
		log.Errorf("proxy service exited with error: %v", err)
	}

	// Cleanup PostgreSQL resources
	if pgClose != nil {
		pgClose()
	}
}

// WaitForCloudDeploy waits indefinitely for shutdown signals in cloud deploy mode
// when no configuration file is available.
func WaitForCloudDeploy() {
	// Clarify that we are intentionally idle for configuration and not running the API server.
	log.Info("Cloud deploy mode: No config found; standing by for configuration. API server is not started. Press Ctrl+C to exit.")

	ctxSignal, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	// Block until shutdown signal is received
	<-ctxSignal.Done()
	log.Info("Cloud deploy mode: Shutdown signal received; exiting")
}
