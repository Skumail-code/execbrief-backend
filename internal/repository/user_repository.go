package repository

import (
	"context"
	"strings"
	"time"

	"github.com/execbrief/backend/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type UserRepository struct {
	col *mongo.Collection
}

func NewUserRepository(db *mongo.Database) *UserRepository {
	return &UserRepository{col: db.Collection("users")}
}

func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	user.ID = primitive.NewObjectID()
	user.JoinedAt = time.Now()
	user.UpdatedAt = time.Now()
	_, err := r.col.InsertOne(ctx, user)
	return err
}

func (r *UserRepository) FindByID(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	var user models.User
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.col.FindOne(ctx, bson.M{"email": strings.ToLower(strings.TrimSpace(email))}).Decode(&user)
	if err == mongo.ErrNoDocuments {
		return nil, nil
	}
	return &user, err
}

func (r *UserRepository) UpdateFields(ctx context.Context, id primitive.ObjectID, fields bson.M) error {
	fields["updated_at"] = time.Now()
	_, err := r.col.UpdateOne(ctx, bson.M{"_id": id}, bson.M{"$set": fields})
	return err
}

func (r *UserRepository) SetPasswordHash(ctx context.Context, id primitive.ObjectID, hash string) error {
	return r.UpdateFields(ctx, id, bson.M{"password_hash": hash})
}

func (r *UserRepository) UpdateProfile(ctx context.Context, id primitive.ObjectID, name, email string) error {
	return r.UpdateFields(ctx, id, bson.M{
		"name":  strings.TrimSpace(name),
		"email": strings.ToLower(strings.TrimSpace(email)),
	})
}

func (r *UserRepository) UpdatePlan(ctx context.Context, id primitive.ObjectID, plan models.Plan, credits, maxCredits int) error {
	return r.UpdateFields(ctx, id, bson.M{
		"plan":        plan,
		"credits":     credits,
		"max_credits":  maxCredits,
	})
}

func (r *UserRepository) ConsumeCredit(ctx context.Context, id primitive.ObjectID) (*models.User, error) {
	filter := bson.M{
		"_id":     id,
		"plan":    bson.M{"$ne": models.PlanTeam},
		"credits": bson.M{"$gt": 0},
	}
	update := bson.M{"$inc": bson.M{"credits": -1}, "$set": bson.M{"updated_at": time.Now()}}
	res, err := r.col.UpdateOne(ctx, filter, update)
	if err != nil {
		return nil, err
	}
	if res.MatchedCount == 0 {
		return nil, nil
	}
	return r.FindByID(ctx, id)
}
