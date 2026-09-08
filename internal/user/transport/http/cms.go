package http

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"aphrodite/internal/shared/httpx/authctx"
	"aphrodite/internal/user/domain"
	"aphrodite/internal/user/usecase"
)

const cmsSessionKey = "cms_session"

type CMSLoginRequest struct {
	Identifier string `json:"identifier" binding:"required"`
	Password   string `json:"password" binding:"required"`
}

type CMSHandler struct {
	sessions   *usecase.CMSSessionManager
	cookieName string
	cookieTTL  time.Duration
	secure     bool
}

func NewCMSHandler(sessions *usecase.CMSSessionManager, cookieName string, ttl time.Duration, secure bool) *CMSHandler {
	if cookieName == "" {
		cookieName = "aphrodite_cms_session"
	}
	if ttl <= 0 {
		ttl = 12 * time.Hour
	}
	return &CMSHandler{sessions: sessions, cookieName: cookieName, cookieTTL: ttl, secure: secure}
}

func RegisterCMS(rg *gin.RouterGroup, h *CMSHandler) {
	cms := rg.Group("/cms")
	cms.POST("/auth/login", h.Login)
	auth := cms.Group("")
	auth.Use(CMSAuthMiddlewareSecure(h.sessions, h.cookieName, h.secure))
	auth.POST("/auth/logout", h.Logout)
	auth.GET("/me", h.Me)
}

func CMSAuthMiddleware(sessions *usecase.CMSSessionManager, cookieName string) gin.HandlerFunc {
	return CMSAuthMiddlewareSecure(sessions, cookieName, false)
}

func CMSAuthMiddlewareSecure(sessions *usecase.CMSSessionManager, cookieName string, secureCookie bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(cookieName)
		if err != nil || token == "" {
			writeError(c, http.StatusUnauthorized, "cms session required")
			return
		}
		session, user, err := sessions.Authenticate(c.Request.Context(), token)
		if err != nil {
			c.SetCookie(cookieName, "", -1, "/", "", secureCookie, true)
			writeError(c, http.StatusUnauthorized, "invalid or expired cms session")
			return
		}
		c.Set(cmsSessionKey, session)
		authctx.Set(c, authctx.Caller{ID: user.ID, Role: string(user.Role)})
		c.Next()
	}
}

func RequireCMSRole(roles ...string) gin.HandlerFunc {
	allowed := make(map[string]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(c *gin.Context) {
		caller, ok := authctx.From(c)
		if !ok {
			writeError(c, http.StatusUnauthorized, "cms session required")
			return
		}
		if _, ok := allowed[caller.Role]; !ok {
			writeError(c, http.StatusForbidden, "cms permission required")
			return
		}
		c.Next()
	}
}

func (h *CMSHandler) Login(c *gin.Context) {
	var req CMSLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		writeError(c, http.StatusBadRequest, err.Error())
		return
	}
	result, err := h.sessions.Login(c.Request.Context(), usecase.CMSLoginInput{Identifier: req.Identifier, Password: req.Password})
	if err != nil {
		mapUserError(c, err)
		return
	}
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(h.cookieName, result.Token, int(h.cookieTTL/time.Second), "/", "", h.secure, true)
	c.JSON(http.StatusOK, gin.H{"user": toUserResponse(result.User), "expires_at": result.Session.ExpiresAt})
}

func (h *CMSHandler) Logout(c *gin.Context) {
	value, _ := c.Get(cmsSessionKey)
	if session, ok := value.(*domain.Session); ok {
		if err := h.sessions.Logout(c.Request.Context(), session); err != nil {
			writeError(c, http.StatusInternalServerError, "internal error")
			return
		}
	}
	c.SetCookie(h.cookieName, "", -1, "/", "", h.secure, true)
	c.Status(http.StatusNoContent)
}

func (h *CMSHandler) Me(c *gin.Context) {
	caller, ok := authctx.From(c)
	if !ok {
		writeError(c, http.StatusUnauthorized, "cms session required")
		return
	}
	c.JSON(http.StatusOK, gin.H{"user_id": caller.ID, "role": caller.Role})
}
