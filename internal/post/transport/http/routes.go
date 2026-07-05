package http

import "github.com/gin-gonic/gin"

func Register(rg *gin.RouterGroup, h *Handler, authMiddleware gin.HandlerFunc) {
	posts := rg.Group("/posts")
	posts.GET("", h.List)
	posts.GET("/:id", h.Get)

	auth := posts.Group("")
	auth.Use(authMiddleware)
	auth.POST("", h.Create)
	auth.PUT("/:id", h.Update)
	auth.DELETE("/:id", h.Delete)
}
