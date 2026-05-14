package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/execbrief/backend/internal/ai"
	"github.com/execbrief/backend/internal/models"
	"github.com/execbrief/backend/internal/repository"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MeetingService struct {
	meetingRepo    *repository.MeetingRepository
	analysisRepo   *repository.AnalysisRepository
	actionItemRepo *repository.ActionItemRepository
	decisionRepo   *repository.DecisionRepository
	authService    *AuthService
	gemini         *ai.GeminiClient
}

func NewMeetingService(
	meetingRepo *repository.MeetingRepository,
	analysisRepo *repository.AnalysisRepository,
	actionItemRepo *repository.ActionItemRepository,
	decisionRepo *repository.DecisionRepository,
	authService *AuthService,
	gemini *ai.GeminiClient,
) *MeetingService {
	return &MeetingService{
		meetingRepo:    meetingRepo,
		analysisRepo:   analysisRepo,
		actionItemRepo: actionItemRepo,
		decisionRepo:   decisionRepo,
		authService:    authService,
		gemini:         gemini,
	}
}

func (s *MeetingService) CreateMeeting(ctx context.Context, userID primitive.ObjectID, title, filename, transcript string) (*models.Meeting, error) {
	cleaned := CleanTranscript(transcript)
	if len(cleaned) < 10 {
		return nil, fmt.Errorf("transcript is empty or too short after cleaning")
	}

	meeting := &models.Meeting{
		UserID:           userID,
		Title:            title,
		OriginalFilename: filename,
		TranscriptText:   cleaned,
		Status:           models.StatusUploaded,
	}

	if err := s.meetingRepo.Create(ctx, meeting); err != nil {
		return nil, fmt.Errorf("failed to save meeting: %w", err)
	}

	if _, err := s.authService.ConsumeCredit(ctx, userID); err != nil {
		_ = s.meetingRepo.DeleteByID(ctx, userID, meeting.ID)
		return nil, err
	}

	return meeting, nil
}

func (s *MeetingService) ListMeetings(ctx context.Context, userID primitive.ObjectID) ([]models.Meeting, error) {
	return s.meetingRepo.FindAll(ctx, userID)
}

