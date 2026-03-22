package handler

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/masa720/resume-optimizer-backend/domain"
	"github.com/masa720/resume-optimizer-backend/service"
)

type AnalysisHandler struct {
	analysisRepo       domain.AnalysisRepository
	suggestionProvider service.SuggestionProvider
}

func NewAnalysisHandler(analysisRepo domain.AnalysisRepository, suggestionProvider service.SuggestionProvider) *AnalysisHandler {
	return &AnalysisHandler{
		analysisRepo:       analysisRepo,
		suggestionProvider: suggestionProvider,
	}
}

type createAnalysisRequest struct {
	JobDescription string `json:"jobDescription" binding:"required"`
	ResumeText     string `json:"resumeText" binding:"required"`
	CompanyName    string `json:"companyName"`
	JobPosition    string `json:"jobPosition"`
}

func (h *AnalysisHandler) Create(ctx *gin.Context) {
	userID, valid := getUserID(ctx)
	if !valid {
		return
	}

	var req createAnalysisRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	keywords := service.ExtractKeywords(req.JobDescription)
	matchedKeywords, missingKeywords := service.MatchKeywords(keywords, req.ResumeText)
	score := service.CalculateMatchScore(len(matchedKeywords), len(matchedKeywords)+len(missingKeywords))

	suggestions, err := h.suggestionProvider.Generate(
		ctx.Request.Context(),
		req.JobDescription,
		req.ResumeText,
		missingKeywords,
	)
	if err != nil {
		// FIXME
		log.Printf("suggestion provider error: %v", err)
		suggestions = service.GenerateSuggestions(missingKeywords)
	}

	suggestionsJSON, err := json.Marshal(suggestions)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate suggestions"})
		return
	}

	analysis := &domain.Analysis{
		UserID:          userID,
		JobDescription:  req.JobDescription,
		ResumeText:      req.ResumeText,
		CompanyName:     req.CompanyName,
		JobPosition:     req.JobPosition,
		MatchScore:      score,
		MatchedKeywords: matchedKeywords,
		MissingKeywords: missingKeywords,
		Suggestions:     suggestionsJSON,
	}

	if err := h.analysisRepo.Create(analysis); err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create analysis"})
		return
	}

	ctx.JSON(http.StatusCreated, analysis)
}

func (h *AnalysisHandler) List(ctx *gin.Context) {
	userID, valid := getUserID(ctx)
	if !valid {
		return
	}

	analyses, err := h.analysisRepo.GetAllByUserID(userID)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch analyses"})
		return
	}

	ctx.JSON(http.StatusOK, analyses)
}

func (h *AnalysisHandler) GetByID(ctx *gin.Context) {
	userID, valid := getUserID(ctx)
	if !valid {
		return
	}

	analysisID := ctx.Param("id")

	analysis, err := h.analysisRepo.GetByID(userID, analysisID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "analysis not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch analysis"})
		return
	}

	ctx.JSON(http.StatusOK, analysis)
}

func (h *AnalysisHandler) Delete(ctx *gin.Context) {
	userID, valid := getUserID(ctx)
	if !valid {
		return
	}

	analysisID := ctx.Param("id")

	if err := h.analysisRepo.Delete(userID, analysisID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			ctx.JSON(http.StatusNotFound, gin.H{"error": "analysis not found"})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete analysis"})
		return
	}

	ctx.Status(http.StatusNoContent)
}
