package http

import (
	"aphrodite/internal/shared/httpx/authctx"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandler_RejectsMissingCallerAndMalformedRequests(t *testing.T) {
	h := &Handler{}
	for _, tc := range []struct {
		name     string
		handler  gin.HandlerFunc
		caller   bool
		id, body string
		want     int
	}{
		{"create without caller", h.Add, false, "valid", "{}", 401},
		{"update without caller", h.Update, false, "valid", "{}", 401},
		{"delete without caller", h.Delete, false, "valid", "", 401},
		{"create malformed JSON", h.Add, true, uuid.NewString(), "{", 400},
		{"update malformed JSON", h.Update, true, uuid.NewString(), "{", 400},
		{"update malformed ID", h.Update, true, "invalid", "{}", 400},
		{"delete malformed ID", h.Delete, true, "invalid", "", 400},
		{"create malformed post ID", h.Add, true, "invalid", "{}", 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(res)
			c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			c.Params = gin.Params{{Key: "id", Value: tc.id}}
			if tc.caller {
				authctx.Set(c, authctx.Caller{ID: uuid.New(), Role: "user"})
			}
			tc.handler(c)
			if res.Code != tc.want {
				t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
			}
		})
	}
}
