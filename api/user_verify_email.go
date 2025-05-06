package api

import (
	"fmt"
	"net/http"

	db "github.com/AutomaticOrca/simplebank/db/sqlc"
	"github.com/AutomaticOrca/simplebank/val"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

type verifyEmailRequest struct {
	EmailID    int64  `form:"email_id" binding:"required,min=1"`
	SecretCode string `form:"secret_code" binding:"required"`
}

func (server *Server) verifyEmail(ctx *gin.Context) {
	var req verifyEmailRequest
	if err := ctx.ShouldBindQuery(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	if err := val.ValidateEmailId(req.EmailID); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("email_id: %w", err)))
		return
	}
	if err := val.ValidateSecretCode(req.SecretCode); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(fmt.Errorf("secret_code: %w", err)))
		return
	}

	txArg := db.VerifyEmailTxParams{
		EmailId:    req.EmailID,
		SecretCode: req.SecretCode,
	}

	txResult, err := server.store.VerifyEmailTx(ctx, txArg)
	if err != nil {
		log.Error().Err(err).Int64("email_id", req.EmailID).Msg("failed to execute verify email transaction")
		ctx.JSON(http.StatusInternalServerError, errorResponse(fmt.Errorf("failed to verify email: %w", err)))
		return
	}

	rsp := gin.H{
		"message":           "Email verified successfully!",
		"username":          txResult.User.Username,
		"is_email_verified": txResult.User.IsEmailVerified,
	}
	log.Info().Str("username", txResult.User.Username).Msg("email verified successfully via HTTP")
	ctx.JSON(http.StatusOK, rsp)
}
