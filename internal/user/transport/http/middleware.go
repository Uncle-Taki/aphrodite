package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"aphrodite/internal/shared/httpx/authctx"
	"aphrodite/internal/user/usecase"
)

func AuthMiddleware(tokens usecase.TokenIssuer) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := c.GetHeader("Authorization")
		token := strings.TrimPrefix(raw, "Bearer ")
		if raw == "" || token == raw {
			writeError(c, http.StatusUnauthorized, "missing bearer token")
			return
		}

		id, role, err := tokens.Verify(c.Request.Context(), token)
		if err != nil {
			writeError(c, http.StatusUnauthorized, "invalid or expired token")
			return
		}

		authctx.Set(c, authctx.Caller{ID: id, Role: string(role)})
		c.Next()
	}
}

func RequireAdmin() gin.HandlerFunc {
	return func(c *gin.Context) {
		caller, ok := authctx.From(c)
		if !ok || caller.Role != "admin" {
			writeError(c, http.StatusForbidden, "admin role required")
			return
		}
		c.Next()
	}
}

func writeError(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, gin.H{"error": msg})
}
