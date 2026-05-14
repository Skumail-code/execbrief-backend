package handlers

import (
	"net/http"
	"strings"

	"github.com/execbrief/backend/internal/middleware"
	"github.com/execbrief/backend/internal/models"
	"github.com/execbrief/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type ActionItemHandler struct {
	service *services.MeetingService
}

func NewActionItemHandler(service *services.MeetingService) *ActionItemHandler {
	return &ActionItemHandler{service: service}
}

func (h *ActionItemHandler) Update(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "authentication required"})
		return
	}

	id, err := parseObjectID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid action item id"})
		return
	}

	var req models.UpdateActionItemRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	updated, err := h.service.UpdateActionItem(c.Request.Context(), user.ID, id, &req)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "action item not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to update action item"})
		return
	}
	if updated == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "action item not found"})
		return
	}

	c.JSON(http.StatusOK, updated)
}
