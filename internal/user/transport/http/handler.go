package http

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"aphrodite/internal/shared/httpx/authctx"
	"aphrodite/internal/user/domain"
	"aphrodite/internal/user/usecase"
)

type Handler struct {
	register     *usecase.RegisterUser
	authenticate *usecase.AuthenticateUser
	getProfile   *usecase.GetUserProfile
}

func NewHandler(register *usecase.RegisterUser, authenticate *usecase.AuthenticateUser, getProfile *usecase.GetUserProfile) *Handler {
	return &Handler{register: register, authenticate: authenticate, getProfile: getProfile}
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// Register creates a new user account.
//
// @Summary  Register a new user
// @Tags     users
// @Accept   json
// @Produce  json
// @Param    request  body      RegisterRequest  true  "Registration payload"
// @Success  201      {object}  UserResponse
// @Failure  400      {object}  ErrorResponse
// @Failure  409      {object}  ErrorResponse
// @Failure  500      {object}  ErrorResponse
// @Router   /v1/users/register [post]
func (h *Handler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	role := domain.Role(req.Role)
	if role == "" {
		role = domain.RoleUser
	}

	u, err := h.register.Execute(c.Request.Context(), usecase.RegisterInput{
		Username:    req.Username,
		Email:       req.Email,
		Password:    req.Password,
		PhoneNumber: req.PhoneNumber,
		Role:        role,
	})
	if err != nil {
		mapUserError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toUserResponse(u))
}

// Authenticate verifies credentials and returns a bearer token.
//
// @Summary  Authenticate (login)
// @Tags     users
// @Accept   json
// @Produce  json
// @Param    request  body      AuthenticateRequest  true  "Login payload — identifier accepts username or email"
// @Success  200      {object}  AuthenticateResponse
// @Failure  400      {object}  ErrorResponse
// @Failure  401      {object}  ErrorResponse
// @Failure  500      {object}  ErrorResponse
// @Router   /v1/users/login [post]
func (h *Handler) Authenticate(c *gin.Context) {
	var req AuthenticateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	result, err := h.authenticate.Execute(c.Request.Context(), usecase.AuthenticateInput{
		Identifier: req.Identifier,
		Password:   req.Password,
	})
	if err != nil {
		mapUserError(c, err)
		return
	}
	c.JSON(http.StatusOK, AuthenticateResponse{Token: result.Token, User: toUserResponse(result.User)})
}

// Me returns the profile of the currently authenticated caller.
//
// @Summary  Get current user profile
// @Tags     users
// @Produce  json
// @Security BearerAuth
// @Success  200  {object}  UserResponse
// @Failure  401  {object}  ErrorResponse
// @Failure  404  {object}  ErrorResponse
// @Failure  500  {object}  ErrorResponse
// @Router   /v1/users/me [get]
func (h *Handler) Me(c *gin.Context) {
	caller, ok := authctx.From(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "not authenticated")
		return
	}
	u, err := h.getProfile.Execute(c.Request.Context(), caller.ID)
	if err != nil {
		mapUserError(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserResponse(u))
}

// GetByID fetches any user's profile. Admin-only.
//
// @Summary  Get user by ID (admin)
// @Tags     users
// @Produce  json
// @Security BearerAuth
// @Param    id   path      string  true  "User UUID"
// @Success  200  {object}  UserResponse
// @Failure  400  {object}  ErrorResponse
// @Failure  401  {object}  ErrorResponse
// @Failure  403  {object}  ErrorResponse
// @Failure  404  {object}  ErrorResponse
// @Failure  500  {object}  ErrorResponse
// @Router   /v1/users/{id} [get]
func (h *Handler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid user id")
		return
	}
	u, err := h.getProfile.Execute(c.Request.Context(), id)
	if err != nil {
		mapUserError(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserResponse(u))
}

func mapUserError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrAlreadyExists):
		writeError(c, http.StatusConflict, err.Error())
	case errors.Is(err, domain.ErrInvalidCredential):
		writeError(c, http.StatusUnauthorized, err.Error())
	case errors.Is(err, domain.ErrInvalidUsername),
		errors.Is(err, domain.ErrInvalidEmail),
		errors.Is(err, domain.ErrInvalidPasswordHash),
		errors.Is(err, domain.ErrInvalidRole):
		writeError(c, http.StatusBadRequest, err.Error())
	default:
		slog.ErrorContext(c.Request.Context(), "user: unexpected error", "err", err)
		writeError(c, http.StatusInternalServerError, "internal error")
	}
}
