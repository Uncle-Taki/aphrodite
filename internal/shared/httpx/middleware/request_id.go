package middleware

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"

	"github.com/gin-gonic/gin"
)

const (
	HeaderRequestID     = "X-Request-ID"
	ContextKeyRequestID = "request_id"
)

type requestIDCtxKey struct{}

var requestIDReader io.Reader = rand.Reader

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.GetHeader(HeaderRequestID)
		if id == "" {
			id = newRequestID()
		}
		c.Set(ContextKeyRequestID, id)
		c.Request = c.Request.WithContext(context.WithValue(c.Request.Context(), requestIDCtxKey{}, id))
		c.Writer.Header().Set(HeaderRequestID, id)
		c.Next()
	}
}

func RequestIDFromContext(c *gin.Context) string {
	v, _ := c.Get(ContextKeyRequestID)
	s, _ := v.(string)
	return s
}

func RequestIDFromStdContext(ctx context.Context) string {
	v, _ := ctx.Value(requestIDCtxKey{}).(string)
	return v
}

func newRequestID() string {
	var buf [16]byte
	if _, err := io.ReadFull(requestIDReader, buf[:]); err != nil {
		// rand.Read on Linux only fails if the syscall
		// is interrupted in a way Go cannot retry(rare)
		return ""
	}
	return hex.EncodeToString(buf[:])
}
