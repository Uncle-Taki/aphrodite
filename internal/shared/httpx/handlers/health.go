package handlers

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type redisPinger interface {
	Ping(c context.Context) error
}

type HealthHandler struct {
	db    *gorm.DB
	redis redisPinger
}

func NewHealthHandler(db *gorm.DB, redisClient redisPinger) *HealthHandler {
	return &HealthHandler{db: db, redis: redisClient}
}

// Check returns 200 if both Postgres and Redis are reachable, 503 otherwise.
// @Summary  Health check
// @Tags     health
// @Produce  json
// @Success  200  {object}  HealthResponse
// @Failure  503  {object}  HealthResponse
// @Router   /healthz [get]
func (h *HealthHandler) Check(c *gin.Context) {
	resp := HealthResponse{}
	healthy := true

	sqlDB, err := h.db.DB()
	if err != nil || sqlDB.PingContext(c.Request.Context()) != nil {
		resp.Postgres = "down"
		healthy = false
	} else {
		resp.Postgres = "up"
	}

	if err := h.redis.Ping(c.Request.Context()); err != nil {
		resp.Redis = "down"
		healthy = false
	} else {
		resp.Redis = "up"
	}

	status := http.StatusOK
	resp.Status = "healthy"
	if !healthy {
		status = http.StatusServiceUnavailable
		resp.Status = "unhealthy"
	}

	c.JSON(status, resp)
}
