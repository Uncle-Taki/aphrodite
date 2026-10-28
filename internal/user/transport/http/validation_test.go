package http

import (
	"aphrodite/internal/shared/httpx/authctx"
	"aphrodite/internal/user/domain"
	"aphrodite/internal/user/usecase"
	"context"
	"errors"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestUserHandler_RejectsMissingCallerAndMalformedRequests(t *testing.T) {
	h := &Handler{}
	for _, tc := range []struct {
		name     string
		handler  gin.HandlerFunc
		caller   bool
		id, body string
		want     int
	}{
		{"me without caller", h.Me, false, "", "", 401},
		{"update me without caller", h.UpdateMe, false, "", "{}", 401},
		{"password without caller", h.ChangePassword, false, "", "{}", 401},
		{"admin update without caller", h.UpdateByID, false, "", "{}", 401},
		{"register malformed JSON", h.Register, false, "", "{", 400},
		{"login malformed JSON", h.Authenticate, false, "", "{", 400},
		{"update malformed JSON", h.UpdateMe, true, "", "{", 400},
		{"password malformed JSON", h.ChangePassword, true, "", "{", 400},
		{"admin malformed JSON", h.UpdateByID, true, uuid.NewString(), "{", 400},
		{"admin malformed ID", h.UpdateByID, true, "invalid", "{}", 400},
	} {
		t.Run(tc.name, func(t *testing.T) {
			res := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(res)
			c.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tc.body))
			c.Params = gin.Params{{Key: "id", Value: tc.id}}
			if tc.caller {
				authctx.Set(c, authctx.Caller{ID: uuid.New(), Role: "admin"})
			}
			tc.handler(c)
			if res.Code != tc.want {
				t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
			}
		})
	}
	res := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(res)
	RequireRoles(domain.RoleAdmin)(c)
	if res.Code != 403 {
		t.Fatalf("role gate without caller: %d", res.Code)
	}
}

type listFailureRepo struct{ *memoryUserRepo }

func (r listFailureRepo) List(context.Context, usecase.ListFilter) ([]*domain.User, int64, error) {
	return nil, 0, errors.New("private database detail")
}

func TestUserHandler_MapsValidationAndDependencyFailures(t *testing.T) {
	repo := newMemoryUserRepo(testUser(uuid.MustParse(callerID), domain.RoleUser))
	router := newUserRouter(repo)
	for _, tc := range []struct {
		method, path, body string
		status             int
	}{
		{"POST", "/v1/users/register", `{"username":"bob","email":"bob@example.com","password":"secret","role":"admin"}`, 403},
		{"POST", "/v1/users/register", `{"username":"bob","email":"invalid","password":"secret"}`, 400},
		{"PUT", "/v1/users/me", `{"username":"alice","email":"invalid"}`, 400},
		{"PUT", "/v1/users/" + uuid.NewString(), `{"username":"alice","email":"alice@example.com"}`, 404},
	} {
		t.Run(tc.path+tc.body, func(t *testing.T) {
			req := adminRequest(tc.method, tc.path, tc.body)
			if tc.path == "/v1/users/me" {
				req = authedRequest(tc.method, tc.path, tc.body)
			}
			res := httptest.NewRecorder()
			router.ServeHTTP(res, req)
			if res.Code != tc.status {
				t.Fatalf("status=%d body=%s", res.Code, res.Body.String())
			}
		})
	}
	h := &Handler{list: usecase.NewListUsers(listFailureRepo{repo}, usecase.ListConfig{DefaultLimit: 10, MaxLimit: 50})}
	res := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(res)
	c.Request = httptest.NewRequest("GET", "/", nil)
	h.List(c)
	if res.Code != 500 || strings.Contains(res.Body.String(), "private database detail") {
		t.Fatalf("dependency error leaked: %d %s", res.Code, res.Body.String())
	}
}
