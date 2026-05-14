package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/execbrief/backend/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MeetingRepository struct {
	col *mongo.Collection
}

func NewMeetingRepository(db *mongo.Database) *MeetingRepository {
	return &MeetingRepository{col: db.Collection("meetings")}
}

func (r *MeetingRepository) Create(ctx context.Context, m *models.Meeting) error {
	m.ID = primitive.NewObjectID()
	m.CreatedAt = time.Now()
	m.UpdatedAt = time.Now()
	_, err := r.col.InsertOne(ctx, m)
	return err
}

func (r *MeetingRepository) FindByID(ctx context.Context, userID, id primitive.ObjectID) (*models.Meeting, error) {
	var m models.Meeting
	err := r.col.FindOne(ctx, bson.M{"_id": id, "user_id": userID}).Decode(&m)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &m, err
}

func (r *MeetingRepository) FindAll(ctx context.Context, userID primitive.ObjectID) ([]models.Meeting, error) {
	opts := options.Find().SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetProjection(bson.M{"transcript_text": 0})

	cursor, err := r.col.Find(ctx, bson.M{"user_id": userID}, opts)
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var meetings []models.Meeting
	if err := cursor.All(ctx, &meetings); err != nil {
		return nil, err
	}
	return meetings, nil
}

func (r *MeetingRepository) UpdateStatus(ctx context.Context, userID, id primitive.ObjectID, status models.MeetingStatus) error {
	_, err := r.col.UpdateOne(ctx,
		bson.M{"_id": id, "user_id": userID},
		bson.M{"$set": bson.M{"status": status, "updated_at": time.Now()}},
	)
	return err
}

func (r *MeetingRepository) UpdateStatusWithError(ctx context.Context, userID, id primitive.ObjectID, status models.MeetingStatus, errorMessage string) error {
	_, err := r.col.UpdateOne(ctx,
		bson.M{"_id": id, "user_id": userID},
		bson.M{"$set": bson.M{"status": status, "error_message": errorMessage, "updated_at": time.Now()}},
	)
	return err
}

func (r *MeetingRepository) DeleteByID(ctx context.Context, userID, id primitive.ObjectID) error {
	res, err := r.col.DeleteOne(ctx, bson.M{"_id": id, "user_id": userID})
	if err != nil {
		return err
	}
	if res.DeletedCount == 0 {
		return fmt.Errorf("meeting not found")
	}
	return nil
}
