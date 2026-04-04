package main

import (
	"log"

	"github.com/gin-gonic/gin"
	"github.com/timlzh/ollama-hack/internal/config"
	"github.com/timlzh/ollama-hack/internal/database"
	"github.com/timlzh/ollama-hack/internal/handlers"
	"github.com/timlzh/ollama-hack/internal/middleware"
	"github.com/timlzh/ollama-hack/internal/services"
)

func main() {
	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Connect to database
	db, err := database.Connect(&cfg.Database)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	// Create tables
	if err := db.CreateTables(); err != nil {
		log.Fatalf("Failed to create tables: %v", err)
	}

	// Start background endpoint tester
	tester := services.NewTester(db)
	tester.Start()
	defer tester.Stop()

	// Start background cleanup
	cleanupSvc := services.NewBackgroundCleanupService(db)
	cleanupSvc.Start()
	defer cleanupSvc.Stop()

	// Start background scraper
	scraperSvc := services.NewBackgroundScraperService(db)
	scraperSvc.Start()
	defer scraperSvc.Stop()

	// Initialize services
	services.InitHealthTracker(db)
	authService := services.NewAuthService(db, cfg)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, db)
	userHandler := handlers.NewUserHandler(db)
	apikeyHandler := handlers.NewAPIKeyHandler(db)
	planHandler := handlers.NewPlanHandler(db)
	endpointHandler := handlers.NewEndpointHandler(db)
	modelHandler := handlers.NewAIModelHandler(db)
	ollamaHandler := handlers.NewOllamaHandler(db)

	if cfg.App.Env == "prod" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := gin.Default()

	// CORS middleware
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// Public routes
	public := r.Group("/api/v2")
	{
		public.POST("/user/login", authHandler.Login)
		public.POST("/user/init", authHandler.InitializeAdmin)
		public.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "healthy"})
		})

		// Unprotected live stats endpoint for frontend dashboard EventSource
		public.GET("/stats/live", handlers.LiveMetrics)
	}

	// Protected routes (any authenticated user)
	protected := r.Group("/api/v2")
	protected.Use(middleware.AuthMiddleware(authService))
	{
		protected.GET("/user/me", authHandler.GetCurrentUser)
		protected.PATCH("/user/me/change-password", authHandler.ChangePassword)

		// Plans for self
		protected.GET("/plan/me", planHandler.GetCurrentUserPlan)

		// Own API keys (non-admin users manage their own)
		protected.GET("/apikey", apikeyHandler.List)
		protected.POST("/apikey", apikeyHandler.Create)
		protected.DELETE("/apikey/:id", apikeyHandler.Delete)
		protected.GET("/apikey/:id/stats", apikeyHandler.GetStats)

		// Admin-only routes
		admin := protected.Group("")
		admin.Use(middleware.AdminMiddleware())
		{
			// User management
			admin.GET("/user", userHandler.List)
			admin.POST("/user", userHandler.Create)
			admin.GET("/user/:id", userHandler.Get)
			admin.PATCH("/user/:id", userHandler.Update)
			admin.DELETE("/user/:id", userHandler.Delete)

			// Plans
			admin.GET("/plan", planHandler.List)
			admin.POST("/plan", planHandler.Create)
			admin.GET("/plan/:id", planHandler.Get)
			admin.PATCH("/plan/:id", planHandler.Update)
			admin.DELETE("/plan/:id", planHandler.Delete)

			// Endpoints
			admin.GET("/endpoint", endpointHandler.List)
			admin.POST("/endpoint", endpointHandler.Create)
			admin.POST("/endpoint/batch", endpointHandler.BatchCreate)
			admin.GET("/endpoint/:id", endpointHandler.Get)
			admin.PATCH("/endpoint/:id", endpointHandler.Update)
			admin.DELETE("/endpoint/:id", endpointHandler.Delete)
			admin.DELETE("/endpoint/batch", endpointHandler.BatchDelete)
			admin.POST("/endpoint/batch-test", endpointHandler.BatchTest)
			admin.POST("/endpoint/:id/test", endpointHandler.TriggerTest)
			admin.GET("/endpoint/:id/task", endpointHandler.GetTask)
			admin.POST("/endpoint/batch/task-status", endpointHandler.BatchGetTasks)

			// AI models
			admin.GET("/ai_model", modelHandler.List)
			admin.GET("/ai_model/:id", modelHandler.Get)
			admin.GET("/ai_model/smart/resolutions", modelHandler.SmartModels)
			admin.PATCH("/ai_model/:id/toggle", modelHandler.Toggle) // enable/disable
		}
	}

	// Ollama-compatible proxy (API-key or JWT auth)
	v1 := r.Group("/v1")
	v1.Use(middleware.AuthMiddleware(authService))
	{
		v1.GET("/models", ollamaHandler.Models)
		v1.POST("/chat/completions", ollamaHandler.ChatCompletions)
		v1.POST("/completions", ollamaHandler.Completions)
	}

	// Native Ollama API proxy endpoints
	apiNative := r.Group("/api")
	apiNative.Use(middleware.AuthMiddleware(authService))
	{
		apiNative.GET("/tags", ollamaHandler.Tags)
		apiNative.POST("/generate", ollamaHandler.Generate)
		apiNative.POST("/chat", ollamaHandler.Chat)
	}

	log.Printf("Starting server on :8000 (env: %s)", cfg.App.Env)
	if err := r.Run(":8000"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
