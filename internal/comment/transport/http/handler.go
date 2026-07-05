package http

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"aphrodite/internal/comment/domain"
	"aphrodite/internal/comment/usecase"
	"aphrodite/internal/shared/httpx/authctx"
)

type Handler struct {
	add    *usecase.AddComment
	list   *usecase.ListComments
	update *usecase.UpdateComment
	del    *usecase.DeleteComment
}

func NewHandler(add *usecase.AddComment, list *usecase.ListComments, update *usecase.UpdateComment, del *usecase.DeleteComment) *Handler {
	return &Handler{add: add, list: list, update: update, del: del}
}

// ── Handlers ──────────────────────────────────────────────────────────────────

// Add creates a comment on the post identified by :id.
//
// @Summary  Add a comment to a post
// @Tags     comments
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    id       path      string             true  "Post UUID"
// @Param    request  body      AddCommentRequest  true  "Comment payload"
// @Success  201      {object}  CommentResponse
// @Failure  400      {object}  ErrorResponse
// @Failure  401      {object}  ErrorResponse
// @Failure  500      {object}  ErrorResponse
// @Router   /v1/posts/{id}/comments [post]
func (h *Handler) Add(c *gin.Context) {
	caller, ok := authctx.From(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "not authenticated")
		return
	}
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid post id")
		return
	}

	var req AddCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	comment, err := h.add.Execute(c.Request.Context(), usecase.AddInput{
		PostID:   postID,
		AuthorID: caller.ID,
		Content:  req.Content,
	})
	if err != nil {
		mapCommentError(c, err)
		return
	}
	c.JSON(http.StatusCreated, toCommentResponse(comment))
}

// List returns comments for the post identified by :id.
//
// @Summary  List comments for a post
// @Tags     comments
// @Produce  json
// @Param    id      path      string  true   "Post UUID"
// @Param    limit   query     int     false  "Page size (defaults to COMMENT_DEFAULT_LIMIT, capped by COMMENT_MAX_LIMIT)"
// @Param    page    query     int     false  "Page number (default 1)"
// @Success  200     {object}  CommentListResponse
// @Failure  400     {object}  ErrorResponse
// @Failure  500     {object}  ErrorResponse
// @Router   /v1/posts/{id}/comments [get]
func (h *Handler) List(c *gin.Context) {
	postID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid post id")
		return
	}
	limit, _ := strconv.Atoi(c.Query("limit"))
	page, _ := strconv.Atoi(c.Query("page"))

	result, err := h.list.Execute(c.Request.Context(), usecase.ListInput{
		PostID: postID,
		Limit:  limit,
		Page:   page,
	})
	if err != nil {
		mapCommentError(c, err)
		return
	}

	items := make([]CommentResponse, 0, len(result.Comments))
	for _, cm := range result.Comments {
		items = append(items, toCommentResponse(cm))
	}
	c.JSON(http.StatusOK, CommentListResponse{
		Comments: items,
		Total:    result.Total,
		Limit:    result.Limit,
		Page:     result.Page,
	})
}

// Update replaces a comment's content. Only the author or an admin may update.
//
// @Summary  Update a comment (author or admin)
// @Tags     comments
// @Accept   json
// @Produce  json
// @Security BearerAuth
// @Param    id       path      string                true  "Comment UUID"
// @Param    request  body      UpdateCommentRequest  true  "Comment payload"
// @Success  200      {object}  CommentResponse
// @Failure  400      {object}  ErrorResponse
// @Failure  401      {object}  ErrorResponse
// @Failure  403      {object}  ErrorResponse
// @Failure  404      {object}  ErrorResponse
// @Failure  500      {object}  ErrorResponse
// @Router   /v1/comments/{id} [put]
func (h *Handler) Update(c *gin.Context) {
	caller, ok := authctx.From(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "not authenticated")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid comment id")
		return
	}

	var req UpdateCommentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}

	comment, err := h.update.Execute(c.Request.Context(), usecase.UpdateInput{
		CommentID:  id,
		CallerID:   caller.ID,
		CallerRole: caller.Role,
		Content:    req.Content,
	})
	if err != nil {
		mapCommentError(c, err)
		return
	}
	c.JSON(http.StatusOK, toCommentResponse(comment))
}

// Delete removes a comment. Only the author or an admin may delete.
//
// @Summary  Delete a comment (author or admin)
// @Tags     comments
// @Produce  json
// @Security BearerAuth
// @Param    id  path  string  true  "Comment UUID"
// @Success  204
// @Failure  400  {object}  ErrorResponse
// @Failure  401  {object}  ErrorResponse
// @Failure  403  {object}  ErrorResponse
// @Failure  404  {object}  ErrorResponse
// @Failure  500  {object}  ErrorResponse
// @Router   /v1/comments/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	caller, ok := authctx.From(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "not authenticated")
		return
	}
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		writeError(c, http.StatusBadRequest, "invalid comment id")
		return
	}

	err = h.del.Execute(c.Request.Context(), usecase.DeleteInput{
		CommentID:  id,
		CallerID:   caller.ID,
		CallerRole: caller.Role,
	})
	if err != nil {
		mapCommentError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func mapCommentError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrNotFound):
		writeError(c, http.StatusNotFound, err.Error())
	case errors.Is(err, domain.ErrForbidden):
		writeError(c, http.StatusForbidden, err.Error())
	case errors.Is(err, domain.ErrInvalidContent),
		errors.Is(err, domain.ErrInvalidPost),
		errors.Is(err, domain.ErrInvalidAuthor):
		writeError(c, http.StatusBadRequest, err.Error())
	default:
		slog.ErrorContext(c.Request.Context(), "comment: unexpected error", "err", err)
		writeError(c, http.StatusInternalServerError, "internal error")
	}
}

func writeError(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, gin.H{"error": msg})
}
