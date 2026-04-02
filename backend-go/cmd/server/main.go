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

	// Initialize services
	authService := services.NewAuthService(db, cfg)

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(authService, db)

	// Setup Gin
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

	// Protected routes
	protected := r.Group("/api/v2")
	protected.Use(middleware.AuthMiddleware(authService))
	{
		protected.GET("/auth/me", authHandler.GetCurrentUser)
		protected.PUT("/auth/password", authHandler.ChangePassword)

		// User management (admin only)
		admin := protected.Group("")
		admin.Use(middleware.AdminMiddleware())
		{
			// Add user CRUD routes here
			// admin.GET("/users", userHandler.List)
			// admin.POST("/users", userHandler.Create)
			// admin.GET("/users/:id", userHandler.Get)
			// admin.PUT("/users/:id", userHandler.Update)
			// admin.DELETE("/users/:id", userHandler.Delete)

			// Add API key routes here
			// admin.GET("/apikeys", apikeyHandler.List)
			// admin.POST("/apikeys", apikeyHandler.Create)
			// admin.DELETE("/apikeys/:id", apikeyHandler.Delete)

			// Add plan routes here
			// admin.GET("/plans", planHandler.List)
			// admin.POST("/plans", planHandler.Create)
			// admin.PUT("/plans/:id", planHandler.Update)
			// admin.DELETE("/plans/:id", planHandler.Delete)

			// Add endpoint routes here
			// admin.GET("/endpoints", endpointHandler.List)
			// admin.POST("/endpoints", endpointHandler.Create)
			// admin.POST("/endpoints/batch", endpointHandler.BatchCreate)
			// admin.PUT("/endpoints/:id", endpointHandler.Update)
			// admin.DELETE("/endpoints/:id", endpointHandler.Delete)
			// admin.DELETE("/endpoints/batch", endpointHandler.BatchDelete)
			// admin.POST("/endpoints/batch-test", endpointHandler.BatchTest)

			// Add AI model routes here
			// admin.GET("/models", modelHandler.List)
			// admin.GET("/models/:id", modelHandler.Get)
		}
	}

	// Ollama proxy routes (with API key auth)
	v1 := r.Group("/v1")
	v1.Use(middleware.AuthMiddleware(authService))
	{
		// Add Ollama proxy routes here
		// v1.POST("/chat/completions", ollamaHandler.ChatCompletions)
		// v1.POST("/completions", ollamaHandler.Completions)
		// v1.GET("/models", ollamaHandler.Models)
	}

	log.Printf("Starting server on :8000 (env: %s)", cfg.App.Env)
	if err := r.Run(":8000"); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
