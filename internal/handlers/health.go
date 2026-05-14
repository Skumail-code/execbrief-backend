package handlers

import (
	"net/http"
	"time"

	"github.com/execbrief/backend/internal/models"
	"github.com/gin-gonic/gin"
)

func Health(c *gin.Context) {
	c.JSON(http.StatusOK, models.HealthResponse{
		Status:    "ok",
		Service:   "execbrief-api",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}
