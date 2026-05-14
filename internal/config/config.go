package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Port            string
	MongoURI        string
	MongoDB         string
	GoogleAIAPIKey  string
	MaxUploadMB     int64
	AppEnv          string
	FrontendOrigin  string
}

func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		Port:           getEnv("PORT", "8080"),
		MongoURI:       getEnv("MONGODB_URI", "mongodb://localhost:27017"),
		MongoDB:        getEnv("MONGODB_DATABASE", "execbrief"),
		GoogleAIAPIKey: getEnv("GOOGLE_AI_API_KEY", ""),
		AppEnv:         getEnv("APP_ENV", "development"),
		FrontendOrigin: getEnv("FRONTEND_ORIGIN", "http://localhost:3000"),
	}

	maxMB, err := strconv.ParseInt(getEnv("MAX_UPLOAD_MB", "10"), 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid MAX_UPLOAD_MB: %w", err)
	}
	cfg.MaxUploadMB = maxMB

	if cfg.GoogleAIAPIKey == "" {
		return nil, fmt.Errorf("GOOGLE_AI_API_KEY is required")
	}

	return cfg, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