func (s *MeetingService) GetMeetingDetail(ctx context.Context, userID, id primitive.ObjectID) (*models.MeetingDetailResponse, error) {
	meeting, err := s.meetingRepo.FindByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if meeting == nil {
		return nil, nil
	}

	analysis, err := s.analysisRepo.FindByMeetingID(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	actionItems, err := s.actionItemRepo.FindByMeetingID(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	decisions, err := s.decisionRepo.FindByMeetingID(ctx, userID, id)
	if err != nil {
		return nil, err
	}

	if actionItems == nil {
		actionItems = []models.ActionItem{}
	}
	if decisions == nil {
		decisions = []models.Decision{}
	}

	return &models.MeetingDetailResponse{
		Meeting:     meeting,
		Analysis:    analysis,
		ActionItems: actionItems,
		Decisions:   decisions,
	}, nil
}

func (s *MeetingService) AnalyzeMeeting(ctx context.Context, userID, id primitive.ObjectID) error {
	meeting, err := s.meetingRepo.FindByID(ctx, userID, id)
	if err != nil {
		return err
	}
	if meeting == nil {
		return fmt.Errorf("meeting not found")
	}
	if meeting.Status == models.StatusProcessing {
		return fmt.Errorf("meeting is already being analyzed")
	}

	if err := s.meetingRepo.UpdateStatus(ctx, userID, id, models.StatusProcessing); err != nil {
		return err
	}

	go s.runAnalysis(context.Background(), meeting)
	return nil
}

func userFacingError(err error) string {
	if err == nil {
		return "Unknown error during analysis"
	}
	msg := err.Error()
	switch {
	case strings.Contains(msg, "API key") || strings.Contains(msg, "PERMISSION_DENIED") || strings.Contains(msg, "api_key"):
		return "AI service authentication failed. Please check your GOOGLE_AI_API_KEY."
	case strings.Contains(msg, "deadline exceeded") || strings.Contains(msg, "timeout") || strings.Contains(msg, "context deadline"):
		return "Analysis timed out. Please try a shorter transcript."
	case strings.Contains(msg, "empty or too short"):
		return "The transcript is too short to analyze."
	case strings.Contains(msg, "SAFETY") || strings.Contains(msg, "safety"):
		return "The AI flagged the content for safety reasons. Please review your transcript."
	case strings.Contains(msg, "not found") && strings.Contains(msg, "model"):
		return "The AI model is unavailable. Please contact support."
	case strings.Contains(msg, "quota") || strings.Contains(msg, "RESOURCE_EXHAUSTED"):
		return "AI API quota exceeded. Please try again later."
	case strings.Contains(msg, "binary data") || strings.Contains(msg, "image"):
		return "The uploaded file appears to contain non-text data. Please upload a plain text transcript."
	case strings.Contains(msg, "no candidates"):
		return "The AI could not analyze this transcript. Please try a different file."
	default:
		if len(msg) > 300 {
			msg = msg[:300] + "..."
		}
		return msg
	}
}

func (s *MeetingService) runAnalysis(ctx context.Context, meeting *models.Meeting) {
	ctx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()

	// Capture panics so a crash in the goroutine doesn't kill the process
	// and the meeting status is always updated.
	defer func() {
		if r := recover(); r != nil {
			panicMsg := fmt.Sprintf("analysis goroutine panic: %v", r)
			fmt.Printf("[ERROR] %s for meeting %s\n", panicMsg, meeting.ID.Hex())
		_ = s.meetingRepo.UpdateStatusWithError(ctx, meeting.UserID, meeting.ID, models.StatusFailed, panicMsg)
		}
	}()

	fail := func(err error) {
		userMsg := userFacingError(err)
		fmt.Printf("[ERROR] analysis failed for meeting %s: %v\n", meeting.ID.Hex(), err)
		if updateErr := s.meetingRepo.UpdateStatusWithError(ctx, meeting.UserID, meeting.ID, models.StatusFailed, userMsg); updateErr != nil {
			fmt.Printf("[ERROR] failed to update meeting status: %v\n", updateErr)
		}
	}

	aiResp, rawMap, err := s.gemini.AnalyzeTranscript(ctx, meeting.TranscriptText)
	if err != nil {
		fail(err)
		return
	}

	analysis := &models.MeetingAnalysis{
		MeetingID:        meeting.ID,
		UserID:           meeting.UserID,
		NormalSummary:    aiResp.NormalSummary,
		ExecutiveSummary: aiResp.ExecutiveSummary,
		RawAIResponse:    rawMap,
	}
	if err := s.analysisRepo.Upsert(ctx, analysis); err != nil {
		fail(fmt.Errorf("save analysis: %w", err))
		return
	}

	// Clear existing items before inserting new ones
		_ = s.actionItemRepo.DeleteByMeetingID(ctx, meeting.UserID, meeting.ID)
		_ = s.decisionRepo.DeleteByMeetingID(ctx, meeting.UserID, meeting.ID)

	actionItems := make([]models.ActionItem, 0, len(aiResp.ActionItems))
	for _, a := range aiResp.ActionItems {
		status := models.ActionStatusPending
		if a.Status == string(models.ActionStatusInProgress) {
			status = models.ActionStatusInProgress
		} else if a.Status == string(models.ActionStatusCompleted) {
			status = models.ActionStatusCompleted
		}
		actionItems = append(actionItems, models.ActionItem{
			MeetingID:   meeting.ID,
			UserID:      meeting.UserID,
			Task:        a.Task,
			Owner:       a.Owner,
			DueDate:     a.DueDate,
			Priority:    a.Priority,
			Status:      status,
			SourceQuote: a.SourceQuote,
		})
	}
	if err := s.actionItemRepo.InsertMany(ctx, actionItems); err != nil {
		fail(fmt.Errorf("save action items: %w", err))
		return
	}

	decisions := make([]models.Decision, 0, len(aiResp.Decisions))
	for _, d := range aiResp.Decisions {
		decisions = append(decisions, models.Decision{
			MeetingID:   meeting.ID,
			UserID:      meeting.UserID,
			Decision:    d.Decision,
			Owner:       d.Owner,
			SourceQuote: d.SourceQuote,
		})
	}
	if err := s.decisionRepo.InsertMany(ctx, decisions); err != nil {
		fail(fmt.Errorf("save decisions: %w", err))
		return
	}

	_ = s.meetingRepo.UpdateStatus(ctx, meeting.UserID, meeting.ID, models.StatusCompleted)
}

func (s *MeetingService) UpdateActionItem(ctx context.Context, userID, id primitive.ObjectID, req *models.UpdateActionItemRequest) (*models.ActionItem, error) {
	item, err := s.actionItemRepo.FindByID(ctx, userID, id)
	if err != nil {
		return nil, err
	}
	if item == nil {
		return nil, nil
	}

	fields := bson.M{}
	if req.Task != nil {
		fields["task"] = *req.Task
	}
	if req.Owner != nil {
		fields["owner"] = *req.Owner
	}
	if req.DueDate != nil {
		fields["due_date"] = *req.DueDate
	}
	if req.Priority != nil {
		fields["priority"] = *req.Priority
	}
	if req.Status != nil {
		fields["status"] = *req.Status
	}

	if len(fields) == 0 {
		return item, nil
	}

	if err := s.actionItemRepo.Update(ctx, userID, id, fields); err != nil {
		return nil, err
	}

	return s.actionItemRepo.FindByID(ctx, userID, id)
}

func (s *MeetingService) DeleteMeeting(ctx context.Context, userID, id primitive.ObjectID) error {
	if err := s.meetingRepo.DeleteByID(ctx, userID, id); err != nil {
		return err
	}
	_ = s.analysisRepo.DeleteByMeetingID(ctx, userID, id)
	_ = s.actionItemRepo.DeleteByMeetingID(ctx, userID, id)
	_ = s.decisionRepo.DeleteByMeetingID(ctx, userID, id)
	return nil
}
