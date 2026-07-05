package http

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"aphrodite/internal/shared/httpx/authctx"
	"aphrodite/internal/user/domain"
	"aphrodite/internal/user/usecase"
)

type Handler struct {
	register       *usecase.RegisterUser
	authenticate   *usecase.AuthenticateUser
	getProfile     *usecase.GetUserProfile
	list           *usecase.ListUsers
	update         *usecase.UpdateUser
	changePassword *usecase.ChangePassword
}

func NewHandler(register *usecase.RegisterUser, authenticate *usecase.AuthenticateUser, getProfile *usecase.GetUserProfile, list *usecase.ListUsers, update *usecase.UpdateUser, changePassword *usecase.ChangePassword) *Handler {
	return &Handler{
		register:       register,
		authenticate:   authenticate,
		getProfile:     getProfile,
		list:           list,
		update:         update,
		changePassword: changePassword,
	}
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
// @Failure  403      {object}  ErrorResponse
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
		Username:      req.Username,
		Email:         req.Email,
		Password:      req.Password,
		PhoneNumber:   req.PhoneNumber,
		Role:          role,
		SuperAdminKey: req.SuperAdminKey,
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

// UpdateMe updates the currently authenticated user's profile.
//
// @Summary  Update current user profile
// @Tags     users
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    request  body      UpdateMeRequest  true  "Profile payload"
// @Success  200      {object}  UserResponse
// @Failure  400      {object}  ErrorResponse
// @Failure  401      {object}  ErrorResponse
// @Failure  409      {object}  ErrorResponse
// @Failure  500      {object}  ErrorResponse
// @Router   /v1/users/me [put]
func (h *Handler) UpdateMe(c *gin.Context) {
	caller, ok := authctx.From(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req UpdateMeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	u, err := h.update.Execute(c.Request.Context(), usecase.UpdateInput{
		TargetID:    caller.ID,
		CallerID:    caller.ID,
		CallerRole:  caller.Role,
		Username:    req.Username,
		Email:       req.Email,
		PhoneNumber: req.PhoneNumber,
	})
	if err != nil {
		mapUserError(c, err)
		return
	}
	c.JSON(http.StatusOK, toUserResponse(u))
}

// ChangePassword changes the currently authenticated user's password.
//
// @Summary  Change current user password
// @Tags     users
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    request  body  ChangePasswordRequest  true  "Password payload"
// @Success  204
// @Failure  400  {object}  ErrorResponse
// @Failure  401  {object}  ErrorResponse
// @Failure  500  {object}  ErrorResponse
// @Router   /v1/users/me/password [put]
func (h *Handler) ChangePassword(c *gin.Context) {
	caller, ok := authctx.From(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	if err := h.changePassword.Execute(c.Request.Context(), usecase.ChangePasswordInput{
		UserID:          caller.ID,
		CurrentPassword: req.CurrentPassword,
		NewPassword:     req.NewPassword,
	}); err != nil {
		mapUserError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

// List returns users for administrators, newest first.
//
// @Summary  List users (admin)
// @Tags     users
// @Produce  json
// @Security BearerAuth
// @Param    limit  query     int  false  "Page size (defaults to USER_DEFAULT_LIMIT, capped by USER_MAX_LIMIT)"
// @Param    page   query     int  false  "Page number (default 1)"
// @Success  200    {object}  UserListResponse
// @Failure  401    {object}  ErrorResponse
// @Failure  403    {object}  ErrorResponse
// @Failure  500    {object}  ErrorResponse
// @Router   /v1/users [get]
func (h *Handler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	page, _ := strconv.Atoi(c.Query("page"))

	result, err := h.list.Execute(c.Request.Context(), usecase.ListInput{Limit: limit, Page: page})
	if err != nil {
		mapUserError(c, err)
		return
	}

	items := make([]UserResponse, 0, len(result.Users))
	for _, u := range result.Users {
		items = append(items, toUserResponse(u))
	}
	c.JSON(http.StatusOK, UserListResponse{
		Users: items,
		Total: result.Total,
		Limit: result.Limit,
		Page:  result.Page,
	})
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

// UpdateByID updates any user's profile. Admin-only.
//
// @Summary  Update user by ID (admin)
// @Tags     users
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    id       path      string                  true  "User UUID"
// @Param    request  body      AdminUpdateUserRequest  true  "Profile payload"
// @Success  200      {object}  UserResponse
// @Failure  400      {object}  ErrorResponse
// @Failure  401      {object}  ErrorResponse
// @Failure  403      {object}  ErrorResponse
// @Failure  404      {object}  ErrorResponse
// @Failure  409      {object}  ErrorResponse
// @Failure  500      {object}  ErrorResponse
// @Router   /v1/users/{id} [put]
func (h *Handler) UpdateByID(c *gin.Context) {
	caller, ok := authctx.From(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "not authenticated")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid user id")
		return
	}

	var req AdminUpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	u, err := h.update.Execute(c.Request.Context(), usecase.UpdateInput{
		TargetID:    id,
		CallerID:    caller.ID,
		CallerRole:  caller.Role,
		Username:    req.Username,
		Email:       req.Email,
		PhoneNumber: req.PhoneNumber,
		Role:        domain.Role(req.Role),
	})
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
	case errors.Is(err, domain.ErrForbidden):
		writeError(c, http.StatusForbidden, err.Error())
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
