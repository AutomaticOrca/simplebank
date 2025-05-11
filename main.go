package main

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"github.com/AutomaticOrca/simplebank/mail"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/AutomaticOrca/simplebank/api"
	db "github.com/AutomaticOrca/simplebank/db/sqlc"
	"github.com/AutomaticOrca/simplebank/util"
	"github.com/hibiken/asynq"
	_ "github.com/lib/pq"
	"golang.org/x/sync/errgroup"
)

var interruptSignals = []os.Signal{
	os.Interrupt,
	syscall.SIGTERM,
	syscall.SIGINT,
}

func main() {
	config, err := util.LoadConfig()
	if err != nil {
		log.Fatal().Err(err).Msg("cannot load config:")
	}

	log.Info().
		Str("db_driver", config.DBDriver).
		Str("db_source_is_empty", fmt.Sprintf("%t", config.DBSource == "")).
		Str("http_server_address", config.HTTPServerAddress).
		Str("redis_address", config.RedisAddress).
		Str("redis_password_is_empty", fmt.Sprintf("%t", config.RedisPassword == "")).
		Msg("Loaded configuration values from environment")

	if config.DBDriver == "" {
		log.Fatal().Msg("DB_DRIVER configuration is empty, cannot proceed")
	}
	if config.RedisAddress == "" {
		log.Fatal().Msg("REDIS_ADDRESS configuration is empty, cannot proceed")
	}

	ctx, stop := signal.NotifyContext(context.Background(), interruptSignals...)
	defer stop()

	conn, err := sql.Open(config.DBDriver, config.DBSource)
	if err != nil {
		log.Fatal().Err(err).Msg("cannot connect to db")
	}
	defer func() {
		log.Info().Msg("Closing database connection...")
		if errDbClose := conn.Close(); errDbClose != nil {
			log.Error().Err(errDbClose).Msg("failed to close db connection")
		}
	}()

	if err = conn.PingContext(ctx); err != nil {
		log.Fatal().Err(err).Msg("cannot ping db")
	}
	store := db.NewStore(conn)
	log.Info().Msg("Database connection established")

	mailer := mail.NewGmailSender(config.EmailSenderName, config.EmailSenderAddress, config.EmailSenderPassword)
	log.Info().Msg("Mailer initialized.")

	waitGroup, gCtx := errgroup.WithContext(ctx)

	// Run Gin HTTP API Server
	runGinAPIServerInGroup(gCtx, waitGroup, config, store, mailer)

	log.Info().Msg("All components scheduled to run. Waiting for interrupt signal or component error...")
	err = waitGroup.Wait() // Block until all goroutines in the group complete

	if err != nil && !errors.Is(err, context.Canceled) && !errors.Is(err, http.ErrServerClosed) && !errors.Is(err, asynq.ErrServerClosed) {
		log.Error().Err(err).Msg("Application exited due to an error from a component.")
		// os.Exit(1) // Consider exiting with non-zero code for critical errors
	} else if errors.Is(err, context.Canceled) {
		log.Info().Msg("Application shutting down due to context cancellation (e.g., interrupt signal).")
	} else {
		log.Info().Msg("Application exited gracefully.")
	}
}

func runGinAPIServerInGroup(
	gCtx context.Context,
	waitGroup *errgroup.Group,
	config util.Config,
	store db.Store,
	mailer mail.EmailSender,
) {
	apiServer, err := api.NewServer(config, store, mailer)
	if err != nil {
		waitGroup.Go(func() error {
			return fmt.Errorf("cannot create API server: %w", err)
		})
		log.Error().Err(err).Msg("Failed to create API server instance.")
		return
	}

	waitGroup.Go(func() error {
		log.Info().Msgf("Gin API server starting at http://%s", config.HTTPServerAddress)
		startErr := apiServer.Start(gCtx, config.HTTPServerAddress)
		if startErr != nil && !errors.Is(startErr, http.ErrServerClosed) {
			log.Error().Err(startErr).Msg("Gin API server failed or crashed")
			return startErr
		}
		log.Info().Msg("Gin API server has stopped.")
		return nil
	})
}
