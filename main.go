package main

import (
	"log"

	"github.com/gin-gonic/gin"

	"github.com/your-username/resume-optimizer-backend/config"
	"github.com/your-username/resume-optimizer-backend/domain"
	"github.com/your-username/resume-optimizer-backend/handler"
	"github.com/your-username/resume-optimizer-backend/repository"
)

func main() {
	config.ConnectDatabase()

	if err := config.DB.AutoMigrate(&domain.Analysis{}, &domain.Profile{}); err != nil {
		log.Fatal("Failed to migrate database: ", err)
	}

	analysisRepo := repository.NewAnalysisRepository(config.DB)

	analysisHandler := handler.NewAnalysisHandler(analysisRepo)

	r := gin.Default()

	r.GET("/hello", func(ctx *gin.Context) {
		ctx.JSON(200, gin.H{
			"message": "Hello World!",
		})
	})

	r.POST("/analyses", analysisHandler.Create)
	r.GET("/analyses", analysisHandler.List)
	r.GET("/analyses/:id", analysisHandler.GetByID)

	if err := r.Run(":8080"); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}
