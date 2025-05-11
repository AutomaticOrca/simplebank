package api

import (
	"context"
	"errors"
	"fmt"
	"github.com/AutomaticOrca/simplebank/mail"
	"github.com/rs/zerolog/log"
	"net/http"
	"time"

	db "github.com/AutomaticOrca/simplebank/db/sqlc"
	"github.com/AutomaticOrca/simplebank/token"
	"github.com/AutomaticOrca/simplebank/util"
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

type Server struct {
	store      db.Store
	router     *gin.Engine
	tokenMaker token.Maker
	config     util.Config
	mailer     mail.EmailSender
}

func NewServer(config util.Config, store db.Store, mailer mail.EmailSender) (*Server, error) {
	tokenMaker, err := token.NewPasetoMaker(config.TokenSymmetricKey)
	if err != nil {
		return nil, fmt.Errorf("cannot create token maker: %w", err)
	}

	server := &Server{
		config:     config,
		store:      store,
		tokenMaker: tokenMaker,
		mailer:     mailer,
	}

	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterValidation("currency", validCurrency)
	}

	server.setupRouter()
	return server, nil
}

func (server *Server) Start(ctx context.Context, address string) error {
	srv := &http.Server{
		Addr:    address,
		Handler: server.router, // Your Gin engine
	}

	serverErrors := make(chan error, 1)

	go func() {
		log.Info().Msgf("HTTP server starting, listening on %s", address)
		err := srv.ListenAndServe()
		if !errors.Is(err, http.ErrServerClosed) {
			serverErrors <- err
		} else {
			serverErrors <- nil
		}
	}()

	log.Info().Msg("API Server goroutine for ListenAndServe started. Waiting for context cancellation or server error.")

	select {
	case <-ctx.Done():
		log.Info().Msg("API Server: Shutdown signal received via context (ctx.Done()).")
		log.Info().Str("reason", ctx.Err().Error()).Msg("Context cancellation reason")

	case err := <-serverErrors:
		if err != nil {
			log.Error().Err(err).Msg("API Server: ListenAndServe error. Server will not attempt graceful shutdown via context.")
			return fmt.Errorf("http server ListenAndServe error: %w", err)
		}

		log.Info().Msg("API Server: ListenAndServe exited cleanly before context cancellation.")
		return nil
	}

	log.Info().Msg("API Server: Attempting graceful shutdown (10s timeout)...")
	shutdownCtx, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancelShutdown()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Error().Err(err).Msg("API Server: Graceful shutdown failed.")
		<-serverErrors
		return fmt.Errorf("http server graceful shutdown failed: %w", err)
	}

	log.Info().Msg("API Server: Gracefully shut down.")
	if err := <-serverErrors; err != nil {
		log.Error().Err(err).Msg("API Server: Error from ListenAndServe goroutine after shutdown.")
		return fmt.Errorf("error from ListenAndServe after shutdown: %w", err)
	}

	return nil
}

func errorResponse(err error) gin.H {
	return gin.H{"error": err.Error()}
}

func (server *Server) setupRouter() {
	router := gin.Default()

	config := cors.Config{
		AllowOrigins:     []string{"http://localhost:3000", "https://simplebank-frontend.vercel.app"},
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
	router.Use(cors.New(config))
	router.POST("/users", server.createUser)
	router.POST("/users/login", server.loginUser)
	router.GET("/users/verify_email", server.verifyEmail)
	router.POST("/tokens/renew_access", server.renewAccessToken)

	authRoutes := router.Group("/").Use(authMiddleware(server.tokenMaker))

	authRoutes.POST("/accounts", server.createAccount)
	authRoutes.GET("/accounts/:id", server.getAccount)
	authRoutes.GET("/accounts", server.listAccounts)

	authRoutes.POST("/transfers", server.createTransfer)

	server.router = router
}
