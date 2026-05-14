package handlers

import (
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/execbrief/backend/internal/middleware"
	"github.com/execbrief/backend/internal/models"
	"github.com/execbrief/backend/internal/services"
	"github.com/gin-gonic/gin"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MeetingHandler struct {
	service     *services.MeetingService
	maxUploadMB int64
}

func NewMeetingHandler(service *services.MeetingService, maxUploadMB int64) *MeetingHandler {
	return &MeetingHandler{service: service, maxUploadMB: maxUploadMB}
}

var allowedExtensions = map[string]bool{
	".txt": true,
	".vtt": true,
	".srt": true,
}

func (h *MeetingHandler) Upload(c *gin.Context) {
	if err := c.Request.ParseMultipartForm(h.maxUploadMB << 20); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "file too large or invalid form data"})
		return
	}

	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "missing file field in form"})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedExtensions[ext] {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: fmt.Sprintf("unsupported file type %q; allowed: .txt, .vtt, .srt", ext),
		})
		return
	}

	content, err := io.ReadAll(file)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to read file"})
		return
	}

	// Validate that the uploaded file contains actual text, not binary data
	contentStr := string(content)
	if !isLikelyTextContent(contentStr) {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: "The uploaded file does not appear to be a text transcript. Please upload a plain text file (.txt, .vtt, or .srt) containing meeting transcript data.",
		})
		return
	}

	title := c.PostForm("title")
	if title == "" {
		title = strings.TrimSuffix(header.Filename, ext)
	}

	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "authentication required"})
		return
	}

	meeting, err := h.service.CreateMeeting(c.Request.Context(), user.ID, title, header.Filename, string(content))
	if err != nil {
		if strings.Contains(err.Error(), "not enough credits") {
			c.JSON(http.StatusPaymentRequired, models.ErrorResponse{Error: err.Error()})
			return
		}
		if strings.Contains(err.Error(), "empty or too short") {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to save meeting"})
		return
	}

	c.JSON(http.StatusCreated, models.UploadResponse{
		MeetingID: meeting.ID.Hex(),
		Title:     meeting.Title,
		Status:    string(meeting.Status),
	})
}

func (h *MeetingHandler) List(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "authentication required"})
		return
	}

	meetings, err := h.service.ListMeetings(c.Request.Context(), user.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to fetch meetings"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"meetings": meetings, "total": len(meetings)})
}

func (h *MeetingHandler) GetDetail(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "authentication required"})
		return
	}

	id, err := parseObjectID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid meeting id"})
		return
	}

	detail, err := h.service.GetMeetingDetail(c.Request.Context(), user.ID, id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to fetch meeting"})
		return
	}
	if detail == nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "meeting not found"})
		return
	}

	c.JSON(http.StatusOK, detail)
}

func (h *MeetingHandler) Analyze(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "authentication required"})
		return
	}

	id, err := parseObjectID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid meeting id"})
		return
	}

	if err := h.service.AnalyzeMeeting(c.Request.Context(), user.ID, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: err.Error()})
			return
		}
		if strings.Contains(err.Error(), "already being analyzed") {
			c.JSON(http.StatusConflict, models.ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to start analysis"})
		return
	}

	c.JSON(http.StatusAccepted, models.SuccessResponse{Message: "analysis started"})
}

func (h *MeetingHandler) Delete(c *gin.Context) {
	user, ok := middleware.CurrentUser(c)
	if !ok || user == nil {
		c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "authentication required"})
		return
	}

	id, err := parseObjectID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{Error: "invalid meeting id"})
		return
	}

	if err := h.service.DeleteMeeting(c.Request.Context(), user.ID, id); err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, models.ErrorResponse{Error: "meeting not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{Error: "failed to delete meeting"})
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse{Message: "meeting deleted"})
}

func parseObjectID(s string) (primitive.ObjectID, error) {
	return primitive.ObjectIDFromHex(s)
}

func isLikelyTextContent(s string) bool {
	if len(s) == 0 {
		return false
	}
	sample := s
	if len(sample) > 4096 {
		sample = sample[:4096]
	}
	nullCount := 0
	controlCount := 0
	for _, r := range sample {
		if r == 0 {
			nullCount++
		} else if r < 32 && r != '\n' && r != '\r' && r != '\t' {
			controlCount++
		}
	}
	return nullCount <= len(sample)/100 && controlCount <= len(sample)/20
}
