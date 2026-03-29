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

	if err := config.DB.AutoMigrate(&domain.Analysis{}, &domain.Profile{}); err != nil {
		log.Fatal("Failed to migrate database: ", err)
	}

	analysisRepo := repository.NewAnalysisRepository(config.DB)

	var suggestionProvider service.SuggestionProvider
	openAIKey := os.Getenv("OPENAI_API_KEY")
	openAIModel := os.Getenv("OPENAI_MODEL")
	openAIBaseURL := os.Getenv("OPENAI_BASE_URL")

	if openAIKey != "" {
		suggestionProvider = service.NewOpenAISuggestionProvider(openAIKey, openAIModel, openAIBaseURL)
	} else {
		suggestionProvider = service.NewTemplateSuggestionProvider()
	}

	analysisHandler := handler.NewAnalysisHandler(analysisRepo, suggestionProvider)

	profileRepo := repository.NewProfileRepository(config.DB)
	profileHandler := handler.NewProfileHandler(profileRepo)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     strings.Split(os.Getenv("CORS_ORIGINS"), ","),
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
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
	auth.DELETE("/analyses/:id", analysisHandler.Delete)

	auth.GET("/profile", profileHandler.GetProfile)
	auth.PUT("/profile", profileHandler.UpdateProfile)

	if err := r.Run(":" + port); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
