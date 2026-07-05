package http

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"aphrodite/internal/post/domain"
	"aphrodite/internal/post/usecase"
	"aphrodite/internal/shared/httpx/authctx"
)

type Handler struct {
	create *usecase.CreatePost
	get    *usecase.GetPost
	list   *usecase.ListPosts
	update *usecase.UpdatePost
	del    *usecase.DeletePost
}

func NewHandler(create *usecase.CreatePost, get *usecase.GetPost, list *usecase.ListPosts, update *usecase.UpdatePost, del *usecase.DeletePost) *Handler {
	return &Handler{create: create, get: get, list: list, update: update, del: del}
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// Create publishes a new post authored by the calling user.
//
// @Summary  Create a post
// @Tags     posts
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    request  body      CreatePostRequest  true  "Post payload"
// @Success  201      {object}  PostResponse
// @Failure  400      {object}  ErrorResponse
// @Failure  401      {object}  ErrorResponse
// @Failure  500      {object}  ErrorResponse
// @Router   /v1/posts [post]
func (h *Handler) Create(c *gin.Context) {
	caller, ok := authctx.From(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "not authenticated")
		return
	}

	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	p, err := h.create.Execute(c.Request.Context(), usecase.CreateInput{
		AuthorID: caller.ID,
		Title:    req.Title,
		Content:  req.Content,
	})
	if err != nil {
		mapPostError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toPostResponse(p))
}

// Get fetches one post by ID.
//
// @Summary  Get a post
// @Tags     posts
// @Produce  json
// @Param    id   path      string  true  "Post UUID"
// @Success  200  {object}  PostResponse
// @Failure  400  {object}  ErrorResponse
// @Failure  404  {object}  ErrorResponse
// @Failure  500  {object}  ErrorResponse
// @Router   /v1/posts/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid post id")
		return
	}
	p, err := h.get.Execute(c.Request.Context(), id)
	if err != nil {
		mapPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, toPostResponse(p))
}

// List returns recent posts, newest first.
//
// @Summary  List posts (paginated)
// @Tags     posts
// @Produce  json
// @Param    limit   query     int  false  "Page size (defaults to POST_DEFAULT_LIMIT, capped by POST_MAX_LIMIT)"
// @Param    page    query     int  false  "Page number (default 1)"
// @Success  200     {object}  PostListResponse
// @Failure  500     {object}  ErrorResponse
// @Router   /v1/posts [get]
func (h *Handler) List(c *gin.Context) {
	limit, _ := strconv.Atoi(c.Query("limit"))
	page, _ := strconv.Atoi(c.Query("page"))

	result, err := h.list.Execute(c.Request.Context(), usecase.ListInput{Limit: limit, Page: page})
	if err != nil {
		mapPostError(c, err)
		return
	}

	items := make([]PostResponse, 0, len(result.Posts))
	for _, p := range result.Posts {
		items = append(items, toPostResponse(p))
	}
	c.JSON(http.StatusOK, PostListResponse{
		Posts: items,
		Total: result.Total,
		Limit: result.Limit,
		Page:  result.Page,
	})
}

// Update replaces a post's title and content. Only the original author or an admin may update.
//
// @Summary  Update a post (author or admin)
// @Tags     posts
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    id       path      string             true  "Post UUID"
// @Param    request  body      UpdatePostRequest  true  "Post payload"
// @Success  200      {object}  PostResponse
// @Failure  400      {object}  ErrorResponse
// @Failure  401      {object}  ErrorResponse
// @Failure  403      {object}  ErrorResponse
// @Failure  404      {object}  ErrorResponse
// @Failure  500      {object}  ErrorResponse
// @Router   /v1/posts/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	caller, ok := authctx.From(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "not authenticated")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid post id")
		return
	}

	var req UpdatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	p, err := h.update.Execute(c.Request.Context(), usecase.UpdateInput{
		PostID:     id,
		CallerID:   caller.ID,
		CallerRole: caller.Role,
		Title:      req.Title,
		Content:    req.Content,
	})
	if err != nil {
		mapPostError(c, err)
		return
	}
	c.JSON(http.StatusOK, toPostResponse(p))
}

// Delete removes a post. Only the original author or an admin may delete.
//
// @Summary  Delete a post (author or admin)
// @Tags     posts
// @Produce  json
// @Security BearerAuth
// @Param    id  path  string  true  "Post UUID"
// @Success  204
// @Failure  400  {object}  ErrorResponse
// @Failure  401  {object}  ErrorResponse
// @Failure  403  {object}  ErrorResponse
// @Failure  404  {object}  ErrorResponse
// @Failure  500  {object}  ErrorResponse
// @Router   /v1/posts/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	caller, ok := authctx.From(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "not authenticated")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid post id")
		return
	}

	err = h.del.Execute(c.Request.Context(), usecase.DeleteInput{
		PostID:     id,
		CallerID:   caller.ID,
		CallerRole: caller.Role,
	})
	if err != nil {
		mapPostError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func mapPostError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		writeError(c, http.StatusForbidden, err.Error())
	case errors.Is(err, domain.ErrInvalidTitle),
		errors.Is(err, domain.ErrInvalidContent),
		errors.Is(err, domain.ErrInvalidAuthor):
		writeError(c, http.StatusBadRequest, err.Error())
	default:
		slog.ErrorContext(c.Request.Context(), "post: unexpected error", "err", err)
		writeError(c, http.StatusInternalServerError, "internal error")
	}
}

func writeError(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, gin.H{"error": msg})
}
