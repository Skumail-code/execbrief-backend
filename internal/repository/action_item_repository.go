package repository

import (
	"context"
	"time"

	"github.com/execbrief/backend/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type ActionItemRepository struct {
	col *mongo.Collection
}

func NewActionItemRepository(db *mongo.Database) *ActionItemRepository {
	return &ActionItemRepository{col: db.Collection("action_items")}
}

func (r *ActionItemRepository) InsertMany(ctx context.Context, items []models.ActionItem) error {
	if len(items) == 0 {
		return nil
	}
	docs := make([]interface{}, len(items))
	for i := range items {
		items[i].ID = primitive.NewObjectID()
		items[i].CreatedAt = time.Now()
		items[i].UpdatedAt = time.Now()
		docs[i] = items[i]
	}
	_, err := r.col.InsertMany(ctx, docs)
	return err
}

func (r *ActionItemRepository) FindByMeetingID(ctx context.Context, userID, meetingID primitive.ObjectID) ([]models.ActionItem, error) {
	cursor, err := r.col.Find(ctx, bson.M{"meeting_id": meetingID, "user_id": userID})
	if err != nil {
		return nil, err
	}
	defer cursor.Close(ctx)

	var items []models.ActionItem
	if err := cursor.All(ctx, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (r *ActionItemRepository) FindByID(ctx context.Context, userID, id primitive.ObjectID) (*models.ActionItem, error) {
	var item models.ActionItem
	err := r.col.FindOne(ctx, bson.M{"_id": id, "user_id": userID}).Decode(&item)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &item, err
}

func (r *ActionItemRepository) Update(ctx context.Context, userID, id primitive.ObjectID, fields bson.M) error {
	fields["updated_at"] = time.Now()
	_, err := r.col.UpdateOne(ctx, bson.M{"_id": id, "user_id": userID}, bson.M{"$set": fields})
	return err
}

func (r *ActionItemRepository) DeleteByMeetingID(ctx context.Context, userID, meetingID primitive.ObjectID) error {
	_, err := r.col.DeleteMany(ctx, bson.M{"meeting_id": meetingID, "user_id": userID})
	return err
}
