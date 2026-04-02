package main

import (
	"log"
	"os"

	"strings"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"github.com/masa720/resume-optimizer-backend/config"
	"github.com/masa720/resume-optimizer-backend/domain"
	"github.com/masa720/resume-optimizer-backend/handler"
	"github.com/masa720/resume-optimizer-backend/middleware"
	"github.com/masa720/resume-optimizer-backend/repository"
	"github.com/masa720/resume-optimizer-backend/service"
)

func main() {
	config.ConnectDatabase()

	if err := config.DB.AutoMigrate(&domain.Analysis{}, &domain.AnalysisVersion{}, &domain.Profile{}); err != nil {
		log.Fatal("Failed to migrate database: ", err)
	}

	analysisRepo := repository.NewAnalysisRepository(config.DB)

	var suggestionProvider service.SuggestionProvider
	openAIKey := os.Getenv("OPENAI_API_KEY")
	openAIModel := os.Getenv("OPENAI_MODEL")
	openAIBaseURL := os.Getenv("OPENAI_BASE_URL")

	var jdAnalyzer service.JDAnalyzer
	var unifiedAnalyzer service.UnifiedAnalyzer
	if openAIKey != "" {
		suggestionProvider = service.NewOpenAISuggestionProvider(openAIKey, openAIModel, openAIBaseURL)
		jdAnalyzer = service.NewOpenAIJDAnalyzer(openAIKey, openAIModel, openAIBaseURL)
		unifiedAnalyzer = service.NewOpenAIUnifiedAnalyzer(openAIKey, openAIModel, openAIBaseURL)
	} else {
		suggestionProvider = service.NewTemplateSuggestionProvider()
		jdAnalyzer = &service.FallbackJDAnalyzer{}
	}

	analysisHandler := handler.NewAnalysisHandler(analysisRepo, suggestionProvider, jdAnalyzer, unifiedAnalyzer)

	profileRepo := repository.NewProfileRepository(config.DB)
	profileHandler := handler.NewProfileHandler(profileRepo)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := gin.Default()

	corsOrigins := os.Getenv("CORS_ORIGINS")
	if corsOrigins == "" {
		corsOrigins = "*"
	}
	r.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Split(corsOrigins, ","),
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: corsOrigins != "*",
	}))

	r.GET("/hello", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"message": "Hello World!",
		})
	})

	auth := r.Group("/")
	auth.Use(middleware.AuthMiddleware())

	auth.POST("/analyses", analysisHandler.Create)
	auth.GET("/analyses", analysisHandler.List)
	auth.GET("/analyses/:id", analysisHandler.GetByID)
	auth.POST("/analyses/:id/versions", analysisHandler.CreateVersion)
	auth.PATCH("/analyses/:id/status", analysisHandler.UpdateStatus)
	auth.DELETE("/analyses/:id", analysisHandler.Delete)

	auth.GET("/profile", profileHandler.GetProfile)
	auth.PUT("/profile", profileHandler.UpdateProfile)

	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
