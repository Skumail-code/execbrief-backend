package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/execbrief/backend/internal/ai"
	"github.com/execbrief/backend/internal/config"
	"github.com/execbrief/backend/internal/db"
	"github.com/execbrief/backend/internal/handlers"
	"github.com/execbrief/backend/internal/middleware"
	"github.com/execbrief/backend/internal/repository"
	"github.com/execbrief/backend/internal/services"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	log.Printf("connecting to MongoDB at %s...", cfg.MongoURI)
	mongodb, err := db.Connect(cfg.MongoURI, cfg.MongoDB)
	if err != nil {
		log.Fatalf("MongoDB connection failed: %v", err)
	}
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = mongodb.Disconnect(ctx)
	}()
	log.Println("MongoDB connected and indexes created")

	ctx := context.Background()
	gemini, err := ai.NewGeminiClient(ctx, cfg.GoogleAIAPIKey)
	if err != nil {
		log.Fatalf("Gemini client error: %v", err)
	}
	defer gemini.Close()
	log.Println("Gemini client initialized")

	meetingRepo := repository.NewMeetingRepository(mongodb.Database)
	analysisRepo := repository.NewAnalysisRepository(mongodb.Database)
	actionItemRepo := repository.NewActionItemRepository(mongodb.Database)
	decisionRepo := repository.NewDecisionRepository(mongodb.Database)
	userRepo := repository.NewUserRepository(mongodb.Database)
	sessionRepo := repository.NewSessionRepository(mongodb.Database)

	authService := services.NewAuthService(userRepo, sessionRepo)
	meetingService := services.NewMeetingService(
		meetingRepo, analysisRepo, actionItemRepo, decisionRepo, authService, gemini,
	)

	authHandler := handlers.NewAuthHandler(authService)
	meetingHandler := handlers.NewMeetingHandler(meetingService, cfg.MaxUploadMB)
	actionItemHandler := handlers.NewActionItemHandler(meetingService)

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.RequestLogger())
	r.Use(middleware.CORS(cfg.FrontendOrigin))

	r.GET("/health", handlers.Health)

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/signup", authHandler.Signup)
			auth.POST("/login", authHandler.Login)
			auth.GET("/me", middleware.RequireAuth(sessionRepo, userRepo), authHandler.Me)
			auth.PATCH("/me", middleware.RequireAuth(sessionRepo, userRepo), authHandler.UpdateMe)
			auth.PATCH("/plan", middleware.RequireAuth(sessionRepo, userRepo), authHandler.UpdatePlan)
			auth.POST("/logout", authHandler.Logout)
		}

		protected := api.Group("")
		protected.Use(middleware.RequireAuth(sessionRepo, userRepo))

		meetings := protected.Group("/meetings")
		{
			meetings.POST("/upload", meetingHandler.Upload)
			meetings.GET("", meetingHandler.List)
			meetings.GET("/:id", meetingHandler.GetDetail)
			meetings.POST("/:id/analyze", meetingHandler.Analyze)
			meetings.DELETE("/:id", meetingHandler.Delete)
		}

		protected.PUT("/action-items/:id", actionItemHandler.Update)
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%s", cfg.Port),
		Handler:      r,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("ExecBrief API running on port %s", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("server error: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	shutCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutCtx); err != nil {
		log.Printf("shutdown error: %v", err)
	}
	log.Println("server stopped")
}
