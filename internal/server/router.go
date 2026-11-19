package server

import (
	"encoding/json"
	"net/http"

	"app/docs"
	"app/internal/config"
	"app/internal/features/ai"
	"app/internal/features/auth"
	"app/internal/features/post"
	"app/internal/middleware"
	"app/internal/response"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/gorm"
)

func SetupRouter(cfg *config.Config, db *gorm.DB) *gin.Engine {
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(middleware.CORS())
	r.Use(middleware.RateLimiter())

	// Initialize repositories, services, handlers
	authRepo := auth.NewRepository(db)
	authSvc := auth.NewService(authRepo, cfg.AppKey)
	authHandler := auth.NewHandler(authSvc)

	postRepo := post.NewRepository(db)
	postSvc := post.NewService(postRepo)
	postHandler := post.NewHandler(postSvc)

	searchClient := ai.NewSearchClient(cfg.SearchEngineEndpoint)
	openAIClient := ai.NewOpenAIClient(cfg.OpenAIEndpoint, cfg.OpenAIKey)
	aiSvc := ai.NewService(cfg, openAIClient, searchClient)
	aiHandler := ai.NewHandler(aiSvc)

	// Swagger UI & dynamic OpenAPI JSON
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
	r.GET("/api.json", func(c *gin.Context) {
		scheme := "http"
		if c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https" {
			scheme = "https"
		}
		doc, _ := docs.GetSwaggerJSON(c.Request.Host, scheme)
		var parsed map[string]interface{}
		_ = json.Unmarshal([]byte(doc), &parsed)
		c.JSON(http.StatusOK, parsed)
	})

	// Health check / base URL: GET / membalas JSON standar
	healthHandler := func(c *gin.Context) {
		response.Success(c, http.StatusOK, "Delcom Blog API berjalan", gin.H{
			"name":    "Delcom Blog API",
			"version": "1.0.0",
		})
	}
	r.GET("/", healthHandler)

	// 404 NoRoute handler
	r.NoRoute(func(c *gin.Context) {
		response.Fail(c, http.StatusNotFound, "Endpoint not found")
	})

	// Main API Group
	api := r.Group("/api")
	{
		// 1. Public Endpoints
		public := api.Group("/public")
		{
			public.GET("/categories", postHandler.GetCategories)
			public.GET("/posts", postHandler.GetPublicPosts)
			public.GET("/posts/:slug", postHandler.GetPublicPostBySlug)
		}

		// 2. Auth Endpoints
		authGroup := api.Group("/auth")
		{
			authGroup.POST("/register", authHandler.Register)
			authGroup.POST("/login", authHandler.Login)
			authGroup.POST("/logout", authHandler.Logout)

			authProtected := authGroup.Group("")
			authProtected.Use(middleware.Auth(cfg.AppKey))
			{
				authProtected.GET("/profile", authHandler.GetProfile)
				authProtected.PUT("/profile", authHandler.UpdateProfile)
			}
		}

		// 3. Author Posts Endpoints (Protected)
		postGroup := api.Group("/posts")
		postGroup.Use(middleware.Auth(cfg.AppKey))
		{
			postGroup.GET("", postHandler.GetAuthorPosts)
			postGroup.POST("", postHandler.CreatePost)
			postGroup.GET("/stats", postHandler.GetAuthorStats)
			postGroup.POST("/ai-generate", aiHandler.GenerateDraft)
			postGroup.GET("/:id", postHandler.GetPostByID)
			postGroup.PUT("/:id", postHandler.UpdatePost)
			postGroup.PATCH("/:id/toggle-publish", postHandler.TogglePublish)
			postGroup.DELETE("/:id", postHandler.DeletePost)
		}

		// 4. Admin Endpoints (Protected + Role Admin)
		adminGroup := api.Group("/admin")
		adminGroup.Use(middleware.Auth(cfg.AppKey), middleware.RequireRole("admin"))
		{
			adminGroup.GET("/posts", postHandler.GetAdminPosts)
			adminGroup.GET("/users", postHandler.GetAdminUsers)
			adminGroup.GET("/users/:id/posts", postHandler.GetAdminUserPosts)
		}
	}

	return r
}
