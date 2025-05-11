package api

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"time"

	db "github.com/AutomaticOrca/simplebank/db/sqlc"
	"github.com/AutomaticOrca/simplebank/util"
	"github.com/AutomaticOrca/simplebank/val"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/rs/zerolog/log"
)

type createUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,min=6"`
	FullName string `json:"full_name" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
}

type userResponse struct {
	Username          string    `json:"username"`
	FullName          string    `json:"full_name"`
	Email             string    `json:"email"`
	PasswordChangedAt time.Time `json:"password_changed_at"`
	CreatedAt         time.Time `json:"created_at"`
}

func newUserResponse(user db.User) userResponse {
	return userResponse{
		Username:          user.Username,
		FullName:          user.FullName,
		Email:             user.Email,
		PasswordChangedAt: user.PasswordChangedAt,
		CreatedAt:         user.CreatedAt,
	}
}

func (server *Server) sendVerificationEmailAsync(createdUser db.User) {
	goroutineCtx := context.Background()

	log.Info().Str("username", createdUser.Username).Msg("[Goroutine] Initiating verification email process...")

	secretCode := util.RandomString(32)
	verifyEmailEntry, err := server.store.CreateVerifyEmail(goroutineCtx, db.CreateVerifyEmailParams{
		Username:   createdUser.Username,
		Email:      createdUser.Email,
		SecretCode: secretCode,
	})
	if err != nil {
		log.Error().Err(err).Str("username", createdUser.Username).Msg("[Goroutine] Failed to create verify_email record in DB.")
		return
	}
	log.Info().Str("username", createdUser.Username).Int64("verify_email_id", verifyEmailEntry.ID).Msg("[Goroutine] verify_email record created successfully.")

	subject := "Welcome to Simple Bank"
	verifyUrl := fmt.Sprintf("%s/users/verify_email?email_id=%d&secret_code=%s",
		server.config.FrontendBaseURL,
		verifyEmailEntry.ID,
		verifyEmailEntry.SecretCode)

	content := fmt.Sprintf(`Hello %s,<br/>
Thank you for registering with us!<br/>
Please <a href="%s">click here</a> to verify your email address.<br/>
`, createdUser.FullName, verifyUrl)
	to := []string{createdUser.Email}

	err = server.mailer.SendEmail(subject, content, to, nil, nil, nil) // Accessing via server struct
	if err != nil {
		log.Error().Err(err).Str("username", createdUser.Username).Msg("[Goroutine] Failed to send verification email.")
		return
	}
	log.Info().Str("username", createdUser.Username).Str("email", createdUser.Email).Msg("[Goroutine] Verification email sent successfully.")
}

func (server *Server) createUser(ctx *gin.Context) {
	var req createUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	if err := val.ValidateUsername(req.Username); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	if err := val.ValidatePassword(req.Password); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	if err := val.ValidateFullName(req.FullName); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}
	if err := val.ValidateEmail(req.Email); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	hashedPassword, err := util.HashPassword(req.Password)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	arg := db.CreateUserTxParams{
		CreateUserParams: db.CreateUserParams{
			Username:       req.Username,
			HashedPassword: hashedPassword,
			FullName:       req.FullName,
			Email:          req.Email,
		},
		AfterCreate: func(user db.User) error {
			log.Info().Str("username", user.Username).Msg("User created successfully. Scheduling async email verification.")
			go server.sendVerificationEmailAsync(user)
			return nil
		},
	}

	txResult, err := server.store.CreateUserTx(ctx, arg)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			switch pqErr.Code.Name() {
			case "unique_violation":
				ctx.JSON(http.StatusConflict, errorResponse(fmt.Errorf("username or email already exists")))
				return
			}
		}

		log.Error().Err(err).Msg("failed to execute create user transaction")
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := newUserResponse(txResult.User)
	ctx.JSON(http.StatusCreated, rsp)
}

type loginUserRequest struct {
	Username string `json:"username" binding:"required,alphanum"`
	Password string `json:"password" binding:"required,min=6"`
}

type loginUserResponse struct {
	SessionID             uuid.UUID    `json:"session_id"`
	AccessToken           string       `json:"access_token"`
	AccessTokenExpiresAt  time.Time    `json:"access_token_expires_at"`
	RefreshToken          string       `json:"refresh_token"`
	RefreshTokenExpiresAt time.Time    `json:"refresh_token_expires_at"`
	User                  userResponse `json:"user"`
}

func (server *Server) loginUser(ctx *gin.Context) {
	var req loginUserRequest
	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, errorResponse(err))
		return
	}

	user, err := server.store.GetUser(ctx, req.Username)
	if err != nil {
		if err == sql.ErrNoRows {
			ctx.JSON(http.StatusNotFound, errorResponse(err))
			return
		}
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	err = util.CheckPassword(req.Password, user.HashedPassword)
	if err != nil {
		ctx.JSON(http.StatusUnauthorized, errorResponse(err))
		return
	}

	accessToken, accessPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		server.config.AccessTokenDuration,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	refreshToken, refreshPayload, err := server.tokenMaker.CreateToken(
		user.Username,
		server.config.RefreshTokenDuration,
	)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	session, err := server.store.CreateSession(ctx, db.CreateSessionParams{
		ID:           refreshPayload.ID,
		Username:     user.Username,
		RefreshToken: refreshToken,
		UserAgent:    ctx.Request.UserAgent(),
		ClientIp:     ctx.ClientIP(),
		IsBlocked:    false,
		ExpiresAt:    refreshPayload.ExpiredAt,
	})
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, errorResponse(err))
		return
	}

	rsp := loginUserResponse{
		SessionID:             session.ID,
		AccessToken:           accessToken,
		AccessTokenExpiresAt:  accessPayload.ExpiredAt,
		RefreshToken:          refreshToken,
		RefreshTokenExpiresAt: refreshPayload.ExpiredAt,
		User:                  newUserResponse(user),
	}
	ctx.JSON(http.StatusOK, rsp)
}
