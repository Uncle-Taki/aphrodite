package http

import (
	"github.com/gin-gonic/gin"

	"aphrodite/internal/user/usecase"
)

func Register(rg *gin.RouterGroup, h *Handler, tokens usecase.TokenIssuer) {
	users := rg.Group("/users")
	users.POST("/register", h.Register)
	users.POST("/login", h.Authenticate)

	auth := users.Group("")
	auth.Use(AuthMiddleware(tokens))
	auth.GET("/me", h.Me)

	admin := users.Group("")
	admin.Use(AuthMiddleware(tokens), RequireAdmin())
	admin.GET("/:id", h.GetByID)
}
