package handlers

import (
	"net/http"

	"github.com/execbrief/backend/internal/middleware"
	"github.com/execbrief/backend/internal/models"
	"github.com/execbrief/backend/internal/services"
	"github.com/gin-gonic/gin"
)

type AuthHandler struct {
	service *services.AuthService
}

func NewAuthHandler(service *services.AuthService) *AuthHandler {
	return &AuthHandler{service: service}
}

type authEnvelope struct {
	User models.User `json:"user"`
}

func (h *AuthHandler) Signup(c *gin.Context) {
	var req models.SignupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	user, token, err := h.service.Signup(c.Request.Context(), req.Name, req.Email, req.Password)
	if err != nil {
		if err.Error() == "an account with that email already exists" {
			c.JSON(http.StatusConflict, models.ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	middleware.SetAuthCookie(c, token)
	c.JSON(http.StatusCreated, authEnvelope{User: *user})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	user, token, err := h.service.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "invalid email or password"})
		return
	}

	middleware.SetAuthCookie(c, token)
	c.JSON(http.StatusOK, authEnvelope{User: *user})
}

func (h *AuthHandler) Me(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "authentication required"})
		return
	}
	c.JSON(http.StatusOK, authEnvelope{User: *user})
}

func (h *AuthHandler) UpdateMe(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "authentication required"})
		return
	}

	var req models.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	name := user.Name
	email := user.Email
	if req.Name != nil {
		name = *req.Name
	}
	if req.Email != nil {
		email = *req.Email
	}

	updated, err := h.service.UpdateProfile(c.Request.Context(), user.ID, name, email)
	if err != nil {
		if err.Error() == "an account with that email already exists" {
			c.JSON(http.StatusConflict, models.ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, authEnvelope{User: *updated})
}

func (h *AuthHandler) UpdatePlan(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "authentication required"})
		return
	}

	var req models.UpdatePlanRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid request body"})
		return
	}

	updated, err := h.service.UpgradePlan(c.Request.Context(), user.ID, req.Plan)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
		return
	}

	c.JSON(http.StatusOK, authEnvelope{User: *updated})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	if token, err := c.Cookie("execbrief_session"); err == nil && token != "" {
		_ = h.service.Logout(c.Request.Context(), token)
	}
	middleware.ClearAuthCookie(c)
	c.JSON(http.StatusOK, models.SuccessResponse{Message: "signed out"})
}
