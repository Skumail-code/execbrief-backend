package repository

import (
	"context"
	"time"

	"github.com/execbrief/backend/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type DecisionRepository struct {
	col *mongo.Collection
}

func NewDecisionRepository(db *mongo.Database) *DecisionRepository {
	return &DecisionRepository{col: db.Collection("decisions")}
}

func (r *DecisionRepository) InsertMany(ctx context.Context, decisions []models.Decision) error {
	if len(decisions) == 0 {
		return nil
	}
	docs := make([]interface{}, len(decisions))
	for i := range decisions {
		decisions[i].ID = primitive.NewObjectID()
		decisions[i].CreatedAt = time.Now()
		docs[i] = decisions[i]
	}
	_, err := r.col.InsertMany(ctx, docs)
	return err
}

func (r *DecisionRepository) FindByMeetingID(ctx context.Context, userID, meetingID primitive.ObjectID) ([]models.Decision, error) {
	cursor, err := r.col.Find(ctx, bson.M{"meeting_id": meetingID, "user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var decisions []models.Decision
	if err := cursor.All(ctx, &decisions); err != nil {
		return nil, err
	}
	return decisions, nil
}

func (r *DecisionRepository) DeleteByMeetingID(ctx context.Context, userID, meetingID primitive.ObjectID) error {
	_, err := r.col.DeleteMany(ctx, bson.M{"meeting_id": meetingID, "user_id": userID})
	return err
}
