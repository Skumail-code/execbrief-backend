package repository

import (
	"context"
	"time"

	"github.com/execbrief/backend/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type SessionRepository struct {
	col *mongo.Collection
}

func NewSessionRepository(db *mongo.Database) *SessionRepository {
	return &SessionRepository{col: db.Collection("sessions")}
}

func (r *SessionRepository) Create(ctx context.Context, session *models.Session) error {
	session.ID = primitive.NewObjectID()
	session.CreatedAt = time.Now()
	_, err := r.col.InsertOne(ctx, session)
	return err
}

func (r *SessionRepository) FindByTokenHash(ctx context.Context, tokenHash string) (*models.Session, error) {
	var session models.Session
	err := r.col.FindOne(ctx, bson.M{"token_hash": tokenHash}).Decode(&session)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if time.Now().After(session.ExpiresAt) {
		_ = r.DeleteByTokenHash(ctx, tokenHash)
		return nil, nil
	}
	return &session, nil
}

func (r *SessionRepository) DeleteByTokenHash(ctx context.Context, tokenHash string) error {
	_, err := r.col.DeleteOne(ctx, bson.M{"token_hash": tokenHash})
	return err
}

func (r *SessionRepository) DeleteByUserID(ctx context.Context, userID primitive.ObjectID) error {
	_, err := r.col.DeleteMany(ctx, bson.M{"user_id": userID})
	return err
}
