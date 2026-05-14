package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Plan string

const (
	PlanFree Plan = "free"
	PlanPro  Plan = "pro"
	PlanTeam Plan = "team"
)

type User struct {
	ID           primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	Name         string             `bson:"name" json:"name"`
	Email        string             `bson:"email" json:"email"`
	PasswordHash string             `bson:"password_hash" json:"-"`
	Plan         Plan               `bson:"plan" json:"plan"`
	Credits      int                `bson:"credits" json:"credits"`
	MaxCredits   int                `bson:"max_credits" json:"maxCredits"`
	JoinedAt     time.Time          `bson:"joined_at" json:"joinedAt"`
	UpdatedAt    time.Time          `bson:"updated_at" json:"updatedAt"`
}

type Session struct {
	ID        primitive.ObjectID `bson:"_id,omitempty" json:"id"`
	UserID    primitive.ObjectID `bson:"user_id" json:"userId"`
	TokenHash string             `bson:"token_hash" json:"-"`
	ExpiresAt time.Time          `bson:"expires_at" json:"expiresAt"`
	CreatedAt time.Time          `bson:"created_at" json:"createdAt"`
}

type SignupRequest struct {
	Name     string `json:"name"`
	Email    string `json:"email"`
	Password string `json:"password"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdateProfileRequest struct {
	Name  *string `json:"name"`
	Email *string `json:"email"`
}

type UpdatePlanRequest struct {
	Plan Plan `json:"plan"`
}

type AuthResponse struct {
	User User `json:"user"`
}
