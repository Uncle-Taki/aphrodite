
package authctx

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const ContextKey = "auth_caller"

type Caller struct {
	ID   uuid.UUID
	Role string
}

func Set(c *gin.Context, caller Caller) {
	c.Set(ContextKey, caller)
}

func From(c *gin.Context) (Caller, bool) {
	v, ok := c.Get(ContextKey)
	if !ok {
		return Caller{}, false
	}
	caller, ok := v.(Caller)
	return caller, ok
}
