package services

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/execbrief/backend/internal/models"
	"github.com/execbrief/backend/internal/repository"
	"golang.org/x/crypto/bcrypt"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

const authSessionDuration = 30 * 24 * time.Hour

type AuthService struct {
	userRepo    *repository.UserRepository
	sessionRepo *repository.SessionRepository
}

func NewAuthService(userRepo *repository.UserRepository, sessionRepo *repository.SessionRepository) *AuthService {
	return &AuthService{userRepo: userRepo, sessionRepo: sessionRepo}
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func defaultMaxCredits(plan models.Plan) int {
	switch plan {
	case models.PlanPro:
		return 100
	case models.PlanTeam:
		return 9999
	case models.PlanFree:
		return 5
	default:
		return 5
	}
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func newToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func (s *AuthService) issueSession(ctx context.Context, userID primitive.ObjectID) (string, error) {
	token, err := newToken()
	if err != nil {
		return "", err
	}
	session := &models.Session{
		UserID:    userID,
		TokenHash: hashToken(token),
		ExpiresAt: time.Now().Add(authSessionDuration),
	}
	if err := s.sessionRepo.Create(ctx, session); err != nil {
		return "", err
	}
	return token, nil
}

func (s *AuthService) Signup(ctx context.Context, name, email, password string) (*models.User, string, error) {
	name = strings.TrimSpace(name)
	email = normalizeEmail(email)
	if name == "" {
		return nil, "", fmt.Errorf("name is required")
	}
	if email == "" || !strings.Contains(email, "@") {
		return nil, "", fmt.Errorf("a valid email is required")
	}
	if len(password) < 8 {
		return nil, "", fmt.Errorf("password must be at least 8 characters")
	}

	existing, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, "", err
	}
	if existing != nil {
		return nil, "", fmt.Errorf("an account with that email already exists")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}

	user := &models.User{
		Name:         name,
		Email:        email,
		PasswordHash: string(hash),
		Plan:         models.PlanFree,
		Credits:      defaultMaxCredits(models.PlanFree),
		MaxCredits:   defaultMaxCredits(models.PlanFree),
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, "", fmt.Errorf("an account with that email already exists")
		}
		return nil, "", err
	}

	token, err := s.issueSession(ctx, user.ID)
	if err != nil {
		return nil, "", err
	}

	return user, token, nil
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*models.User, string, error) {
	email = normalizeEmail(email)
	user, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, "", err
	}
	if user == nil {
		return nil, "", fmt.Errorf("invalid email or password")
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", fmt.Errorf("invalid email or password")
	}

	token, err := s.issueSession(ctx, user.ID)
	if err != nil {
		return nil, "", err
	}
	return user, token, nil
}

func (s *AuthService) GetUserFromToken(ctx context.Context, token string) (*models.User, error) {
	session, err := s.sessionRepo.FindByTokenHash(ctx, hashToken(token))
	if err != nil {
		return nil, err
	}
	if session == nil {
		return nil, nil
	}
	return s.userRepo.FindByID(ctx, session.UserID)
}

func (s *AuthService) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.sessionRepo.DeleteByTokenHash(ctx, hashToken(token))
}

func (s *AuthService) UpdateProfile(ctx context.Context, userID primitive.ObjectID, name, email string) (*models.User, error) {
	name = strings.TrimSpace(name)
	email = normalizeEmail(email)
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}
	if email == "" || !strings.Contains(email, "@") {
		return nil, fmt.Errorf("a valid email is required")
	}

	existing, err := s.userRepo.FindByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	if existing != nil && existing.ID != userID {
		return nil, fmt.Errorf("an account with that email already exists")
	}

	if err := s.userRepo.UpdateProfile(ctx, userID, name, email); err != nil {
		if mongo.IsDuplicateKeyError(err) {
			return nil, fmt.Errorf("an account with that email already exists")
		}
		return nil, err
	}
	return s.userRepo.FindByID(ctx, userID)
}

func (s *AuthService) UpgradePlan(ctx context.Context, userID primitive.ObjectID, plan models.Plan) (*models.User, error) {
	if plan != models.PlanFree && plan != models.PlanPro && plan != models.PlanTeam {
		return nil, fmt.Errorf("invalid plan")
	}
	credits := defaultMaxCredits(plan)
	if err := s.userRepo.UpdatePlan(ctx, userID, plan, credits, credits); err != nil {
		return nil, err
	}
	return s.userRepo.FindByID(ctx, userID)
}

func (s *AuthService) ConsumeCredit(ctx context.Context, userID primitive.ObjectID) (*models.User, error) {
	current, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	if current == nil {
		return nil, fmt.Errorf("user not found")
	}
	if current.Plan == models.PlanTeam {
		return current, nil
	}

	user, err := s.userRepo.ConsumeCredit(ctx, userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, fmt.Errorf("not enough credits")
	}
	return user, nil
}
