package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type MeetingStatus string

const (
	StatusUploaded   MeetingStatus = "uploaded"
	StatusProcessing MeetingStatus = "processing"
	StatusCompleted  MeetingStatus = "completed"
	StatusFailed     MeetingStatus = "failed"
)

type ActionItemPriority string

const (
	PriorityLow    ActionItemPriority = "low"
	PriorityMedium ActionItemPriority = "medium"
	PriorityHigh   ActionItemPriority = "high"
)

type ActionItemStatus string

const (
	ActionStatusPending    ActionItemStatus = "pending"
	ActionStatusInProgress ActionItemStatus = "in_progress"
	ActionStatusCompleted  ActionItemStatus = "completed"
)

type Meeting struct {
	ID               primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID           primitive.ObjectID `bson:"user_id" json:"-"`
	Title            string             `bson:"title" json:"title"`
	OriginalFilename string             `bson:"original_filename" json:"original_filename"`
	TranscriptText   string             `bson:"transcript_text" json:"transcript_text,omitempty"`
	Status           MeetingStatus      `bson:"status" json:"status"`
	ErrorMessage     string             `bson:"error_message" json:"error_message,omitempty"`
	CreatedAt        time.Time          `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time          `bson:"updated_at" json:"updated_at"`
}

type ExecutiveSummary struct {
	Accomplishments       []string `bson:"accomplishments" json:"accomplishments"`
	Blockers              []string `bson:"blockers" json:"blockers"`
	PendingTasks          []string `bson:"pending_tasks" json:"pending_tasks"`
	Risks                 []string `bson:"risks" json:"risks"`
	RecommendedCEOActions []string `bson:"recommended_ceo_actions" json:"recommended_ceo_actions"`
}

type MeetingAnalysis struct {
	ID               primitive.ObjectID     `bson:"_id,omitempty" json:"id"`
	MeetingID        primitive.ObjectID     `bson:"meeting_id" json:"meeting_id"`
	UserID           primitive.ObjectID     `bson:"user_id" json:"-"`
	NormalSummary    string                 `bson:"normal_summary" json:"normal_summary"`
	ExecutiveSummary ExecutiveSummary       `bson:"executive_summary" json:"executive_summary"`
	RawAIResponse    map[string]interface{} `bson:"raw_ai_response" json:"raw_ai_response,omitempty"`
	CreatedAt        time.Time              `bson:"created_at" json:"created_at"`
	UpdatedAt        time.Time              `bson:"updated_at" json:"updated_at"`
}

type ActionItem struct {
	ID          primitive.ObjectID  `bson:"_id,omitempty" json:"id"`
	MeetingID   primitive.ObjectID  `bson:"meeting_id" json:"meeting_id"`
	UserID      primitive.ObjectID  `bson:"user_id" json:"-"`
	Task        string              `bson:"task" json:"task"`
	Owner       *string             `bson:"owner" json:"owner"`
	DueDate     *string             `bson:"due_date" json:"due_date"`
	Priority    *ActionItemPriority `bson:"priority" json:"priority"`
	Status      ActionItemStatus    `bson:"status" json:"status"`
	SourceQuote *string             `bson:"source_quote" json:"source_quote"`
	CreatedAt   time.Time           `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time           `bson:"updated_at" json:"updated_at"`
}

type Decision struct {
	ID          primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	MeetingID   primitive.ObjectID `bson:"meeting_id" json:"meeting_id"`
	UserID      primitive.ObjectID `bson:"user_id" json:"-"`
	Decision    string             `bson:"decision" json:"decision"`
	Owner       *string            `bson:"owner" json:"owner"`
	SourceQuote *string            `bson:"source_quote" json:"source_quote"`
	CreatedAt   time.Time          `bson:"created_at" json:"created_at"`
}

// AI response structures

type AIActionItem struct {
	Task        string              `json:"task"`
	Owner       *string             `json:"owner"`
	DueDate     *string             `json:"due_date"`
	Priority    *ActionItemPriority `json:"priority"`
	Status      string              `json:"status"`
	SourceQuote *string             `json:"source_quote"`
}

type AIDecision struct {
	Decision    string  `json:"decision"`
	Owner       *string `json:"owner"`
	SourceQuote *string `json:"source_quote"`
}

type AIResponse struct {
	NormalSummary    string           `json:"normal_summary"`
	ExecutiveSummary ExecutiveSummary `json:"executive_summary"`
	ActionItems      []AIActionItem   `json:"action_items"`
	Decisions        []AIDecision     `json:"decisions"`
}

// Request/Response types

type UpdateActionItemRequest struct {
	Task     *string             `json:"task"`
	Owner    *string             `json:"owner"`
	DueDate  *string             `json:"due_date"`
	Priority *ActionItemPriority `json:"priority"`
	Status   *ActionItemStatus   `json:"status"`
}

type MeetingDetailResponse struct {
	Meeting     *Meeting         `json:"meeting"`
	Analysis    *MeetingAnalysis `json:"analysis"`
	ActionItems []ActionItem     `json:"action_items"`
	Decisions   []Decision       `json:"decisions"`
}

type ErrorResponse struct {
	Error string `json:"error"`
}

type SuccessResponse struct {
	Message string `json:"message"`
}

type UploadResponse struct {
	MeetingID string `json:"meeting_id"`
	Title     string `json:"title"`
	Status    string `json:"status"`
}

type HealthResponse struct {
	Status    string `json:"status"`
	Service   string `json:"service"`
	Timestamp string `json:"timestamp"`
}
