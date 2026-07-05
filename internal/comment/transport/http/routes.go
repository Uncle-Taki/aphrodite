package http

import "github.com/gin-gonic/gin"

func Register(rg *gin.RouterGroup, h *Handler, authMiddleware gin.HandlerFunc) {
	nested := rg.Group("/posts/:id/comments")
	nested.GET("", h.List)

	nestedAuth := nested.Group("")
	nestedAuth.Use(authMiddleware)
	nestedAuth.POST("", h.Add)

	comments := rg.Group("/comments")
	comments.Use(authMiddleware)
	comments.PUT("/:id", h.Update)
	comments.DELETE("/:id", h.Delete)
}
