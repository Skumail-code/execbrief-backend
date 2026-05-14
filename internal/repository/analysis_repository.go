package repository

import (
	"context"
	"time"

	"github.com/execbrief/backend/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AnalysisRepository struct {
	col *mongo.Collection
}

func NewAnalysisRepository(db *mongo.Database) *AnalysisRepository {
	return &AnalysisRepository{col: db.Collection("meeting_analysis")}
}

func (r *AnalysisRepository) Upsert(ctx context.Context, a *models.MeetingAnalysis) error {
	now := time.Now()
	a.UpdatedAt = now
	if a.ID.IsZero() {
		a.ID = primitive.NewObjectID()
		a.CreatedAt = now
	}

	filter := bson.M{"meeting_id": a.MeetingID, "user_id": a.UserID}
	update := bson.M{"$set": a}
	opts := options.Update().SetUpsert(true)
	_, err := r.col.UpdateOne(ctx, filter, update, opts)
	return err
}

func (r *AnalysisRepository) FindByMeetingID(ctx context.Context, userID, meetingID primitive.ObjectID) (*models.MeetingAnalysis, error) {
	var a models.MeetingAnalysis
	err := r.col.FindOne(ctx, bson.M{"meeting_id": meetingID, "user_id": userID}).Decode(&a)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &a, err
}

func (r *AnalysisRepository) DeleteByMeetingID(ctx context.Context, userID, meetingID primitive.ObjectID) error {
	_, err := r.col.DeleteMany(ctx, bson.M{"meeting_id": meetingID, "user_id": userID})
	return err
}
