package http

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"aphrodite/internal/shared/httpx/authctx"
	"aphrodite/internal/user/domain"
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
	return RequireRoles(domain.RoleAdmin)
}

// RequireRoles permits a request when its authenticated caller has any of the
// supplied Aphrodite roles. AuthMiddleware must run before this middleware.
func RequireRoles(roles ...domain.Role) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[string(role)] = struct{}{}
	}
	return func(c *gin.Context) {
		caller, ok := authctx.From(c)
		if !ok {
			writeError(c, http.StatusForbidden, "required role not granted")
			return
		}
		if _, ok := allowed[caller.Role]; !ok {
			writeError(c, http.StatusForbidden, "required role not granted")
			return
		}
		c.Next()
	}
}

func writeError(c *gin.Context, status int, msg string) {
	c.AbortWithStatusJSON(status, gin.H{"error": msg})
}
