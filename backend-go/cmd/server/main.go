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

	// Initialize services
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
		public.POST("/auth/login", authHandler.Login)
		public.POST("/init", authHandler.InitializeAdmin)
		public.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "healthy"})
		})
	}

	// Protected routes (any authenticated user)
	protected := r.Group("/api/v2")
	protected.Use(middleware.AuthMiddleware(authService))
	{
		protected.GET("/auth/me", authHandler.GetCurrentUser)
		protected.PUT("/auth/password", authHandler.ChangePassword)
		// Own API keys (non-admin users manage their own)
		protected.GET("/apikeys", apikeyHandler.List)
		protected.POST("/apikeys", apikeyHandler.Create)
		protected.DELETE("/apikeys/:id", apikeyHandler.Delete)

		// Admin-only routes
		admin := protected.Group("")
		admin.Use(middleware.AdminMiddleware())
		{
			// User management
			admin.GET("/users", userHandler.List)
			admin.POST("/users", userHandler.Create)
			admin.GET("/users/:id", userHandler.Get)
			admin.PUT("/users/:id", userHandler.Update)
			admin.DELETE("/users/:id", userHandler.Delete)

			// Plans
			admin.GET("/plans", planHandler.List)
			admin.POST("/plans", planHandler.Create)
			admin.PUT("/plans/:id", planHandler.Update)
			admin.DELETE("/plans/:id", planHandler.Delete)

			// Endpoints
			admin.GET("/endpoints", endpointHandler.List)
			admin.POST("/endpoints", endpointHandler.Create)
			admin.POST("/endpoints/batch", endpointHandler.BatchCreate)
			admin.PUT("/endpoints/:id", endpointHandler.Update)
			admin.DELETE("/endpoints/:id", endpointHandler.Delete)
			admin.DELETE("/endpoints/batch", endpointHandler.BatchDelete)
			admin.POST("/endpoints/batch-test", endpointHandler.BatchTest)

			// AI models
			admin.GET("/models", modelHandler.List)
			admin.GET("/models/:id", modelHandler.Get)
			admin.PATCH("/models/:id/toggle", modelHandler.Toggle) // enable/disable
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

	log.Printf("Starting server on :8000 (env: %s)", cfg.App.Env)
	if err := r.Run(":8000"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
