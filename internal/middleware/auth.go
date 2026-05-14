package middleware

import (
	"crypto/sha256"
	"encoding/hex"
	"net/http"

	"github.com/execbrief/backend/internal/models"
	"github.com/execbrief/backend/internal/repository"
	"github.com/gin-gonic/gin"
)

const (
	authCookieName = "execbrief_session"
	authUserKey    = "authUser"
)

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func CurrentUser(c *gin.Context) (*models.User, bool) {
	user, ok := c.Get(authUserKey)
	if !ok {
		return nil, false
	}
	u, ok := user.(*models.User)
	return u, ok
}

func RequireAuth(sessionRepo *repository.SessionRepository, userRepo *repository.UserRepository) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie(authCookieName)
		if err != nil || token == "" {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "authentication required"})
			c.Abort()
			return
		}

		session, err := sessionRepo.FindByTokenHash(c.Request.Context(), hashToken(token))
		if err != nil || session == nil {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "session expired or invalid"})
			c.Abort()
			return
		}

		user, err := userRepo.FindByID(c.Request.Context(), session.UserID)
		if err != nil || user == nil {
			c.JSON(http.StatusUnauthorized, models.ErrorResponse{Error: "user not found"})
			c.Abort()
			return
		}

		c.Set(authUserKey, user)
		c.Next()
	}
}

func SetAuthCookie(c *gin.Context, token string) {
	secure := c.Request.TLS != nil
	if c.GetHeader("X-Forwarded-Proto") == "https" {
		secure = true
	}
	sameSite := http.SameSiteLaxMode
	if secure {
		sameSite = http.SameSiteNoneMode
	}

	http.SetCookie(c.Writer, &http.Cookie{
		Name:     authCookieName,
		Value:    token,
		Path:     "/",
		MaxAge:   30 * 24 * 60 * 60,
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
	})
}

func ClearAuthCookie(c *gin.Context) {
	secure := c.Request.TLS != nil
	if c.GetHeader("X-Forwarded-Proto") == "https" {
		secure = true
	}
	sameSite := http.SameSiteLaxMode
	if secure {
		sameSite = http.SameSiteNoneMode
	}
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     authCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   secure,
		SameSite: sameSite,
	})
}
